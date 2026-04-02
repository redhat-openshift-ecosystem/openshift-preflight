package engine

import (
	"archive/tar"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-logr/logr"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
)

// linkType a convenience type just to make the consuming functions more clear.
type linkType byte

const (
	hardlink linkType = tar.TypeLink
	symlink  linkType = tar.TypeSymlink
)

// String returns the string representation of the LinkType.
func (lt linkType) String() string {
	if lt == tar.TypeLink {
		return "hardlink"
	}
	return "symlink"
}

type linkNode struct {
	Name              string
	Deps              *linkNode
	OriginalLinkname  string   // For symlinks, the original target string before resolution
	Type              linkType // Type of link (symlink or hardlink)
	VirtualLinkTarget string   // For hardlinks pointing to symlinks, the symlink's target string
}

func (n *linkNode) IsHardlink() bool {
	return n.Type == hardlink
}

func (n *linkNode) IsSymlink() bool {
	return n.Type == symlink
}

// ChainTypesToFollow returns the link types that should be followed in a chain
// starting from this node. Hardlinks follow both hardlink and symlink chains,
// while symlinks only follow symlink chains.
func (n *linkNode) ChainTypesToFollow() []linkType {
	if n.IsHardlink() {
		return []linkType{hardlink, symlink}
	}
	return []linkType{symlink}
}

type LinkGraph map[string]*linkNode

type extractionContext struct {
	linkGraph   LinkGraph
	neededFiles *[]string
	logger      logr.Logger
}

// ExpandFilePathAliases generates all possible paths to a file through directory symlinks.
// For example, for a symlink /a/b/c -> /foo/bar (directory), and a file /foo/bar/baz, this
// will return both /a/b/c/baz and /foo/bar/baz.
func (lg LinkGraph) ExpandFilePathAliases(filePath string, symlinkAliases map[string][]string) []string {
	expanding := make(map[string]struct{})
	return lg.expandFilePathAliasesRec(filePath, symlinkAliases, expanding)
}

func (lg LinkGraph) expandFilePathAliasesRec(filePath string, symlinkAliases map[string][]string, expanding map[string]struct{}) []string {
	results := []string{filePath}
	visited := make(map[string]struct{})
	visited[filePath] = struct{}{}

	expanding[filePath] = struct{}{}
	defer delete(expanding, filePath)

	// Check all parent directories to see if they have symlink aliases
	dir := filePath
	for {
		dir = filepath.Dir(dir)
		if dir == "." || dir == "/" {
			break
		}

		// Check if this directory has any symlinks pointing to it
		if aliases, ok := symlinkAliases[dir]; ok {
			for _, symlinkPath := range aliases {
				// Replace the directory portion with the symlink path
				// For example: /usr/share/rpm/file.db with symlink /usr/lib/sysimage/rpm -> /usr/share/rpm
				// becomes /usr/lib/sysimage/rpm/file.db
				relativePath, err := filepath.Rel(dir, filePath)
				if err != nil {
					continue
				}
				aliasedPath := filepath.Join(symlinkPath, relativePath)
				if _, seen := visited[aliasedPath]; !seen {
					visited[aliasedPath] = struct{}{}
					results = append(results, aliasedPath)
					// Recursively find aliases of the aliased path. Skip recursion if aliasedPath
					// is already being expanded (directory symlink cycles).
					if _, onStack := expanding[aliasedPath]; !onStack {
						edgeKey := dir + "\x00" + symlinkPath
						if _, edgeSeen := expanding[edgeKey]; edgeSeen {
							continue
						}
						expanding[edgeKey] = struct{}{}
						nestedAliases := lg.expandFilePathAliasesRec(aliasedPath, symlinkAliases, expanding)
						delete(expanding, edgeKey)
						for _, nested := range nestedAliases {
							if _, seen := visited[nested]; !seen {
								visited[nested] = struct{}{}
								results = append(results, nested)
							}
						}
					}
				}
			}
		}
	}

	return results
}

