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

// finishTarReadCloser drains and closes a tar layer reader, merging drain/close errors into err.
func finishTarReadCloser(rc io.ReadCloser, err *error) {
	_, drainErr := io.Copy(io.Discard, rc)
	closeErr := rc.Close()

	var errs []error
	if *err != nil {
		errs = append(errs, *err)
	}
	if drainErr != nil {
		errs = append(errs, fmt.Errorf("failed to drain io reader: %w", drainErr))
	}
	if closeErr != nil {
		//coverage:ignore
		errs = append(errs, fmt.Errorf("failed to close io reader: %w", closeErr))
	}
	switch len(errs) {
	case 0:
	case 1:
		*err = errs[0]
	default:
		//coverage:ignore
		*err = errors.Join(errs...)
	}
}

func validateRequiredFilePatterns(patterns []string) error {
	for _, p := range patterns {
		if !doublestar.ValidatePattern(p) {
			return fmt.Errorf("invalid glob pattern: %q", p)
		}
	}
	return nil
}

func pathMatchesAnyValidatedPattern(path string, patterns []string) bool {
	return slices.ContainsFunc(patterns, func(p string) bool {
		return doublestar.MatchUnvalidated(p, path)
	})
}

// buildExtractionPlan decides which archive paths must be extracted given validated glob patterns.
func buildExtractionPlan(
	linkGraph LinkGraph,
	allFiles map[string]struct{},
	requiredFilePatterns []string,
	logger logr.Logger,
) []string {
	// Build an alias map: canonical targets -> symlink paths that point at them.
	linkAliases := linkGraph.BuildDirectoryAliasMap(logger)

	neededFiles := make([]string, 0)
	extractCtx := &extractionContext{
		linkGraph:   linkGraph,
		neededFiles: &neededFiles,
		logger:      logger,
	}

	// Some paths match only via symlink aliases (e.g. pattern under /usr/lib/sysimage/rpm
	// while the inode lives under /usr/share/rpm). When no ancestor is a symlink target,
	// ExpandFilePathAliases yields only the canonical path, so we skip the expensive expansion.
	for filePath := range allFiles {
		var aliasedPaths []string
		if pathMayHaveSymlinkAliasPaths(filePath, linkAliases) {
			aliasedPaths = linkGraph.ExpandFilePathAliases(filePath, linkAliases)
		} else {
			aliasedPaths = []string{filePath}
		}

		for _, aliasPath := range aliasedPaths {
			if !pathMatchesAnyValidatedPattern(aliasPath, requiredFilePatterns) {
				continue
			}
			neededFiles = append(neededFiles, filePath)
			visited := make(map[string]struct{})
			extractCtx.addParentLinks(aliasPath, filePath, visited)
			for _, sym := range linkAliases[filePath] {
				if _, ok := visited[sym]; ok {
					//coverage:ignore
					continue
				}
				visited[sym] = struct{}{}
				if node, ok := linkGraph[sym]; ok {
					extractCtx.processLink(sym, node, "symlink to extracted file", filePath, visited)
				}
			}
			break
		}
	}

	filesToExtract := make(map[string]struct{})
	// neededFiles lists matched archive paths plus link parents; addTransitiveDependencies
	// closes over hardlink/symlink targets so the plan is self-consistent.
	addTransitiveDependencies(neededFiles, linkGraph, allFiles, filesToExtract)

	return slices.Collect(maps.Keys(filesToExtract))
}

