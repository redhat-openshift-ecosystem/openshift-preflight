package engine

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// createImageWithLayer creates a v1.Image with a single layer containing the provided tar content
func createImageWithLayer(tarContent []byte) (v1.Image, error) {
	layer := static.NewLayer(tarContent, types.DockerLayer)
	return mutate.AppendLayers(empty.Image, layer)
}

var _ = Describe("Link Path Resolution", func() {
	DescribeTable(
		"Link targets should resolve correctly",
		func(old, new, expectedOld, expectedNew string) {
			resO, resN := resolveLinkPaths(old, new)
			Expect(resO).To(Equal(expectedOld))
			Expect(resN).To(Equal(expectedNew))
		},
		Entry("Link at root with relative origin", "../usr/lib/file", "file", "/usr/lib/file", "file"),
		Entry("Origin is absolute", "/usr/lib/file", "file", "/usr/lib/file", "file"),
		Entry("Link in dir with relative origin", "../usr/lib/file", "etc/file", "usr/lib/file", "etc/file"),
		Entry("Link in dir with relative origin and up multiple levels", "../../cfg/file", "etc/foo/file", "cfg/file", "etc/foo/file"),
	)
})

var _ = Describe("planExtraction symlink targets", func() {
	It("stores absolute symlink linknames as archive-relative paths", func() {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		content := []byte("elf")
		regHdr := tar.Header{Name: "usr/bin/bash", Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg, Format: tar.FormatPAX}
		Expect(tw.WriteHeader(&regHdr)).To(Succeed())
		_, err := tw.Write(content)
		Expect(err).ToNot(HaveOccurred())
		symHdr := tar.Header{Name: "bin/sh", Typeflag: tar.TypeSymlink, Linkname: "/usr/bin/bash", Mode: 0o777, Format: tar.FormatPAX}
		Expect(tw.WriteHeader(&symHdr)).To(Succeed())
		Expect(tw.Close()).To(Succeed())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		_, lg, err := planExtraction(context.Background(), img, []string{"usr/bin/bash"})
		Expect(err).ToNot(HaveOccurred())

		sh, ok := lg["bin/sh"]
		Expect(ok).To(BeTrue())
		Expect(sh.Deps).ToNot(BeNil())
		Expect(sh.Deps.Name).To(Equal("usr/bin/bash"))
	})

	It("returns an error for invalid glob patterns", func() {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		Expect(tw.Close()).To(Succeed())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		_, _, err = planExtraction(context.Background(), img, []string{"[["})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid required file patterns"))
	})
})

var _ = Describe("planExtraction buildExtractionPlan symlink alias dedup", func() {
	It("plans when a file symlink path is already visited as a parent of the matched alias path", func() {
		// Directory symlink entry -> deep/real; file symlink entry/nested -> deep/real/nested/f.txt.
		// addParentLinks("entry/nested/f.txt", ...) marks "entry/nested" before linkAliases[canonical file]
		// lists sym "entry/nested", so buildExtractionPlan skips re-processing that sym.
		content := []byte("nested alias dedup content")
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		regPath := "deep/real/nested/f.txt"
		Expect(tw.WriteHeader(&tar.Header{
			Name: regPath, Typeflag: tar.TypeReg, Size: int64(len(content)),
			Mode: 0o644, Format: tar.FormatPAX,
		})).To(Succeed())
		_, err := tw.Write(content)
		Expect(err).ToNot(HaveOccurred())
		Expect(tw.WriteHeader(&tar.Header{
			Name: "entry", Typeflag: tar.TypeSymlink, Linkname: "deep/real",
			Mode: 0o777, Format: tar.FormatPAX,
		})).To(Succeed())
		Expect(tw.WriteHeader(&tar.Header{
			Name: "entry/nested", Typeflag: tar.TypeSymlink, Linkname: "deep/real/nested/f.txt",
			Mode: 0o777, Format: tar.FormatPAX,
		})).To(Succeed())
		Expect(tw.Close()).To(Succeed())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		planned, _, err := planExtraction(context.Background(), img, []string{"entry/nested/f.txt"})
		Expect(err).ToNot(HaveOccurred())
		Expect(planned).To(ContainElement(regPath))
		Expect(planned).To(ContainElement("entry"))
	})
})

var _ = Describe("planExtraction and runExtraction TypeDir skipping", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "engine-typedir-test-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	It("plans a regular file when a directory header precedes it in the stream", func() {
		content := []byte("under typedir")
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		Expect(tw.WriteHeader(&tar.Header{
			Name: "typedir/case/", Typeflag: tar.TypeDir, Mode: 0o755, Format: tar.FormatPAX,
		})).To(Succeed())
		Expect(tw.WriteHeader(&tar.Header{
			Name: "typedir/case/file.txt", Typeflag: tar.TypeReg, Size: int64(len(content)),
			Mode: 0o644, Format: tar.FormatPAX,
		})).To(Succeed())
		_, err := tw.Write(content)
		Expect(err).ToNot(HaveOccurred())
		Expect(tw.Close()).To(Succeed())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		planned, _, err := planExtraction(context.Background(), img, []string{"typedir/case/file.txt"})
		Expect(err).ToNot(HaveOccurred())
		Expect(planned).To(ContainElement("typedir/case/file.txt"))
	})

	It("extracts a regular file when a directory header precedes it in the stream", func() {
		content := []byte("extract under typedir")
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		Expect(tw.WriteHeader(&tar.Header{
			Name: "typedir/extract/", Typeflag: tar.TypeDir, Mode: 0o755, Format: tar.FormatPAX,
		})).To(Succeed())
		Expect(tw.WriteHeader(&tar.Header{
			Name: "typedir/extract/out.txt", Typeflag: tar.TypeReg, Size: int64(len(content)),
			Mode: 0o644, Format: tar.FormatPAX,
		})).To(Succeed())
		_, err := tw.Write(content)
		Expect(err).ToNot(HaveOccurred())
		Expect(tw.Close()).To(Succeed())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		err = untar(context.Background(), tmpDir, img, []string{"typedir/extract/out.txt"})
		Expect(err).ToNot(HaveOccurred())
		got, err := os.ReadFile(filepath.Join(tmpDir, "typedir/extract/out.txt"))
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal(content))
	})
})

