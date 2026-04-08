package engine

import (
	"fmt"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LinkGraph ExpandFilePathAliases", func() {
	It("terminates with a finite set when directory symlink aliases form a cycle", func() {
		// Mutual directory symlinks: link "b" points at directory "a", link "a" points at "b".
		// Matches BuildDirectoryAliasMap: aliases[target] = symlink paths pointing to target.
		symlinkAliases := map[string][]string{
			"a": {"b"},
			"b": {"a"},
		}
		var lg LinkGraph

		out := lg.ExpandFilePathAliases("a/x", symlinkAliases)

		Expect(out).To(ConsistOf("a/x", "b/x"))
	})

	It("normalizes absolute OriginalLinkname to archive-relative alias keys", func() {
		// Mirrors tar symlinks whose Linkname starts with '/' (planExtraction strips the slash).
		shareDir := &linkNode{Name: "data/share"}
		inner := &linkNode{Name: "link/inner", Deps: shareDir, OriginalLinkname: "/data/share", Type: symlink}
		lg := LinkGraph{
			"data/share": shareDir,
			"link/inner": inner,
			"alias/out":  &linkNode{Name: "alias/out", Deps: inner, OriginalLinkname: "link/inner", Type: symlink},
		}

		aliases := lg.BuildDirectoryAliasMap(logr.Discard())

		Expect(aliases["data/share"]).To(ContainElements("link/inner", "alias/out"))
		expanded := lg.ExpandFilePathAliases("data/share/doc", aliases)
		Expect(expanded).To(ContainElement("alias/out/doc"))
	})

	It("maps chained symlink aliases using the inner link target directory", func() {
		// Outer symlink at other/deep/here -> ../../foo/bar/baz; inner at foo/bar/baz has linkname "..".
		// Resolving ".." from the outer symlink's directory would incorrectly yield "other";
		// resolution must be relative to the inner symlink path (foo/bar/baz) so ".." means "foo".
		fooLeaf := &linkNode{Name: "foo"}
		baz := &linkNode{Name: "foo/bar/baz", Deps: fooLeaf, OriginalLinkname: "..", Type: symlink}
		lg := LinkGraph{
			"foo":             fooLeaf,
			"foo/bar/baz":     baz,
			"other/deep/here": &linkNode{Name: "other/deep/here", Deps: baz, OriginalLinkname: "../../foo/bar/baz", Type: symlink},
		}

		aliases := lg.BuildDirectoryAliasMap(logr.Discard())

		Expect(aliases["foo"]).To(ContainElements("foo/bar/baz", "other/deep/here"))
	})

	It("converges for a deep symlink chain", func() {
		const depth = 100
		lg := LinkGraph{
			"chain/0": &linkNode{Name: "chain/0"},
		}
		for i := 1; i <= depth; i++ {
			prev := fmt.Sprintf("chain/%d", i-1)
			cur := fmt.Sprintf("chain/%d", i)
			// Sibling-style linknames under chain/ so resolveRelativeLinkFrom(chain/i, name)
			// yields chain/{name} (Join("chain", "../0") would incorrectly clean to "0").
			lg[cur] = &linkNode{
				Name:             cur,
				Deps:             &linkNode{Name: prev},
				OriginalLinkname: fmt.Sprintf("%d", i-1),
				Type:             symlink,
			}
		}

		aliases := lg.BuildDirectoryAliasMap(logr.Discard())

		// Direct child and first propagation hop onto chain/0.
		Expect(aliases["chain/0"]).To(ContainElements("chain/1", "chain/2"))

		expanded := lg.ExpandFilePathAliases("chain/0/sub", aliases)
		Expect(expanded).To(ContainElement("chain/100/sub"))
	})

	It("stops walking a graph chain on a symlink cycle", func() {
		lg := LinkGraph{
			"a": &linkNode{Name: "a", Deps: &linkNode{Name: "b"}, Type: symlink},
			"b": &linkNode{Name: "b", Deps: &linkNode{Name: "a"}, Type: symlink},
		}

		var visited []string
		walkGraphChain("a", lg, func(n string, deps *linkNode) bool {
			visited = append(visited, n)
			return true
		})

		Expect(visited).To(Equal([]string{"b"}))
	})
})
