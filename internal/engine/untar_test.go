package engine

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

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
		err := writeTarballWithLinkChain(&buf, content, "usr/share/rpm/rpmdb.sqlite", chain)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		// Extract using pattern that matches through the symlink
		err = untar(context.Background(), tmpDir, img, []string{"usr/lib/sysimage/rpm/rpmdb.sqlite"})
		Expect(err).ToNot(HaveOccurred())

		// Verify the symlink was created
		verifySymlinkExists(tmpDir, "usr/lib/sysimage/rpm", "usr/lib/sysimage/rpm")

		// Verify extraction and readability through the symlink
		verifyLinkChainExtraction(tmpDir, "usr/share/rpm/rpmdb.sqlite", "usr/lib/sysimage/rpm/rpmdb.sqlite", content)
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
		err := writeTarballWithLinkChain(&buf, content, "usr/share/rpm/rpmdb.sqlite", chain)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		// Extract using pattern that matches through the chained symlinks
		err = untar(context.Background(), tmpDir, img, []string{"usr/lib/sysimage/rpm/rpmdb.sqlite"})
		Expect(err).ToNot(HaveOccurred())

		// Verify both symlinks were created
		verifySymlinkExists(tmpDir, "foo/bar/rpm", "foo/bar/rpm")
		verifySymlinkExists(tmpDir, "usr/lib/sysimage/rpm", "usr/lib/sysimage/rpm")

		// Verify the actual file was extracted and we can read through both symlink paths
		verifyLinkChainExtraction(tmpDir, "usr/share/rpm/rpmdb.sqlite", "usr/share/rpm/rpmdb.sqlite", content)
		verifyLinkChainExtraction(tmpDir, "usr/share/rpm/rpmdb.sqlite", "foo/bar/rpm/rpmdb.sqlite", content)
		verifyLinkChainExtraction(tmpDir, "usr/share/rpm/rpmdb.sqlite", "usr/lib/sysimage/rpm/rpmdb.sqlite", content)
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

		chain := []linkChainEntry{
			{name: "sysroot/ostree/repo/objects/53/hash.file", linkType: symlink, target: "../../share/rpm"},
			{name: "usr/lib/sysimage/rpm", linkType: hardlink, target: "sysroot/ostree/repo/objects/53/hash.file"},
		}
		err := writeTarballWithLinkChain(&buf, content, "usr/share/rpm/rpmdb.sqlite", chain)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		// Extract using pattern that matches through the hardlink-to-symlink
		err = untar(context.Background(), tmpDir, img, []string{"usr/lib/sysimage/rpm/rpmdb.sqlite"})
		Expect(err).ToNot(HaveOccurred())

		// Verify both links were extracted as symlinks (hardlink to symlink becomes symlink)
		verifySymlinkExists(tmpDir, "sysroot/ostree/repo/objects/53/hash.file", "ostree object")
		verifySymlinkExists(tmpDir, "usr/lib/sysimage/rpm", "hardlink to symlink")

		// Verify extraction and readability through the chain
		verifyLinkChainExtraction(tmpDir, "usr/share/rpm/rpmdb.sqlite", "usr/lib/sysimage/rpm/rpmdb.sqlite", content)
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
		err := writeTarballWithLinkChain(&buf, content, "actual/dir/file.txt", chain)
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
		err := writeTarballWithLinkChain(&buf, content, "real/location/file.txt", chain)
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
		err := writeTarballWithLinkChain(&buf, content, "target/data.db", chain)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		// Extract using pattern that matches through the alternating chain
		err = untar(context.Background(), tmpDir, img, []string{"alias/hlink1/data.db"})
		Expect(err).ToNot(HaveOccurred())

		verifyLinkChainExtraction(tmpDir, "target/data.db", "alias/hlink1/data.db", content)
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

// writeTarballWithLinkChain writes a tar archive with a regular file and a chain of hardlinks/symlinks.
// The chain is specified as a slice of linkChainEntry, where each entry points to the next (or the final file).
// Example: for chain [link1->link2, link2->file], pass entries for link1 and link2.
func writeTarballWithLinkChain(out io.Writer, contents []byte, filename string, chain []linkChainEntry) error {
	tw := tar.NewWriter(out)
	defer tw.Close()

	// Write the actual file first
	fileHeader := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     filename,
		Size:     int64(len(contents)),
		Mode:     0o644,
		Format:   tar.FormatPAX,
	}
	if err := tw.WriteHeader(fileHeader); err != nil {
		return err
	}
	if _, err := tw.Write(contents); err != nil {
		return err
	}

	// Write each link in the chain
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