var _ = Describe("runExtraction deferred hardlink edge cases", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "engine-deferred-hl-test-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	It("continues when the deferred hardlink is missing from the link graph at replay", func() {
		content := []byte("deferred hl missing graph entry")
		var buf bytes.Buffer
		chain := []linkChainEntry{
			{name: "by-hardlink/data.bin", linkType: hardlink, target: "actual/data.bin"},
		}
		err := writeTarballWithLinkChain(&buf, content, "actual/data.bin", chain, true)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		files, lg, err := planExtraction(context.Background(), img, []string{"by-hardlink/data.bin"})
		Expect(err).ToNot(HaveOccurred())

		lgPartial := maps.Clone(lg)
		delete(lgPartial, "by-hardlink/data.bin")

		err = runExtraction(context.Background(), tmpDir, img, files, lgPartial)
		Expect(err).ToNot(HaveOccurred())

		got, err := os.ReadFile(filepath.Join(tmpDir, "actual/data.bin"))
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal(content))

		_, err = os.Lstat(filepath.Join(tmpDir, "by-hardlink/data.bin"))
		Expect(errors.Is(err, os.ErrNotExist)).To(BeTrue())
	})

	It("does not create the deferred hardlink when its target file is not in the extraction plan", func() {
		content := []byte("deferred hl target not extracted")
		var buf bytes.Buffer
		chain := []linkChainEntry{
			{name: "by-hardlink/data.bin", linkType: hardlink, target: "actual/data.bin"},
		}
		err := writeTarballWithLinkChain(&buf, content, "actual/data.bin", chain, true)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		_, lg, err := planExtraction(context.Background(), img, []string{"by-hardlink/data.bin"})
		Expect(err).ToNot(HaveOccurred())

		// Omit the hardlink target from the extraction list so replay sees target not extracted.
		err = runExtraction(context.Background(), tmpDir, img, []string{"by-hardlink/data.bin"}, lg)
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Lstat(filepath.Join(tmpDir, "by-hardlink/data.bin"))
		Expect(errors.Is(err, os.ErrNotExist)).To(BeTrue())
		_, err = os.Lstat(filepath.Join(tmpDir, "actual/data.bin"))
		Expect(errors.Is(err, os.ErrNotExist)).To(BeTrue())
	})
})

var _ = Describe("sortHardlinksByDependencies", func() {
	It("returns an error when deferred hardlinks form a cycle", func() {
		g := LinkGraph{
			"a": &linkNode{Name: "a", Deps: &linkNode{Name: "b"}, Type: hardlink},
			"b": &linkNode{Name: "b", Deps: &linkNode{Name: "a"}, Type: hardlink},
		}
		_, err := sortHardlinksByDependencies([]string{"a", "b"}, g)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cyclic hardlink dependency"))
		Expect(err.Error()).To(And(ContainSubstring("a"), ContainSubstring("b")))
	})

	It("returns a topological order when there is no cycle", func() {
		g := LinkGraph{
			"target": &linkNode{Name: "target"},
			"a":      &linkNode{Name: "a", Deps: &linkNode{Name: "target"}, Type: hardlink},
		}
		sorted, err := sortHardlinksByDependencies([]string{"a"}, g)
		Expect(err).ToNot(HaveOccurred())
		Expect(sorted).To(Equal([]string{"a"}))
	})

	It("orders deferred hardlinks when one deferred link targets another deferred link", func() {
		// hlA -> hlB (both deferred) -> base (regular file, not in deferred set).
		// Kahn: hlB has in-degree 0, hlA depends on hlB, so hlB must precede hlA.
		g := LinkGraph{
			"base": &linkNode{Name: "base"},
			"hlB":  &linkNode{Name: "hlB", Deps: &linkNode{Name: "base"}, Type: hardlink},
			"hlA":  &linkNode{Name: "hlA", Deps: &linkNode{Name: "hlB"}, Type: hardlink},
		}
		sorted, err := sortHardlinksByDependencies([]string{"hlA", "hlB"}, g)
		Expect(err).ToNot(HaveOccurred())
		idxB := slices.Index(sorted, "hlB")
		idxA := slices.Index(sorted, "hlA")
		Expect(idxB).To(BeNumerically("<", idxA), "hardlink target must be ordered before dependent deferred hardlink")
	})
})

