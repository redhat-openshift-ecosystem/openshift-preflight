package engine

import (
	"archive/tar"
	"context"
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

func planExtraction(ctx context.Context, img v1.Image, requiredFilePatterns []string) (result []string, links LinkGraph, err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(log.TRC).Info("planning extraction from tar stream")

	fs := mutate.Extract(img)
	defer func() {
		// Drain any remaining data from the reader and capture any errors
		_, drainErr := io.Copy(io.Discard, fs)
		if drainErr != nil {
			err = fmt.Errorf("failed to drain io reader: %w", drainErr)
		}
		fs.Close()
	}()

	tr := tar.NewReader(fs)
	linkGraph := make(LinkGraph, 0)

	neededFiles := make([]string, 0)
	// Track all files that exist in the tar archive (as a set for quick lookup)
	allFiles := make(map[string]struct{})

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			// Build an alias map. This map lists all of the different ways
			// that a path can be reached using a combination of hardlinks or
			// symlinks present within the tar stream.
			linkAliases := linkGraph.BuildDirectoryAliasMap(logger)

			extractCtx := &extractionContext{
				linkGraph:   linkGraph,
				neededFiles: &neededFiles,
				logger:      logger,
			}

			// Now take the list of all tar entries. Some may match the filter pattern
			// using their canonical path (as recorded in the tar archive). However, it's
			// possible for files not to match the filter pattern using their canonical
			// path while there is still a chain of links that does lead to the file and
			// matches the filter pattern. Check if any files match patterns through symlink
			// aliases: this handles the case where /usr/share/rpm/rpmdb.sqlite should be
			// extracted because /usr/lib/sysimage/rpm -> /usr/share/rpm and the pattern matches
			// /usr/lib/sysimage/rpm/rpmdb.sqlite.
			for filePath := range allFiles {
				// Generate all possible paths to this file through links
				aliasedPaths := linkGraph.ExpandFilePathAliases(filePath, linkAliases)

				// For each possible path to this file through links, check if it
				// matches the filter patterns.
				for _, aliasPath := range aliasedPaths {
					matches := slices.ContainsFunc(requiredFilePatterns, func(p string) bool {
						result, _ := filepath.Match(p, aliasPath)
						return result
					})
					if matches {
						// If there is a match, we need to extract the original file,
						// as well as any symlinks/hardlinks that need to be present
						// to resolve the file at the matching path.
						neededFiles = append(neededFiles, filePath)
						visited := make(map[string]struct{})
						extractCtx.addParentLinks(aliasPath, filePath, visited)
						break
					}
				}
			}

			// Build set of files to extract
			filesToExtract := make(map[string]struct{})

			// Now neededFiles contains a list of paths that
			addTransitiveDependencies(neededFiles, linkGraph, allFiles, filesToExtract)

			// Convert set to slice for the extraction plan
			// Order doesn't matter because hardlinks are deferred in untarOnce
			result = append(result, slices.Collect(maps.Keys(filesToExtract))...)

			return result, linkGraph, nil
		case err != nil:
			return nil, nil, err
		case header == nil:
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			allFiles[header.Name] = struct{}{}
		case tar.TypeSymlink:
			allFiles[header.Name] = struct{}{}
			resolvedTargetName := filepath.Clean(filepath.Join(filepath.Dir(header.Name), header.Linkname))
			dep, ok := linkGraph[resolvedTargetName]
			if !ok {
				dep = &linkNode{
					Name: resolvedTargetName,
					Deps: nil,
				}
			}
			linkGraph[header.Name] = &linkNode{
				Name:             header.Name,
				Deps:             dep,
				OriginalLinkname: header.Linkname,
				Type:             tar.TypeSymlink,
			}
		case tar.TypeLink:
			allFiles[header.Name] = struct{}{}
			dep, ok := linkGraph[header.Linkname]
			if !ok {
				dep = &linkNode{
					Name: header.Linkname,
					Deps: nil,
				}
			}
			linkGraph[header.Name] = &linkNode{
				Name: header.Name,
				Deps: dep,
				Type: tar.TypeLink,
			}
		}
	}
}

// untar takes a destination path, a container image, and a list of files or match patterns
// which should be extracted out of the image.
func untar(ctx context.Context, dst string, img v1.Image, requiredFilePatterns []string) error {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(log.DBG).Info("exporting and flattening image")

	files, linkGraph, err := planExtraction(ctx, img, requiredFilePatterns)
	if err != nil {
		return fmt.Errorf("failed to build extraction plan: %w", err)
	}
	logger.V(log.DBG).Info("built list of files for extraction", "files", files, "count", len(files))

	logger.V(log.DBG).Info("extracting container filesystem", "path", dst)

	if err = runExtraction(ctx, dst, img, files, linkGraph); err != nil {
		return fmt.Errorf("failed to extract tarball: %w", err)
	}

	return nil
}

