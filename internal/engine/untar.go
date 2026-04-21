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

	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-logr/logr"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
)

// expandLiteralPatternsWithDescendantGlob adds a "path/**" pattern for each pattern that has no
// glob metacharacters, so a follow-up pass that only lists a symlink's target directory (e.g.
// usr/share/licenses) still matches nested files in the tarball.
//
// Patterns containing `[` are skipped: we treat * ? [ as glob syntax; literal `[` in a path is rare.
func expandLiteralPatternsWithDescendantGlob(patterns []string) []string {
	if len(patterns) == 0 {
		return patterns
	}
	out := slices.Clone(patterns)
	for _, p := range patterns {
		if p == "" || strings.ContainsAny(p, "*?[") {
			continue
		}
		child := p + "/**"
		if !slices.Contains(out, child) {
			out = append(out, child)
		}
	}
	return out
}

// clearUnresolvedLinkTargetsForExtractedPath removes entries from unresolved when an extracted
// path satisfies them: either an exact key match, or the longest key that is a strict parent
// directory prefix of extractedPath (directory symlink targets such as usr/share/licenses).
func clearUnresolvedLinkTargetsForExtractedPath(unresolved map[string]struct{}, extractedPath string) {
	var longestKey string
	for k := range unresolved {
		if extractedPath == k || strings.HasPrefix(extractedPath, k+"/") {
			if len(k) > len(longestKey) {
				longestKey = k
			}
		}
	}
	if longestKey != "" {
		delete(unresolved, longestKey)
	}
}

// appendSymlinkTargetPatterns appends the resolved symlink target and target/** to filterPatterns
// when missing, so layer paths under the real directory (e.g. usr/share/licenses/...) match.
func appendSymlinkTargetPatterns(filterPatterns []string, logger logr.Logger, resolvedTargetName string) []string {
	nestedGlob := resolvedTargetName + "/**"
	if !slices.Contains(filterPatterns, resolvedTargetName) {
		logger.V(log.TRC).Info("adding symlink target path to filter patterns", "target", resolvedTargetName)
		filterPatterns = append(filterPatterns, resolvedTargetName)
	}
	if !slices.Contains(filterPatterns, nestedGlob) {
		logger.V(log.TRC).Info("adding symlink target descendant glob to filter patterns", "targetGlob", nestedGlob)
		filterPatterns = append(filterPatterns, nestedGlob)
	}
	return filterPatterns
}

// untar takes a destination path, a container image, and a list of files or match patterns
// which should be extracted out of the image.
func untar(ctx context.Context, dst string, img v1.Image, requiredFilePatterns []string) error {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(log.DBG).Info("exporting and flattening image")

	// Extract all files matching the required file patterns.
	state := make(map[string]struct{})
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
func untarOnce(ctx context.Context, dst string, img v1.Image, filterPatterns []string, state map[string]struct{}) (remaining []string, err error) {
	logger := logr.FromContextOrDiscard(ctx)
	filterPatterns = expandLiteralPatternsWithDescendantGlob(filterPatterns)
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

	filesProcessedInThisPass := make(map[string]struct{})
	unresolvedLinkTargets := make(map[string]struct{})

	tr := tar.NewReader(fs)
	dst = filepath.Clean(dst)
	dstRoot, openErr := os.OpenRoot(dst)
	if openErr != nil {
		//coverage:ignore
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
			//coverage:ignore
			logger.V(log.TRC).Info("extracted files", "files", filesProcessedInThisPass)
			logger.V(log.TRC).Info("remaining files", "files", unresolvedLinkTargets)
			return slices.Collect(maps.Keys(unresolvedLinkTargets)), err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			//coverage:ignore
			continue
		}

		if _, ok := state[header.Name]; ok {
			continue
		}

		matches := slices.ContainsFunc(filterPatterns, func(p string) bool {
			result, _ := doublestar.Match(p, header.Name)
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
			//coverage:ignore
			continue

		// if it's a file create it
		case tar.TypeReg:
			dirname := filepath.Dir(header.Name)
			if err := dstRoot.MkdirAll(dirname, 0o755); err != nil && !os.IsExist(err) {
				return slices.Collect(maps.Keys(unresolvedLinkTargets)), err
			}

			// Mask non-permission bits, which are not supported by dstRoot.OpenFile
			fileMode := os.FileMode(header.Mode & 0o777)
			f, err := dstRoot.OpenFile(header.Name, os.O_CREATE|os.O_WRONLY, fileMode)
			if err != nil {
				//coverage:ignore
				return slices.Collect(maps.Keys(unresolvedLinkTargets)), err
			}

			// copy over contents
			if _, err := io.CopyBuffer(f, tr, buf); err != nil {
				//coverage:ignore
				f.Close()
				return slices.Collect(maps.Keys(unresolvedLinkTargets)), err
			}

			filesProcessedInThisPass[header.Name] = struct{}{}
			state[header.Name] = struct{}{}

			// Drop satisfied unresolved symlink targets (exact path or longest parent-directory key).
			clearUnresolvedLinkTargetsForExtractedPath(unresolvedLinkTargets, header.Name)

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()

		// if it's a link create it
		case tar.TypeSymlink, tar.TypeLink:
			// Create the new link's directory if it doesn't exist.
			dirname := filepath.Dir(header.Name)
			if err := dstRoot.MkdirAll(dirname, 0o755); err != nil && !os.IsExist(err) {
				//coverage:ignore
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
						//coverage:ignore
						return errors.New("link resolves to path outside of extraction root")
					}

					// otherwise, link the two. newname validation is done by dstRoot.
					return dstRoot.Symlink(finalOldname, newname)
				}
			}

			err := linkFn(header.Linkname, header.Name)
			if err != nil {
				//coverage:ignore
				logger.V(log.DBG).Info("error creating link, ignoring", "link", header.Name, "linkedTo", header.Linkname, "type", header.Typeflag, "reason", err.Error())
				continue
			}

			filesProcessedInThisPass[header.Name] = struct{}{}
			state[header.Name] = struct{}{}

			clearUnresolvedLinkTargetsForExtractedPath(unresolvedLinkTargets, header.Name)

			resolvedTargetName := filepath.Clean(filepath.Join(filepath.Dir(header.Name), header.Linkname))

			// If the target of the symlink has already been processed (in this pass or an
			// earlier pass), then no further action is needed.
			if _, ok := state[resolvedTargetName]; ok {
				continue
			}

			// Add the resolved target and target/** so nested layer paths match (e.g. /licenses ->
			// /usr/share/licenses with files stored as usr/share/licenses/...).
			filterPatterns = appendSymlinkTargetPatterns(filterPatterns, logger, resolvedTargetName)

			// Also add the target of this symlink to the list of unresolved link targets,
			// if not already present. It's possible that we've already passed the target
			// on this pass, in which case this can only be resolved on another pass.
			unresolvedLinkTargets[resolvedTargetName] = struct{}{}
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
