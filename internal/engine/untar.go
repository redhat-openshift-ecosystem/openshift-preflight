package engine

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
)

// untar takes a destination path, a container image, and a list of files or match patterns
// which should be extracted out of the image.
func untar(ctx context.Context, dst string, img v1.Image, requiredFilePatterns []string) error {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(log.DBG).Info("exporting and flattening image")

	// Extract all files matching the required file patterns.
	state := make(map[string]bool)
	var err error

	logger.V(log.DBG).Info("extracting container filesystem", "path", dst)

	remaining := slices.Clone(requiredFilePatterns)

	// In the case of symlinks, the targets may not be included in the original required file
	// patterns, so make additional passes through the layers as needed to find them.
	// Make at least one pass to validate the tar format, even if there are no required patterns.
	for {
		if remaining, err = untarOnce(ctx, dst, img, remaining, state); err != nil {
			return fmt.Errorf("failed to extract tarball: %w", err)
		}
		if len(remaining) == 0 {
			break
		}
	}

	return nil
}

// untarOnce takes a destination path, a container image, a list of files or match patterns
// which should be extracted out of the image, and a map in which to store extraction progress/state.
// The function returns a list of files that should be extracted in another invocation of
// untarOnce along with an error if one was encountered.
// A tar reader loops over the tarfile creating the file structure at
// 'dst' along the way, and writing any files. This function uses a pre-allocated buffer to
// reduce allocations and is not goroutine-safe.
// Uses os.Root to restrict extraction to dst.
func untarOnce(ctx context.Context, dst string, img v1.Image, filterPatterns []string, state map[string]bool) (remaining []string, err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(log.TRC).Info("extracting from tar stream with filter patterns", "patterns", filterPatterns)

	fs := mutate.Extract(img)
	defer func() {
		// Drain any remaining data from the reader and capture any errors
		_, drainErr := io.Copy(io.Discard, fs)
		if drainErr != nil {
			err = fmt.Errorf("failed to drain io reader: %w", drainErr)
		}
		fs.Close()
	}()

	filesProcessedInThisPass := make(map[string]bool)
	unresolvedLinkTargets := make(map[string]bool)

	tr := tar.NewReader(fs)
	dst = filepath.Clean(dst)
	dstRoot, openErr := os.OpenRoot(dst)
	if openErr != nil {
		return slices.Collect(maps.Keys(unresolvedLinkTargets)), fmt.Errorf("untar error, unable to open extraction destination %s: %w", dst, openErr)
	}
	defer dstRoot.Close()

	// Buffer for io.CopyBuffer operations to reduce allocations
	buf := make([]byte, 32*1024)
	for {
		header, err := tr.Next()

		switch {
		// if no more files are found return
		case err == io.EOF:
			logger.V(log.TRC).Info("extracted files", "files", filesProcessedInThisPass)
			logger.V(log.TRC).Info("remaining files", "files", unresolvedLinkTargets)
			return slices.Collect(maps.Keys(unresolvedLinkTargets)), nil

		// return any other error
		case err != nil:
			logger.V(log.TRC).Info("extracted files", "files", filesProcessedInThisPass)
			logger.V(log.TRC).Info("remaining files", "files", unresolvedLinkTargets)
			return slices.Collect(maps.Keys(unresolvedLinkTargets)), err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		if state[header.Name] {
			continue
		}

		matches := slices.ContainsFunc(filterPatterns, func(p string) bool {
			result, _ := filepath.Match(p, header.Name)
			return result
		})
		if !matches {
			continue
		}

		// check the file type
		switch header.Typeflag {
		// skip all directories, we'll only create the needed directory
		// structure for the files/symlinks that need to be created
		case tar.TypeDir:
			continue

		// if it's a file create it
		case tar.TypeReg:
			dirname := filepath.Dir(header.Name)
			if err := dstRoot.MkdirAll(dirname, 0o755); err != nil {
				return slices.Collect(maps.Keys(unresolvedLinkTargets)), err
			}

			// Mask non-permission bits, which are not supported by dstRoot.OpenFile
			fileMode := os.FileMode(header.Mode & 0o777)
			f, err := dstRoot.OpenFile(header.Name, os.O_CREATE|os.O_WRONLY, fileMode)
			if err != nil {
				return slices.Collect(maps.Keys(unresolvedLinkTargets)), err
			}

			// copy over contents
			if _, err := io.CopyBuffer(f, tr, buf); err != nil {
				f.Close()
				return slices.Collect(maps.Keys(unresolvedLinkTargets)), err
			}

			filesProcessedInThisPass[header.Name] = true
			state[header.Name] = true

			// If the file being processed is the target of a symlink we encountered earlier in
			// this pass, it will also be in unresolvedLinkTargets. Now that we've found it,
			// remove it from the list.
			delete(unresolvedLinkTargets, header.Name)

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()

		// if it's a link create it
		case tar.TypeSymlink, tar.TypeLink:
			// Create the new link's directory if it doesn't exist.
			dirname := filepath.Dir(header.Name)
			if err := dstRoot.MkdirAll(dirname, 0o755); err != nil {
				return slices.Collect(maps.Keys(unresolvedLinkTargets)), err
			}

			linkFn := dstRoot.Link
			if header.Typeflag == tar.TypeSymlink {
				// for dstRoot, basepath enforcement is not done on
				// oldname when symlinking, so we'll do it here instead.
				linkFn = func(oldname, newname string) error {
					// resolved the oldname relative to the new name
					resolvedON, _ := resolveLinkPaths(oldname, newname)
					// Identify extraction root traversal with post resolution
					finalOldname := filepath.Join(dstRoot.Name(), resolvedON)
					if finalOldname != dstRoot.Name() && !strings.HasPrefix(finalOldname, dstRoot.Name()+string(os.PathSeparator)) {
						return errors.New("link resolves to path outside of extraction root")
					}

					// otherwise, link the two. newname validation is done by dstRoot.
					return dstRoot.Symlink(finalOldname, newname)
				}
			}

			err := linkFn(header.Linkname, header.Name)
			if err != nil {
				logger.V(log.DBG).Info("error creating link, ignoring", "link", header.Name, "linkedTo", header.Linkname, "type", header.Typeflag, "reason", err.Error())
				continue
			}

			filesProcessedInThisPass[header.Name] = true
			state[header.Name] = true

			// If the file being processed is the target of a symlink we encountered earlier in
			// this pass, it will also be in unresolvedLinkTargets. Now that we've found it,
			// remove it from the list.
			delete(unresolvedLinkTargets, header.Name)

			resolvedTargetName := filepath.Clean(filepath.Join(filepath.Dir(header.Name), header.Linkname))

			// If the target of the symlink has already been processed (in this pass or an
			// earlier pass), then no further action is needed.
			if state[resolvedTargetName] {
				continue
			}

			// If the target of the symlink is not already in the list of search patterns,
			// add it to the list. We might get lucky and encounter it later in this pass.
			if !slices.Contains(filterPatterns, resolvedTargetName) {
				logger.V(log.TRC).Info("adding to the filter patterns for the current pass", "target", resolvedTargetName)
				filterPatterns = append(filterPatterns, resolvedTargetName)
			}

			// Also add the target of this symlink to the list of unresolved link targets,
			// if not already present. It's possible that we've already passed the target
			// on this pass, in which case this can only be resolved on another pass.
			unresolvedLinkTargets[resolvedTargetName] = true
		}
	}
}

// resolveLinkPaths determines if oldname is an absolute path or a relative
// path, and returns oldname relative to newname if necessary.
func resolveLinkPaths(oldname, newname string) (string, string) {
	if filepath.IsAbs(oldname) {
		return oldname, newname
	}

	linkDir := filepath.Dir(newname)
	// If the newname is at the root of the filesystem, but the oldname is
	// relative, we'll swap out the value we get from filepath.Dir for a / to
	// allow relative pathing to resolve. This strips `..` references given the
	// link exists at the very base of the filesystem. In effect, it converts
	// oldname to an absolute path
	if linkDir == "." {
		linkDir = "/"
	}

	return filepath.Join(linkDir, oldname), newname
}