func planExtraction(ctx context.Context, img v1.Image, requiredFilePatterns []string) (result []string, links LinkGraph, err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(log.TRC).Info("planning extraction from tar stream")

	if err := validateRequiredFilePatterns(requiredFilePatterns); err != nil {
		return nil, nil, fmt.Errorf("invalid required file patterns: %w", err)
	}

	fs := mutate.Extract(img)
	defer func() {
		finishTarReadCloser(fs, &err)
	}()

	tr := tar.NewReader(fs)
	linkGraph := make(LinkGraph, 0)

	// Track all files that exist in the tar archive (as a set for quick lookup)
	allFiles := make(map[string]struct{})

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			result = buildExtractionPlan(linkGraph, allFiles, requiredFilePatterns, logger)
			return result, linkGraph, nil
		case err != nil:
			//coverage:ignore
			return nil, nil, err
		case header == nil:
			//coverage:ignore
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			//coverage:ignore
			continue
		case tar.TypeReg:
			allFiles[header.Name] = struct{}{}
		case tar.TypeSymlink:
			allFiles[header.Name] = struct{}{}
			resolvedTargetName := resolveRelativeLinkFrom(header.Name, header.Linkname)
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
func sortHardlinksByDependencies(hardlinks []string, graph LinkGraph) ([]string, error) {
	if len(hardlinks) == 0 {
		return hardlinks, nil
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

	if len(sorted) != len(hardlinks) {
		sortedSet := make(map[string]struct{}, len(sorted))
		for _, s := range sorted {
			//coverage:ignore
			sortedSet[s] = struct{}{}
		}
		var remaining []string
		for _, hl := range hardlinks {
			if _, ok := sortedSet[hl]; !ok {
				remaining = append(remaining, hl)
			}
		}
		slices.Sort(remaining)
		return nil, fmt.Errorf("cyclic hardlink dependency; could not order deferred hardlinks (remaining: %v)", remaining)
	}

	return sorted, nil
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
		finishTarReadCloser(fs, &err)
	}()

	extractedFiles := make(map[string]struct{})
	var deferredHardlinks []string

	tr := tar.NewReader(fs)
	dst = filepath.Clean(dst)
	dstRoot, openErr := os.OpenRoot(dst)
	if openErr != nil {
		//coverage:ignore
		return fmt.Errorf("untar error, unable to open extraction destination %s: %w", dst, openErr)
	}
	defer dstRoot.Close()

	mkdirDone := make(map[string]struct{})
	mkdirOnce := func(dir string) error {
		if dir == "." || dir == "/" {
			return nil
		}
		if _, ok := mkdirDone[dir]; ok {
			return nil
		}
		if err := dstRoot.MkdirAll(dir, 0o755); err != nil && !os.IsExist(err) {
			return err
		}
		mkdirDone[dir] = struct{}{}
		return nil
	}

	// Buffer for io.CopyBuffer operations to reduce allocations
	buf := make([]byte, 32*1024)
	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			// At this point, all regular files and symlinks have been created, along with
			// some hardlinks. Now create deferred hardlinks in dependency order.
			orderedHardlinks, sortErr := sortHardlinksByDependencies(deferredHardlinks, linkGraph)
			if sortErr != nil {
				//coverage:ignore
				return fmt.Errorf("deferred hardlink ordering: %w", sortErr)
			}

			for _, hlName := range orderedHardlinks {
				// Look up the hardlink target from the graph
				node, ok := linkGraph[hlName]
				if !ok || node.Deps == nil {
					//coverage:ignore
					logger.V(log.DBG).Info("skipping deferred hardlink, no graph entry", "link", hlName)
					continue
				}
				linkTarget := node.Deps.Name

				if _, targetExists := extractedFiles[linkTarget]; !targetExists {
					//coverage:ignore
					logger.V(log.DBG).Info("skipping deferred hardlink, target not extracted", "link", hlName, "target", linkTarget)
					continue
				}

				dirname := filepath.Dir(hlName)
				if err := mkdirOnce(dirname); err != nil {
					//coverage:ignore
					return err
				}

				if err := dstRoot.Link(linkTarget, hlName); err != nil {
					//coverage:ignore
					logger.V(log.DBG).Info("error creating deferred hardlink, ignoring", "link", hlName, "linkedTo", linkTarget, "reason", err.Error())
					continue
				}

				extractedFiles[hlName] = struct{}{}
			}

			logger.V(log.TRC).Info("extracted files", "files", extractedFiles, "count", len(extractedFiles))
			return nil

		case err != nil:
			//coverage:ignore
			logger.V(log.TRC).Info("error reading tar stream", "extractedCount", len(extractedFiles))
			return err

		case header == nil:
			//coverage:ignore
			continue
		}

		// Skip files not in the extraction plan
		if _, shouldExtract := filesToExtract[header.Name]; !shouldExtract {
			continue
		}

		// check the file type
		switch header.Typeflag {
		case tar.TypeDir:
			//coverage:ignore
			continue

		case tar.TypeReg:
			dirname := filepath.Dir(header.Name)
			if err := mkdirOnce(dirname); err != nil {
				return err
			}

			// Mask non-permission bits, which are not supported by dstRoot.OpenFile
			fileMode := os.FileMode(header.Mode & 0o777)
			f, err := dstRoot.OpenFile(header.Name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
			if err != nil {
				//coverage:ignore
				return err
			}

			_, copyErr := io.CopyBuffer(f, tr, buf)
			closeErr := f.Close()
			if copyErr != nil {
				//coverage:ignore
				return copyErr
			}
			if closeErr != nil {
				//coverage:ignore
				return closeErr
			}

			extractedFiles[header.Name] = struct{}{}

		case tar.TypeSymlink:
			// Symlinks can be created even if target doesn't exist
			dirname := filepath.Dir(header.Name)
			if err := mkdirOnce(dirname); err != nil {
				//coverage:ignore
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
				// Canonical relative path from the symlink's directory to the vetted target (POSIX
				// resolves relative symlink targets from the link's parent directory, not dst root).
				linkDirAbs := filepath.Join(dstRoot.Name(), filepath.Dir(header.Name))
				cleanFinal := filepath.Clean(finalOldname)
				relTarget, relErr := filepath.Rel(linkDirAbs, cleanFinal)
				if relErr != nil {
					//coverage:ignore
					logger.V(log.DBG).Info("symlink relative path failed, ignoring", "link", header.Name, "target", header.Linkname, "reason", relErr.Error())
					continue
				}
				if filepath.Clean(filepath.Join(linkDirAbs, relTarget)) != cleanFinal {
					//coverage:ignore
					logger.V(log.DBG).Info("symlink sanitized target mismatch, ignoring", "link", header.Name, "target", header.Linkname)
					continue
				}
				symlinkTarget = relTarget
			}

			if err := dstRoot.Symlink(symlinkTarget, header.Name); err != nil {
				//coverage:ignore
				logger.V(log.DBG).Info("error creating symlink, ignoring", "link", header.Name, "linkedTo", header.Linkname, "reason", err.Error())
				continue
			}

			extractedFiles[header.Name] = struct{}{}

		case tar.TypeLink:
			// Hardlinks require the target to exist
			// Check if target has been extracted already
			if _, targetExists := extractedFiles[header.Linkname]; targetExists {
				dirname := filepath.Dir(header.Name)
				if err := mkdirOnce(dirname); err != nil {
					//coverage:ignore
					return err
				}

				if err := dstRoot.Link(header.Linkname, header.Name); err != nil {
					//coverage:ignore
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