// walkGraphChain walks a graph chain starting from a node, applying a visitor function
// to each node in the chain. Returns when the chain ends, visitor returns false, or a cycle is detected.
func walkGraphChain(start string, graph LinkGraph, visitor func(node string, deps *linkNode) bool) {
	current := start
	visited := make(map[string]struct{})
	for {
		if _, seen := visited[current]; seen {
			break
		}
		visited[current] = struct{}{}

		node, ok := graph[current]
		if !ok || node.Deps == nil {
			break
		}
		next := node.Deps.Name
		if _, seen := visited[next]; seen {
			break
		}
		if !visitor(next, node.Deps) {
			break
		}
		current = next
	}
}

// followLinkChain adds all links in the chain starting from the given link.
// For example, if linkA -> linkB -> directory, this adds both linkA and linkB
// to neededFiles. If filterType is provided (non-nil), only links of that type are followed.
func (ec *extractionContext) followLinkChain(startLink string, filterType *linkType) {
	walkGraphChain(startLink, ec.linkGraph, func(target string, deps *linkNode) bool {
		// If the target is also a link in the graph
		if targetNode, isTargetLink := ec.linkGraph[target]; isTargetLink {
			// If filtering by type, check if target matches
			if filterType != nil && targetNode.Type != *filterType {
				return false
			}
			*ec.neededFiles = append(*ec.neededFiles, target)
			linkTypeStr := targetNode.Type.String()
			ec.logger.V(log.TRC).Info("adding transitive directory "+linkTypeStr, linkTypeStr, target, "via", startLink)
			return true
		}
		return false
	})
}

// processLink processes a link (symlink or hardlink) by adding it to neededFiles,
// following its chain, and recursively processing its target and parents.
// The logContext parameter provides context for logging (e.g., "parent directory", "target").
func (ec *extractionContext) processLink(path string, node *linkNode, logContext string, originalFile string, visited map[string]struct{}) {
	*ec.neededFiles = append(*ec.neededFiles, path)

	ec.logger.V(log.TRC).Info("adding "+logContext+" "+node.Type.String(), node.Type.String(), path, "for_file", originalFile)

	// Follow link chains based on node type
	for _, chainType := range node.ChainTypesToFollow() {
		ec.followLinkChain(path, &chainType)
	}

	// Recursively process the target's parent directories AND the target itself
	// (the target might also be a link)
	if node.Deps != nil {
		depName := node.Deps.Name

		// First check if the target itself is a link
		if _, seen := visited[depName]; !seen {
			visited[depName] = struct{}{}
			if depNode, isLink := ec.linkGraph[depName]; isLink {
				ec.processLink(depName, depNode, "target", originalFile, visited)
			}
		}

		// Then check the target's parents
		ec.addParentLinks(depName, originalFile, visited)
	}
}

func (ec *extractionContext) addParentLinks(path string, originalFile string, visited map[string]struct{}) {
	// Check all parent directories up to root
	dir := filepath.Dir(path)
	for dir != "." && dir != "/" {
		if _, seen := visited[dir]; seen {
			break
		}
		visited[dir] = struct{}{}

		// Check if this directory is a link (symlink or hardlink)
		if node, isLink := ec.linkGraph[dir]; isLink {
			ec.processLink(dir, node, "parent directory", originalFile, visited)
		}

		dir = filepath.Dir(dir)
	}
}

// resolveRelativeLinkFrom resolves a symlink's target string the same way as planExtraction:
// POSIX-style absolute linknames (leading '/') are archive-root paths; relative linknames
// are resolved from linkPath's directory.
func resolveRelativeLinkFrom(linkPath, linkTarget string) string {
	if strings.HasPrefix(linkTarget, "/") {
		return filepath.Clean(strings.TrimPrefix(linkTarget, "/"))
	}
	return filepath.Clean(filepath.Join(filepath.Dir(linkPath), linkTarget))
}

// addAliasIfNew adds an alias to the aliases map if it doesn't already exist
// Returns true if the alias was added (indicating a change)
func addAliasIfNew(aliases map[string][]string, target, alias string) bool {
	if !slices.Contains(aliases[target], alias) {
		aliases[target] = append(aliases[target], alias)
		return true
	}
	return false
}