var _ = Describe("Untar Directory Traversal Protection", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "untar-traversal-test-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	DescribeTable("for files and directories",
		func(targetFilePath string) {
			content := []byte("malicious content")
			var buf bytes.Buffer

			err := writeTarball(&buf, content, targetFilePath, 0)
			Expect(err).ToNot(HaveOccurred())

			// Extract the tar archive - should fail with an error
			img, err := createImageWithLayer(buf.Bytes())
			Expect(err).ToNot(HaveOccurred())
			err = untar(context.Background(), tmpDir, img, []string{targetFilePath})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("path escapes from parent"))
		},
		Entry("has a relative path with single dot-dot", "../malicious.txt"),
		Entry("has a relative path with multiple dot-dots", "../../../malicious.txt"),
	)

	It("should allow extraction of legitimate files within the destination directory", func() {
		content := []byte("legitimate content")
		var buf bytes.Buffer

		err := writeTarball(&buf, content, "subdir/legitimate.txt", 0)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())
		err = untar(context.Background(), tmpDir, img, []string{"subdir/legitimate.txt"})
		Expect(err).ToNot(HaveOccurred())

		// Check that the legitimate file was created
		legitimateFile := filepath.Join(tmpDir, "subdir", "legitimate.txt")
		fileContent, err := os.ReadFile(legitimateFile)
		Expect(err).ToNot(HaveOccurred())
		Expect(fileContent).To(Equal(content))
	})

	It("skips symlinks whose target resolves outside the extraction root", func() {
		content := []byte("safe payload")
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		regHdr := tar.Header{Name: "safe/ok.txt", Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg, Format: tar.FormatPAX}
		Expect(tw.WriteHeader(&regHdr)).To(Succeed())
		_, err := tw.Write(content)
		Expect(err).ToNot(HaveOccurred())
		// Lexical Join(safe, linkname) cleans to a path with enough ".." to escape dst when joined with tmpDir.
		escapeLinkname := strings.Repeat("../", 24) + "outside_escape_marker"
		symHdr := tar.Header{Name: "safe/evil_symlink", Typeflag: tar.TypeSymlink, Linkname: escapeLinkname, Mode: 0o777, Format: tar.FormatPAX}
		Expect(tw.WriteHeader(&symHdr)).To(Succeed())
		Expect(tw.Close()).To(Succeed())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())
		err = untar(context.Background(), tmpDir, img, []string{"safe/ok.txt", "safe/evil_symlink"})
		Expect(err).ToNot(HaveOccurred())

		okPath := filepath.Join(tmpDir, "safe", "ok.txt")
		got, err := os.ReadFile(okPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal(content))

		_, err = os.Lstat(filepath.Join(tmpDir, "safe", "evil_symlink"))
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("should extract nested files when /licenses is a symlink to usr/share/licenses (tar paths under usr/share/licenses)", func() {
		// Mirrors UBI/RHEL images: layer stores license files under usr/share/licenses/... while
		// /licenses is a symlink to /usr/share/licenses. Nested paths must match after resolving
		// the symlink target (including follow-up passes when the symlink appears after paths).
		content := []byte("license text")
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		nestedPath := "usr/share/licenses/demo/LICENSE"
		err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     nestedPath,
			Size:     int64(len(content)),
			Mode:     0o644,
			Format:   tar.FormatPAX,
		})
		Expect(err).ToNot(HaveOccurred())
		_, err = tw.Write(content)
		Expect(err).ToNot(HaveOccurred())

		err = tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeSymlink,
			Name:     "licenses",
			Linkname: "/usr/share/licenses",
			Mode:     0o777,
			Format:   tar.FormatPAX,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(tw.Close()).To(Succeed())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		err = untar(context.Background(), tmpDir, img, []string{"licenses/**"})
		Expect(err).ToNot(HaveOccurred())

		got, err := os.ReadFile(filepath.Join(tmpDir, nestedPath))
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal(content))
	})

	It("should extract files in nested subdirectories when pattern ends with /**", func() {
		// Some checks (e.g. HasModifiedFiles) require extraction of nested subdirectories
		// https://github.com/redhat-openshift-ecosystem/openshift-preflight/pull/1393
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		match_files := []string{
			"a/b",
			"a/c/d",
			"a/e/f/g",
			"a/h/i/j/k",
		}

		reject_files := []string{
			"z",
			"y/x",
		}

		content := []byte("foo")

		for _, f := range slices.Concat(match_files, reject_files) {
			err := tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeReg,
				Name:     f,
				Size:     int64(len(content)),
				Mode:     0o644,
				Format:   tar.FormatPAX,
			})
			Expect(err).ToNot(HaveOccurred())
			_, err = tw.Write(content)
			Expect(err).ToNot(HaveOccurred())
		}
		Expect(tw.Close()).To(Succeed())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		err = untar(context.Background(), tmpDir, img, []string{"a/**"})
		Expect(err).ToNot(HaveOccurred())

		for _, f := range match_files {
			read, err := os.ReadFile(filepath.Join(tmpDir, f))
			Expect(err).ToNot(HaveOccurred(), "expected %s to be extracted", f)
			Expect(read).To(Equal(content))
		}

		for _, f := range reject_files {
			_, err = os.Stat(filepath.Join(tmpDir, f))
			Expect(os.IsNotExist(err)).To(BeTrue(), "expected %s not to be extracted", f)
		}
	})

	DescribeTable("for (sym)links",
		func(linkType linkType, linkTarget string) {
			content := []byte("placeholder")
			var buf bytes.Buffer

			// Create link with Linkname that tries to escape the extraction
			// directory. We don't really care about normal-file.txt here.
			err := writeTarballWithLink(&buf, linkType, content, "normal-file", "malicious-link", linkTarget)
			Expect(err).ToNot(HaveOccurred())

			img, err := createImageWithLayer(buf.Bytes())
			Expect(err).ToNot(HaveOccurred())
			err = untar(context.Background(), tmpDir, img, []string{linkTarget})
			// invalid links do not throw an error, they're just skipped
			Expect(err).ToNot(HaveOccurred())

			linkPath := filepath.Join(tmpDir, "malicious-link")
			_, err = os.Stat(linkPath)
			// link should not exist
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no such file or directory"), "links should not be untar'd if pointing to path outside of base")
		},

		Entry("has a hard link with single dot-dot traversal", hardlink, "../../mnt"),
		Entry("has a hard link with multiple dot-dot traversal", hardlink, "../../../external-file.txt"),
		Entry("has a hard link with mixed traversal", hardlink, "../../usr/../external-file.txt"),
		Entry("has a hard link with absolute path oldname", hardlink, "/mnt"),
		Entry("has a symlink with single dot-dot traversal", symlink, "../../mnt"),
		Entry("has a symlink with multiple dot-dot traversal", symlink, "../../../external-file.txt"),
		Entry("has a symlink with mixed traversal", symlink, "../../usr/../external-file.txt"),
		Entry("has a symlink with absolute path oldname", symlink, "/mnt"),
	)

	It("should allow creation of legitimate hard links within the destination directory", func() {
		content := []byte("legitimate content")
		var buf bytes.Buffer
		err := writeTarballWithLink(&buf, hardlink, content, "original.txt", "legitimate-hardlink.txt", "original.txt")
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())
		err = untar(context.Background(), tmpDir, img, []string{"original.txt", "legitimate-hardlink.txt"})
		Expect(err).ToNot(HaveOccurred())

		// Check that both the original file and hard link were created
		originalFile := filepath.Join(tmpDir, "original.txt")
		linkFile := filepath.Join(tmpDir, "legitimate-hardlink.txt")

		originalContent, err := os.ReadFile(originalFile)
		Expect(err).ToNot(HaveOccurred())
		Expect(originalContent).To(Equal(content))

		linkContent, err := os.ReadFile(linkFile)
		Expect(err).ToNot(HaveOccurred())
		Expect(linkContent).To(Equal(content))
	})

	It("should allow creation of legitimate symlinks within the destination directory", func() {
		content := []byte("legitimate content")
		var buf bytes.Buffer
		err := writeTarballWithLink(&buf, symlink, content, "original.txt", "legitimate-symlink.txt", "original.txt")
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())
		err = untar(context.Background(), tmpDir, img, []string{"original.txt", "legitimate-symlink.txt"})
		Expect(err).ToNot(HaveOccurred())

		// Check that both the original file and symlink were created
		originalFile := filepath.Join(tmpDir, "original.txt")
		linkFile := filepath.Join(tmpDir, "legitimate-symlink.txt")

		originalContent, err := os.ReadFile(originalFile)
		Expect(err).ToNot(HaveOccurred())
		Expect(originalContent).To(Equal(content))

		linkContent, err := os.ReadFile(linkFile)
		Expect(err).ToNot(HaveOccurred())
		Expect(linkContent).To(Equal(content))
	})

	It("sanitizes redundant relative symlink targets to a canonical path", func() {
		const (
			dataPath          = "p/mnt/data.txt"
			maliciousLinkPath = "a/b/c/malicious-link"
			redundantLinkname = "../../../p/mnt/../mnt"
		)
		content := []byte("data under p/mnt")
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		regHdr := tar.Header{Name: dataPath, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg, Format: tar.FormatPAX}
		Expect(tw.WriteHeader(&regHdr)).To(Succeed())
		_, err := tw.Write(content)
		Expect(err).ToNot(HaveOccurred())
		symHdr := tar.Header{Name: maliciousLinkPath, Typeflag: tar.TypeSymlink, Linkname: redundantLinkname, Mode: 0o777, Format: tar.FormatPAX}
		Expect(tw.WriteHeader(&symHdr)).To(Succeed())
		Expect(tw.Close()).To(Succeed())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())
		err = untar(context.Background(), tmpDir, img, []string{dataPath, maliciousLinkPath})
		Expect(err).ToNot(HaveOccurred())

		linkAbs := filepath.Join(tmpDir, maliciousLinkPath)
		wantTarget, err := filepath.Rel(filepath.Join(tmpDir, filepath.Dir(maliciousLinkPath)), filepath.Join(tmpDir, "p", "mnt"))
		Expect(err).ToNot(HaveOccurred())
		Expect(redundantLinkname).ToNot(Equal(wantTarget), "fixture should use a non-canonical tar linkname")

		gotTarget, err := os.Readlink(linkAbs)
		Expect(err).ToNot(HaveOccurred())
		Expect(gotTarget).To(Equal(wantTarget), "symlink target should be canonical relative form, not raw tar Linkname")

		throughLink, err := os.ReadFile(filepath.Join(linkAbs, "data.txt"))
		Expect(err).ToNot(HaveOccurred())
		Expect(throughLink).To(Equal(content))
	})

	It("should allow creation of multi-layered symlinks within the destination directory", func() {
		// Test both normal and reverse order; symlinks may appear before or after
		// the target file in a tar stream.
		testCases := []struct {
			description  string
			reverseOrder bool
			subdir       string
		}{
			{
				description:  "with entries in normal order (file, then symlinks)",
				reverseOrder: false,
				subdir:       "normal",
			},
			{
				description:  "with entries in reverse order (symlinks, then file)",
				reverseOrder: true,
				subdir:       "reverse",
			},
		}

		for _, tc := range testCases {
			By(tc.description)
			content := []byte("multi-layer content")
			var buf bytes.Buffer

			// Use subdirectories to avoid conflicts between test cases
			originalName := filepath.Join(tc.subdir, "original.txt")
			link1Name := filepath.Join(tc.subdir, "link1.txt")
			link2Name := filepath.Join(tc.subdir, "link2.txt")

			// Create a tarball with: original.txt -> link1.txt -> link2.txt
			err := writeTarballWithMultiLayerLinks(&buf, content, originalName, link1Name, link2Name, tc.reverseOrder)
			Expect(err).ToNot(HaveOccurred())

			img, err := createImageWithLayer(buf.Bytes())
			Expect(err).ToNot(HaveOccurred())
			err = untar(context.Background(), tmpDir, img, []string{originalName, link1Name, link2Name})
			Expect(err).ToNot(HaveOccurred())

			// Check that the original file was created
			originalFile := filepath.Join(tmpDir, originalName)
			originalContent, err := os.ReadFile(originalFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(originalContent).To(Equal(content))

			// Check that the first symlink exists and points to original
			// Note: untar keeps relative symlinks as relative (for OSTree compatibility)
			link1File := filepath.Join(tmpDir, link1Name)
			link1Info, err := os.Lstat(link1File)
			Expect(err).ToNot(HaveOccurred())
			Expect(link1Info.Mode()&os.ModeSymlink).To(Equal(os.ModeSymlink), "link1.txt should be a symlink")

			link1Target, err := os.Readlink(link1File)
			Expect(err).ToNot(HaveOccurred())
			// The symlink target is relative (e.g., "original.txt")
			Expect(link1Target).To(Equal(filepath.Base(originalName)), "link1.txt should have relative target to original.txt")

			// Check that the second symlink exists and points to link1
			link2File := filepath.Join(tmpDir, link2Name)
			link2Info, err := os.Lstat(link2File)
			Expect(err).ToNot(HaveOccurred())
			Expect(link2Info.Mode()&os.ModeSymlink).To(Equal(os.ModeSymlink), "link2.txt should be a symlink")

			link2Target, err := os.Readlink(link2File)
			Expect(err).ToNot(HaveOccurred())
			// The symlink target is relative (e.g., "link1.txt")
			Expect(link2Target).To(Equal(filepath.Base(link1Name)), "link2.txt should have relative target to link1.txt")

			// Check that reading through the chain of symlinks gives the correct content
			link1Content, err := os.ReadFile(link1File)
			Expect(err).ToNot(HaveOccurred())
			Expect(link1Content).To(Equal(content))

			link2Content, err := os.ReadFile(link2File)
			Expect(err).ToNot(HaveOccurred())
			Expect(link2Content).To(Equal(content))
		}
	})

	It("should resolve multi-layered symlinks in a single extraction", func() {
		content := []byte("test content")
		var buf bytes.Buffer

		originalName := "progressive-test/original.txt"
		link1Name := "progressive-test/link1.txt"
		link2Name := "progressive-test/link2.txt"

		// Create a tarball with multi-layered symlinks
		err := writeTarballWithMultiLayerLinks(&buf, content, originalName, link1Name, link2Name, false)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		// Extract using the untar function with only link2 as the required file
		// This should automatically resolve and extract link1 and original as well
		err = untar(context.Background(), tmpDir, img, []string{link2Name})
		Expect(err).ToNot(HaveOccurred())

		// Verify all files were extracted
		link2File := filepath.Join(tmpDir, link2Name)
		link1File := filepath.Join(tmpDir, link1Name)
		originalFile := filepath.Join(tmpDir, originalName)

		_, err = os.Lstat(link2File)
		Expect(err).ToNot(HaveOccurred(), "link2.txt should exist")

		_, err = os.Lstat(link1File)
		Expect(err).ToNot(HaveOccurred(), "link1.txt should exist")

		_, err = os.Stat(originalFile)
		Expect(err).ToNot(HaveOccurred(), "original.txt should exist")

		// Verify the complete symlink chain works
		link2Content, err := os.ReadFile(link2File)
		Expect(err).ToNot(HaveOccurred())
		Expect(link2Content).To(Equal(content), "reading through link2 should return original content")

		link1Content, err := os.ReadFile(link1File)
		Expect(err).ToNot(HaveOccurred())
		Expect(link1Content).To(Equal(content), "reading through link1 should return original content")
	})

	It("should extract files when parent directory is a symlink and child matches filter", func() {
		// This tests the scenario where:
		// - /usr/lib/sysimage/rpm is a symlink to ../../share/rpm
		// - The actual file is at /usr/share/rpm/rpmdb.sqlite
		// - The filter pattern matches /usr/lib/sysimage/rpm/rpmdb.sqlite (the symlinked path)
		// - The file should be extracted along with the directory symlink

		content := []byte("database content")
		var buf bytes.Buffer

		chain := []linkChainEntry{
			{name: "usr/lib/sysimage/rpm", linkType: symlink, target: "../../share/rpm"},
		}
		err := writeTarballWithLinkChain(&buf, content, "usr/share/rpm/rpmdb.sqlite", chain, false)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		// Extract using pattern that matches through the symlink
		err = untar(context.Background(), tmpDir, img, []string{testOstreeExtractPattern})
		Expect(err).ToNot(HaveOccurred())

		// Verify the symlink was created
		verifySymlinkExists(tmpDir, testOstreeConsumerHardlink, testOstreeConsumerHardlink)

		// Verify extraction and readability through the symlink
		verifyLinkChainExtraction(tmpDir, testOstreeRPMDBPath, testOstreeExtractPattern, content)
	})

	It("should extract files through chained directory symlinks", func() {
		// This tests the scenario where:
		// - usr/lib/sysimage/rpm -> ../../../foo/bar/rpm (symlink to symlink)
		// - foo/bar/rpm -> ../../usr/share/rpm (symlink to directory)
		// - The actual file is at usr/share/rpm/rpmdb.sqlite
		// - The filter pattern matches usr/lib/sysimage/rpm/rpmdb.sqlite
		// - Both symlinks and the file should be extracted

		content := []byte("chained symlink content")
		var buf bytes.Buffer

		chain := []linkChainEntry{
			{name: "foo/bar/rpm", linkType: symlink, target: "../../usr/share/rpm"},
			{name: "usr/lib/sysimage/rpm", linkType: symlink, target: "../../../foo/bar/rpm"},
		}
		err := writeTarballWithLinkChain(&buf, content, "usr/share/rpm/rpmdb.sqlite", chain, false)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		// Extract using pattern that matches through the chained symlinks
		err = untar(context.Background(), tmpDir, img, []string{testOstreeExtractPattern})
		Expect(err).ToNot(HaveOccurred())

		// Verify both symlinks were created
		verifySymlinkExists(tmpDir, "foo/bar/rpm", "foo/bar/rpm")
		verifySymlinkExists(tmpDir, testOstreeConsumerHardlink, testOstreeConsumerHardlink)

		// Verify the actual file was extracted and we can read through both symlink paths
		verifyLinkChainExtraction(tmpDir, testOstreeRPMDBPath, testOstreeRPMDBPath, content)
		verifyLinkChainExtraction(tmpDir, testOstreeRPMDBPath, "foo/bar/rpm/rpmdb.sqlite", content)
		verifyLinkChainExtraction(tmpDir, testOstreeRPMDBPath, testOstreeExtractPattern, content)
	})

	It("should extract files through hardlinks to directory symlinks (OSTree pattern)", func() {
		// This tests the OSTree container image pattern where:
		// - sysroot/ostree/repo/objects/HASH.file is a symlink to ../../share/rpm
		// - usr/lib/sysimage/rpm is a hardlink to sysroot/ostree/repo/objects/HASH.file
		// - usr/share/rpm/rpmdb.sqlite is a regular file (or hardlink to ostree object)
		// - The pattern matches usr/lib/sysimage/rpm/rpmdb.sqlite
		// - Both the file and the hardlink-to-symlink chain should be extracted

		content := []byte("rpm database content")
		var buf bytes.Buffer

		err := writeOstreeRPMTestFixture(&buf, content)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		// Extract using pattern that matches through the hardlink-to-symlink
		err = untar(context.Background(), tmpDir, img, []string{testOstreeExtractPattern})
		Expect(err).ToNot(HaveOccurred())

		// Verify both links were extracted as symlinks (hardlink to symlink becomes symlink)
		verifySymlinkExists(tmpDir, testOstreeObjectSymlinkPath, "ostree object")
		verifySymlinkExists(tmpDir, testOstreeConsumerHardlink, "hardlink to symlink")

		// Verify extraction and readability through the chain
		verifyLinkChainExtraction(tmpDir, testOstreeRPMDBPath, filepath.Join(testOstreeConsumerHardlink, "rpmdb.sqlite"), content)
	})

	It("should extract entry/nested when the regular file appears before the directory symlink in the tar stream", func() {
		content := []byte("layer-payload")
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		regPath := "deep/real/nested/f.txt"
		Expect(tw.WriteHeader(&tar.Header{Name: regPath, Typeflag: tar.TypeReg, Size: int64(len(content)), Mode: 0o644, Format: tar.FormatPAX})).To(Succeed())
		_, err := tw.Write(content)
		Expect(err).ToNot(HaveOccurred())
		Expect(tw.WriteHeader(&tar.Header{Name: "entry", Typeflag: tar.TypeSymlink, Linkname: "deep/real", Mode: 0o777, Format: tar.FormatPAX})).To(Succeed())
		Expect(tw.Close()).To(Succeed())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())
		err = untar(context.Background(), tmpDir, img, []string{"entry/nested/f.txt"})
		Expect(err).ToNot(HaveOccurred())

		got, err := os.ReadFile(filepath.Join(tmpDir, regPath))
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal(content))

		got, err = os.ReadFile(filepath.Join(tmpDir, "entry", "nested", "f.txt"))
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal(content))
	})

	It("should not fail extraction when a requested symlink is dangling", func() {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		Expect(tw.WriteHeader(&tar.Header{Name: "keep.txt", Typeflag: tar.TypeReg, Size: 1, Mode: 0o644, Format: tar.FormatPAX})).To(Succeed())
		_, err := tw.Write([]byte("y"))
		Expect(err).ToNot(HaveOccurred())
		Expect(tw.WriteHeader(&tar.Header{Name: "dangle", Typeflag: tar.TypeSymlink, Linkname: "ghost/missing.txt", Mode: 0o777, Format: tar.FormatPAX})).To(Succeed())
		Expect(tw.Close()).To(Succeed())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())
		err = untar(context.Background(), tmpDir, img, []string{"keep.txt", "dangle"})
		Expect(err).ToNot(HaveOccurred())

		_, err = os.ReadFile(filepath.Join(tmpDir, "keep.txt"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Lstat(filepath.Join(tmpDir, "dangle"))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should resolve symlink filter targets from a good hardlink peer when the bad peer appears first in the tar", func() {
		// Same relative Linkname on two hard-linked symlink paths; resolving from the first path's
		// dirname yields a bogus prefix (bad/target/...) that is not in the archive. Pattern
		// discovery must use the peer that names real members under target/. Include each symlink
		// path in required patterns so both headers are extracted (target/data.txt alone does not
		// mark optional symlinks for extraction).
		content := []byte("hardlink symlink peer order content")
		var buf bytes.Buffer
		err := writeTarSymlinkHardlinkPeersBadSymlinkFirst(&buf, content)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		err = untar(context.Background(), tmpDir, img, []string{"target/data.txt", "good/sym", "bad/deep/sym"})
		Expect(err).ToNot(HaveOccurred())

		actualFile := filepath.Join(tmpDir, "target", "data.txt")
		fileContent, err := os.ReadFile(actualFile)
		Expect(err).ToNot(HaveOccurred())
		Expect(fileContent).To(Equal(content))

		goodSym := filepath.Join(tmpDir, "good", "sym")
		_, err = os.Lstat(goodSym)
		Expect(err).ToNot(HaveOccurred())
		raw, err := os.Readlink(goodSym)
		Expect(err).ToNot(HaveOccurred())
		throughGood, err := os.ReadFile(filepath.Join(filepath.Dir(goodSym), filepath.FromSlash(raw)))
		Expect(err).ToNot(HaveOccurred())
		Expect(throughGood).To(Equal(content), "read through good/sym following raw linkname")

		badJoin := filepath.Join(tmpDir, "bad", "target", "data.txt")
		_, statErr := os.Lstat(badJoin)
		Expect(os.IsNotExist(statErr)).To(BeTrue(), "bogus path from wrong relative resolution must not be created")
	})

	It("should extract a symlink whose target is a hardlink peer when the pattern matches only the other peer", func() {
		// Pattern matches deep/path/file only; short/file is a hardlink peer; consumer is a symlink
		// to short/file. The consumer symlink must still be materialized to read the file via consumer.
		content := []byte("peer alias symlink content")
		var buf bytes.Buffer

		chain := []linkChainEntry{
			{name: "short/file", linkType: hardlink, target: "deep/path/file"},
			{name: "consumer", linkType: symlink, target: "short/file"},
		}
		err := writeTarballWithLinkChain(&buf, content, "deep/path/file", chain, false)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		err = untar(context.Background(), tmpDir, img, []string{"deep/path/file"})
		Expect(err).ToNot(HaveOccurred())

		verifyLinkChainExtraction(tmpDir, "deep/path/file", "consumer", content)
	})

	It("should extract files through hardlink -> hardlink -> symlink -> directory chain", func() {
		// Test: hardlink1 -> hardlink2 -> symlink -> directory
		// Pattern matches through hardlink1
		content := []byte("test content")
		var buf bytes.Buffer

		chain := []linkChainEntry{
			{name: "storage/symlink-to-dir", linkType: symlink, target: "../actual/dir"},
			{name: "intermediate/link2", linkType: hardlink, target: "storage/symlink-to-dir"},
			{name: "alias/link1", linkType: hardlink, target: "intermediate/link2"},
		}
		err := writeTarballWithLinkChain(&buf, content, "actual/dir/file.txt", chain, false)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		// Extract using pattern that matches through the hardlink chain
		err = untar(context.Background(), tmpDir, img, []string{"alias/link1/file.txt"})
		Expect(err).ToNot(HaveOccurred())

		verifyLinkChainExtraction(tmpDir, "actual/dir/file.txt", "alias/link1/file.txt", content)
	})

	It("should extract files through symlink -> hardlink -> symlink chain", func() {
		// Test: symlink1 -> hardlink -> symlink2 -> directory
		// Pattern matches through symlink1
		content := []byte("test content")
		var buf bytes.Buffer

		chain := []linkChainEntry{
			{name: "intermediate/sym2", linkType: symlink, target: "../real/location"},
			{name: "middle/hlink", linkType: hardlink, target: "intermediate/sym2"},
			{name: "alias/sym1", linkType: symlink, target: "../middle/hlink"},
		}
		err := writeTarballWithLinkChain(&buf, content, "real/location/file.txt", chain, false)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		// Extract using pattern that matches through the chain
		err = untar(context.Background(), tmpDir, img, []string{"alias/sym1/file.txt"})
		Expect(err).ToNot(HaveOccurred())

		verifyLinkChainExtraction(tmpDir, "real/location/file.txt", "alias/sym1/file.txt", content)
	})

	It("should extract files through alternating hardlink/symlink chain (hardlink -> symlink -> hardlink -> symlink -> directory)", func() {
		// Test complex alternating chain where hardlinks point to symlinks
		content := []byte("alternating chain content")
		var buf bytes.Buffer

		chain := []linkChainEntry{
			{name: "level3/sym3", linkType: symlink, target: "../target"},
			{name: "level2/hlink2", linkType: hardlink, target: "level3/sym3"},
			{name: "level1/sym1", linkType: symlink, target: "../level2/hlink2"},
			{name: "alias/hlink1", linkType: hardlink, target: "level1/sym1"},
		}
		err := writeTarballWithLinkChain(&buf, content, "target/data.db", chain, false)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		// Extract using pattern that matches through the alternating chain
		err = untar(context.Background(), tmpDir, img, []string{"alias/hlink1/data.db"})
		Expect(err).ToNot(HaveOccurred())

		verifyLinkChainExtraction(tmpDir, "target/data.db", "alias/hlink1/data.db", content)
	})

	It("should extract when a hardlink header appears before its target file in the tar stream", func() {
		// runExtraction defers hardlinks until EOF when the target is not extracted yet; replay must still create the link.
		content := []byte("deferred hardlink stream order")
		var buf bytes.Buffer

		chain := []linkChainEntry{
			{name: "by-hardlink/data.bin", linkType: hardlink, target: "actual/data.bin"},
		}
		err := writeTarballWithLinkChain(&buf, content, "actual/data.bin", chain, true)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		err = untar(context.Background(), tmpDir, img, []string{"by-hardlink/data.bin"})
		Expect(err).ToNot(HaveOccurred())

		verifyLinkChainExtraction(tmpDir, "actual/data.bin", "by-hardlink/data.bin", content)
	})
})