// sortHardlinksByDependencies sorts deferred hardlinks so that targets are created before
// hardlinks that depend on them. Uses the link graph to determine dependencies.
func sortHardlinksByDependencies(hardlinks []string, graph LinkGraph) []string {
	if len(hardlinks) == 0 {
		return hardlinks
	}

	// Build a set of hardlink names for quick lookup
	hardlinkSet := make(map[string]struct{})
	for _, hlName := range hardlinks {
		hardlinkSet[hlName] = struct{}{}
	}

	// Build adjacency list and in-degree map for hardlinks only
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	for _, hlName := range hardlinks {
		inDegree[hlName] = 0
		adjList[hlName] = []string{}
	}

	// Build dependency graph
	for _, hlName := range hardlinks {
		if node, ok := graph[hlName]; ok && node.Deps != nil {
			target := node.Deps.Name
			// Only add edge if target is also a deferred hardlink
			if _, isHardlink := hardlinkSet[target]; isHardlink {
				adjList[target] = append(adjList[target], hlName)
				inDegree[hlName]++
			}
		}
	}

	// Topological sort using Kahn's algorithm
	queue := []string{}
	for _, hlName := range hardlinks {
		if inDegree[hlName] == 0 {
			queue = append(queue, hlName)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		for _, neighbor := range adjList[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	return sorted
}

// addTransitiveDependencies follows the graph forwards from the given files and adds
// files that the given files link to (potentially recursively).
func addTransitiveDependencies(neededFiles []string, graph LinkGraph, allFiles map[string]struct{}, filesToExtract map[string]struct{}) {
	// neededFiles and graph may contain virtual links (i.e. paths which do not
	// exist in the tar stream, but match the filter patterns and would exist if
	// the tar stream were fully extracted). The filesToExtract map should
	// only contain real paths present in the tar stream, so filter out virtual
	// paths before adding to the map.

	visited := make(map[string]struct{})
	for _, f := range neededFiles {
		if _, seen := visited[f]; seen {
			continue
		}
		visited[f] = struct{}{}

		if _, exists := allFiles[f]; exists {
			filesToExtract[f] = struct{}{}
		}

		// Consider what files would need to be recursively extracted in order
		// for the matching file from neededFiles to resolve properly (not be
		// a broken link)
		walkGraphChain(f, graph, func(target string, deps *linkNode) bool {
			if _, seen := visited[target]; seen {
				return false
			}
			visited[target] = struct{}{}

			if _, exists := allFiles[target]; exists {
				filesToExtract[target] = struct{}{}
			}
			return true
		})
	}
}

// runExtraction extracts the specified files from the container image.
// This function makes a single pass through the tar stream and handles hardlinks by
// deferring their creation until after their targets are extracted.
// The linkGraph is used to process deferred hardlinks in dependency order.
// Uses os.Root to restrict extraction to dst.
func runExtraction(ctx context.Context, dst string, img v1.Image, files []string, linkGraph LinkGraph) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(log.TRC).Info("extracting from tar stream")

	// Build a set of all files to extract from the plan
	filesToExtract := make(map[string]struct{})
	for _, file := range files {
		filesToExtract[file] = struct{}{}
	}

	fs := mutate.Extract(img)
	defer func() {
		// Drain any remaining data from the reader and capture any errors
		_, drainErr := io.Copy(io.Discard, fs)
		if drainErr != nil {
			err = fmt.Errorf("failed to drain io reader: %w", drainErr)
		}
		fs.Close()
	}()

	extractedFiles := make(map[string]struct{})
	var deferredHardlinks []string

	tr := tar.NewReader(fs)
	dst = filepath.Clean(dst)
	dstRoot, openErr := os.OpenRoot(dst)
	if openErr != nil {
		return fmt.Errorf("untar error, unable to open extraction destination %s: %w", dst, openErr)
	}
	defer dstRoot.Close()

	// Buffer for io.CopyBuffer operations to reduce allocations
	buf := make([]byte, 32*1024)
	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			// At this point, all regular files and symlinks have been created, along with
			// some hardlinks. Now create deferred hardlinks in dependency order.
			orderedHardlinks := sortHardlinksByDependencies(deferredHardlinks, linkGraph)

			for _, hlName := range orderedHardlinks {
				// Look up the hardlink target from the graph
				node, ok := linkGraph[hlName]
				if !ok || node.Deps == nil {
					logger.V(log.DBG).Info("skipping deferred hardlink, no graph entry", "link", hlName)
					continue
				}
				linkTarget := node.Deps.Name

				if _, targetExists := extractedFiles[linkTarget]; !targetExists {
					logger.V(log.DBG).Info("skipping deferred hardlink, target not extracted", "link", hlName, "target", linkTarget)
					continue
				}

				dirname := filepath.Dir(hlName)
				if err := dstRoot.MkdirAll(dirname, 0o755); err != nil && !os.IsExist(err) {
					return err
				}

				if err := dstRoot.Link(linkTarget, hlName); err != nil {
					logger.V(log.DBG).Info("error creating deferred hardlink, ignoring", "link", hlName, "linkedTo", linkTarget, "reason", err.Error())
					continue
				}

				extractedFiles[hlName] = struct{}{}
			}

			logger.V(log.TRC).Info("extracted files", "files", extractedFiles, "count", len(extractedFiles))
			return nil

		case err != nil:
			logger.V(log.TRC).Info("error reading tar stream", "extractedCount", len(extractedFiles))
			return err

		case header == nil:
			continue
		}

		// Skip files not in the extraction plan
		if _, shouldExtract := filesToExtract[header.Name]; !shouldExtract {
			continue
		}

		// check the file type
		switch header.Typeflag {
		case tar.TypeDir:
			continue

		case tar.TypeReg:
			dirname := filepath.Dir(header.Name)
			if err := dstRoot.MkdirAll(dirname, 0o755); err != nil && !os.IsExist(err) {
				return err
			}

			// Mask non-permission bits, which are not supported by dstRoot.OpenFile
			fileMode := os.FileMode(header.Mode & 0o777)
			f, err := dstRoot.OpenFile(header.Name, os.O_CREATE|os.O_WRONLY, fileMode)
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.CopyBuffer(f, tr, buf); err != nil {
				f.Close()
				return err
			}

			extractedFiles[header.Name] = struct{}{}
			f.Close()

		case tar.TypeSymlink:
			// Symlinks can be created even if target doesn't exist
			dirname := filepath.Dir(header.Name)
			if err := dstRoot.MkdirAll(dirname, 0o755); err != nil && !os.IsExist(err) {
				return err
			}

			// for dstRoot, basepath enforcement is not done on
			// oldname when symlinking, so we'll do it here instead.
			resolvedON, _ := resolveLinkPaths(header.Linkname, header.Name)
			// Identify extraction root traversal with post resolution
			finalOldname := filepath.Join(dstRoot.Name(), resolvedON)
			if finalOldname != dstRoot.Name() && !strings.HasPrefix(finalOldname, dstRoot.Name()+string(os.PathSeparator)) {
				logger.V(log.DBG).Info("symlink resolves outside extraction root, ignoring", "link", header.Name, "target", header.Linkname)
				continue
			}

			// A symlink may be reachable via one or more hardlinks. The relative target is resolved
			// differently depending on which hardlink path is accessed.
			var symlinkTarget string
			if filepath.IsAbs(header.Linkname) {
				// Absolute symlinks are converted to absolute within the extraction root
				symlinkTarget = finalOldname
			} else {
				// Relative symlinks are kept relative (they're already security-checked above)
				symlinkTarget = header.Linkname
			}

			if err := dstRoot.Symlink(symlinkTarget, header.Name); err != nil {
				logger.V(log.DBG).Info("error creating symlink, ignoring", "link", header.Name, "linkedTo", header.Linkname, "reason", err.Error())
				continue
			}

			extractedFiles[header.Name] = struct{}{}

		case tar.TypeLink:
			// Hardlinks require the target to exist
			// Check if target has been extracted already
			if _, targetExists := extractedFiles[header.Linkname]; targetExists {
				dirname := filepath.Dir(header.Name)
				if err := dstRoot.MkdirAll(dirname, 0o755); err != nil && !os.IsExist(err) {
					return err
				}

				if err := dstRoot.Link(header.Linkname, header.Name); err != nil {
					logger.V(log.DBG).Info("error creating hardlink, ignoring", "link", header.Name, "linkedTo", header.Linkname, "reason", err.Error())
					continue
				}

				extractedFiles[header.Name] = struct{}{}
			} else {
				// Defer this hardlink until after we've seen all files
				deferredHardlinks = append(deferredHardlinks, header.Name)
			}
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