// getLinkTarget returns the link target (original linkname) for a path,
// checking both real symlinks and virtual links
func getLinkTarget(path string, linkGraph LinkGraph) (linkTarget string, found bool) {
	if linkNode, exists := linkGraph[path]; exists {
		if linkNode.IsSymlink() && linkNode.OriginalLinkname != "" {
			return linkNode.OriginalLinkname, true
		}
		if linkNode.VirtualLinkTarget != "" {
			return linkNode.VirtualLinkTarget, true
		}
	}
	return "", false
}

// getEffectiveLinkTarget checks if a path is a symlink/virtual-symlink,
// or if it's a hardlink pointing to a symlink/virtual-symlink.
// Returns the ultimate symlink target string.
func getEffectiveLinkTarget(path string, linkGraph LinkGraph) (linkTarget string, found bool) {
	// First check if path itself is a symlink/virtual-symlink
	if target, ok := getLinkTarget(path, linkGraph); ok {
		return target, true
	}

	// If path is a hardlink, check if its target is a symlink/virtual-symlink
	if node, exists := linkGraph[path]; exists && node.IsHardlink() && node.Deps != nil {
		return getLinkTarget(node.Deps.Name, linkGraph)
	}

	return "", false
}

// BuildDirectoryAliasMap builds a backlink map for all links in the
// LinkGraph. The resulting map associates each link in the graph
// with its aliases.
func (lg LinkGraph) BuildDirectoryAliasMap(logger logr.Logger) map[string][]string {
	// Start with basic symlink aliases
	aliases := make(map[string][]string)
	for linkPath, node := range lg {
		if node.IsSymlink() && node.Deps != nil {
			targetPath := node.Deps.Name
			aliases[targetPath] = append(aliases[targetPath], linkPath)
		}
	}

	iteration := 0
	// Monotonic fixpoint: each successful addAliasIfNew strictly increases the
	// multiset of (target, alias) pairs; possible pairs are finite for a finite graph.
	for {
		changed := false
		iteration++

		for linkPath, linkNode := range lg {
			if linkNode.Deps == nil {
				continue
			}
			targetPath := linkNode.Deps.Name

			// Handle hardlinks pointing to symlinks (real or virtual)
			if linkNode.IsHardlink() {
				// Skip if we've already processed this hardlink as a virtual symlink
				if linkNode.VirtualLinkTarget != "" {
					continue
				}

				// Skip if the target is not a symlink/virtual-symlink
				linkTarget, found := getLinkTarget(targetPath, lg)
				if !found {
					continue
				}

				// Create a virtual symlink at the hardlink's location: interpret the
				// target symlink's linkname as relative to this hardlink's directory
				// (same inode semantics users see when opening the path via the hardlink).
				resolvedFromHardlink := resolveRelativeLinkFrom(linkPath, linkTarget)

				if addAliasIfNew(aliases, resolvedFromHardlink, linkPath) {
					linkNode.VirtualLinkTarget = linkTarget
					logger.V(log.TRC).Info("adding hardlink alias",
						"from", linkPath,
						"to", resolvedFromHardlink,
						"via_target", targetPath,
						"iteration", iteration)
					changed = true
				}
			}

			// Handle symlinks pointing to other links (directly or via hardlink)
			if linkNode.IsSymlink() && linkNode.OriginalLinkname != "" {
				linkTarget, found := getEffectiveLinkTarget(targetPath, lg)
				if !found {
					continue
				}

				resolvedFromSymlink := resolveRelativeLinkFrom(targetPath, linkTarget)

				if addAliasIfNew(aliases, resolvedFromSymlink, linkPath) {
					logger.V(log.TRC).Info("adding symlink alias",
						"from", linkPath,
						"to", resolvedFromSymlink,
						"via_target", targetPath,
						"iteration", iteration)
					changed = true
				}
			}
		}

		if !changed {
			break
		}
	}

	logger.V(log.DBG).Info("directory alias map built", "iterations", iteration, "total_aliases", len(aliases))

	return aliases
}