// writeTarballWithLink writes a tar archive with a regular file and a hard
// link. The ability to write a regular file allows for testing happy paths.
// note: this should only be used as a helper function in tests.
func writeTarballWithLink(out io.Writer, linkTypeFlag linkType, contents []byte, filename string, linkname string, linkTarget string) error {
	tw := tar.NewWriter(out)
	defer tw.Close()

	header := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     filename,
		Size:     int64(len(contents)),
		Mode:     0o644,
		Format:   tar.FormatPAX,
	}
	err := tw.WriteHeader(header)
	if err != nil {
		return err
	}
	_, err = tw.Write(contents)
	if err != nil {
		return err
	}

	linkHeader := &tar.Header{
		Typeflag: byte(linkTypeFlag),
		Name:     linkname,
		Linkname: linkTarget,
		Mode:     0o644,
		Format:   tar.FormatPAX,
	}
	return tw.WriteHeader(linkHeader)
}

// writeTarballWithMultiLayerLinks writes a tar archive with a regular file and
// multiple symlinks forming a chain (e.g., link2 -> link1 -> original).
// If reverseOrder is true, the entries are written in reverse order (symlinks before the target file).
// note: this should only be used as a helper function in tests.
func writeTarballWithMultiLayerLinks(out io.Writer, contents []byte, filename string, firstLink string, secondLink string, reverseOrder bool) error {
	tw := tar.NewWriter(out)
	defer tw.Close()

	// Prepare the tar headers
	// For symlinks, use basename for the linkname to create relative symlinks
	fileHeader := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     filename,
		Size:     int64(len(contents)),
		Mode:     0o644,
		Format:   tar.FormatPAX,
	}

	firstLinkHeader := &tar.Header{
		Typeflag: tar.TypeSymlink,
		Name:     firstLink,
		Linkname: filepath.Base(filename),
		Mode:     0o644,
		Format:   tar.FormatPAX,
	}

	secondLinkHeader := &tar.Header{
		Typeflag: tar.TypeSymlink,
		Name:     secondLink,
		Linkname: filepath.Base(firstLink),
		Mode:     0o644,
		Format:   tar.FormatPAX,
	}

	var err error
	if reverseOrder {
		// Write in reverse order: second symlink, first symlink, then original file
		err = tw.WriteHeader(secondLinkHeader)
		if err != nil {
			return err
		}

		err = tw.WriteHeader(firstLinkHeader)
		if err != nil {
			return err
		}

		err = tw.WriteHeader(fileHeader)
		if err != nil {
			return err
		}
		_, err = tw.Write(contents)
		if err != nil {
			return err
		}
	} else {
		// Write in normal order: original file, first symlink, second symlink
		err = tw.WriteHeader(fileHeader)
		if err != nil {
			return err
		}
		_, err = tw.Write(contents)
		if err != nil {
			return err
		}

		err = tw.WriteHeader(firstLinkHeader)
		if err != nil {
			return err
		}

		err = tw.WriteHeader(secondLinkHeader)
		if err != nil {
			return err
		}
	}

	return nil
}

// linkChainEntry represents a single link in a chain of hardlinks/symlinks
type linkChainEntry struct {
	name     string
	linkType linkType
	target   string
}

// Shared OSTree-style layout: object store symlink + hardlink consumer + rpmdb under usr/share/rpm.
const (
	testOstreeObjectSymlinkPath = "sysroot/ostree/repo/objects/53/hash.file"
	testOstreeRPMDBPath         = "usr/share/rpm/rpmdb.sqlite"
	testOstreeConsumerHardlink  = "usr/lib/sysimage/rpm"
	// Slash-separated path as required by filter patterns (not filepath.Join).
	testOstreeExtractPattern = testOstreeConsumerHardlink + "/rpmdb.sqlite"
)

func ostreeRPMTestFixtureChain() []linkChainEntry {
	return []linkChainEntry{
		{name: testOstreeObjectSymlinkPath, linkType: symlink, target: "../../share/rpm"},
		{name: testOstreeConsumerHardlink, linkType: hardlink, target: testOstreeObjectSymlinkPath},
	}
}

func writeOstreeRPMTestFixture(out io.Writer, contents []byte) error {
	return writeTarballWithLinkChain(out, contents, testOstreeRPMDBPath, ostreeRPMTestFixtureChain(), false)
}

// writeTarSymlinkHardlinkPeersBadSymlinkFirst writes: a symlink at bad/deep/sym with the same
// relative Linkname as good/sym (so resolving from bad/deep alone yields a nonexistent path),
// then the regular file, then good/sym, then TypeLink bad/deep/sym -> good/sym.
func writeTarSymlinkHardlinkPeersBadSymlinkFirst(out io.Writer, content []byte) error {
	tw := tar.NewWriter(out)
	defer tw.Close()

	raw := "../target/data.txt"
	headers := []tar.Header{
		{Typeflag: tar.TypeSymlink, Name: "bad/deep/sym", Linkname: raw, Mode: 0o777, Format: tar.FormatPAX},
		{Typeflag: tar.TypeReg, Name: "target/data.txt", Size: int64(len(content)), Mode: 0o644, Format: tar.FormatPAX},
		{Typeflag: tar.TypeSymlink, Name: "good/sym", Linkname: raw, Mode: 0o777, Format: tar.FormatPAX},
		{Typeflag: tar.TypeLink, Name: "bad/deep/sym", Linkname: "good/sym", Mode: 0o777, Format: tar.FormatPAX},
	}
	for i := range headers {
		if err := tw.WriteHeader(&headers[i]); err != nil {
			return err
		}
		if headers[i].Typeflag == tar.TypeReg {
			if _, err := tw.Write(content); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeTarballWithLinkChain writes a tar archive with a regular file and a chain of hardlinks/symlinks.
// The chain is specified as a slice of linkChainEntry, where each entry points to the next (or the final file).
// Example: for chain [link1->link2, link2->file], pass entries for link1 and link2.
// If linksBeforeFile is true, link headers are written before the regular file so hardlinks can
// appear before their target in the stream (deferred hardlink replay in runExtraction).
func writeTarballWithLinkChain(out io.Writer, contents []byte, filename string, chain []linkChainEntry, linksBeforeFile bool) error {
	tw := tar.NewWriter(out)
	defer tw.Close()

	fileHeader := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     filename,
		Size:     int64(len(contents)),
		Mode:     0o644,
		Format:   tar.FormatPAX,
	}

	writeLinks := func() error {
		for _, link := range chain {
			linkHeader := &tar.Header{
				Typeflag: byte(link.linkType),
				Name:     link.name,
				Linkname: link.target,
				Mode:     0o777,
				Format:   tar.FormatPAX,
			}
			if err := tw.WriteHeader(linkHeader); err != nil {
				return err
			}
		}
		return nil
	}

	if linksBeforeFile {
		if err := writeLinks(); err != nil {
			return err
		}
		if err := tw.WriteHeader(fileHeader); err != nil {
			return err
		}
		if _, err := tw.Write(contents); err != nil {
			return err
		}
		return nil
	}

	if err := tw.WriteHeader(fileHeader); err != nil {
		return err
	}
	if _, err := tw.Write(contents); err != nil {
		return err
	}
	return writeLinks()
}

// verifyLinkChainExtraction verifies that files were extracted correctly through a link chain.
// It checks that the actual file exists with correct content, and that the file can be read
// through the specified chain path.
func verifyLinkChainExtraction(tmpDir string, actualFilePath, chainPath string, expectedContent []byte) {
	// Verify the actual file was extracted
	fullActualPath := filepath.Join(tmpDir, actualFilePath)
	fileContent, err := os.ReadFile(fullActualPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(fileContent).To(Equal(expectedContent))

	// Verify we can read through the chain
	fullChainPath := filepath.Join(tmpDir, chainPath)
	chainFileContent, err := os.ReadFile(fullChainPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(chainFileContent).To(Equal(expectedContent), "should be able to read file through link chain")
}

// verifySymlinkExists verifies that a symlink exists at the given path.
// Returns the symlink target for further verification if needed.
func verifySymlinkExists(tmpDir, symlinkPath, description string) string {
	fullPath := filepath.Join(tmpDir, symlinkPath)
	info, err := os.Lstat(fullPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(info.Mode()&os.ModeSymlink).To(Equal(os.ModeSymlink), description+" should be a symlink")

	target, err := os.Readlink(fullPath)
	Expect(err).ToNot(HaveOccurred())
	return target
}
