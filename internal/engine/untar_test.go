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
			// Note: untar converts relative symlinks to absolute paths for security
			link1File := filepath.Join(tmpDir, link1Name)
			link1Info, err := os.Lstat(link1File)
			Expect(err).ToNot(HaveOccurred())
			Expect(link1Info.Mode()&os.ModeSymlink).To(Equal(os.ModeSymlink), "link1.txt should be a symlink")

			link1Target, err := os.Readlink(link1File)
			Expect(err).ToNot(HaveOccurred())
			Expect(link1Target).To(Equal(originalFile), "link1.txt should point to original.txt")

			// Check that the second symlink exists and points to link1
			link2File := filepath.Join(tmpDir, link2Name)
			link2Info, err := os.Lstat(link2File)
			Expect(err).ToNot(HaveOccurred())
			Expect(link2Info.Mode()&os.ModeSymlink).To(Equal(os.ModeSymlink), "link2.txt should be a symlink")

			link2Target, err := os.Readlink(link2File)
			Expect(err).ToNot(HaveOccurred())
			Expect(link2Target).To(Equal(link1File), "link2.txt should point to link1.txt")

			// Check that reading through the chain of symlinks gives the correct content
			link1Content, err := os.ReadFile(link1File)
			Expect(err).ToNot(HaveOccurred())
			Expect(link1Content).To(Equal(content))

			link2Content, err := os.ReadFile(link2File)
			Expect(err).ToNot(HaveOccurred())
			Expect(link2Content).To(Equal(content))
		}
	})

	It("should progressively resolve multi-layered symlinks across multiple untarOnce passes", func() {
		content := []byte("test content")
		var buf bytes.Buffer

		originalName := "progressive-test/original.txt"
		link1Name := "progressive-test/link1.txt"
		link2Name := "progressive-test/link2.txt"

		// Create a tarball in the order of file first, then links. Thus, to resolve
		// each layer of links, another pass is needed.
		err := writeTarballWithMultiLayerLinks(&buf, content, originalName, link1Name, link2Name, false)
		Expect(err).ToNot(HaveOccurred())

		img, err := createImageWithLayer(buf.Bytes())
		Expect(err).ToNot(HaveOccurred())

		state := make(map[string]bool)

		// First pass: Start with only link2.txt in the filter patterns
		filterPatterns := []string{link2Name}
		remaining, err := untarOnce(context.Background(), tmpDir, img, filterPatterns, state)
		Expect(err).ToNot(HaveOccurred())

		// After first pass: only link2 should be created (it appears in the tar)
		// link2 points to link1, which hasn't been extracted yet, so link1 is in remaining
		Expect(state[link2Name]).To(BeTrue(), "link2.txt should be in state after pass 1")
		Expect(state[link1Name]).To(BeFalse(), "link1.txt should not be in state after pass 1")
		Expect(state[originalName]).To(BeFalse(), "original.txt should not be in state after pass 1")
		Expect(len(state)).To(Equal(1), "state should only contain one extracted file")

		link2File := filepath.Join(tmpDir, link2Name)
		_, err = os.Lstat(link2File)
		Expect(err).ToNot(HaveOccurred(), "link2.txt should exist on disk after pass 1")

		link1File := filepath.Join(tmpDir, link1Name)
		_, err = os.Lstat(link1File)
		Expect(os.IsNotExist(err)).To(BeTrue(), "link1.txt should not exist on disk after pass 1")

		// remaining should include link1 (the unresolved target of link2)
		Expect(remaining).To(ContainElement(link1Name), "link1.txt should be in remaining after pass 1")
		Expect(len(remaining)).To(Equal(1), "only link1.txt should be in remaining after pass 1")

		// Second pass: Use the remaining list from first pass
		filterPatterns = remaining
		remaining, err = untarOnce(context.Background(), tmpDir, img, filterPatterns, state)
		Expect(err).ToNot(HaveOccurred())

		// After second pass: link1 should be created (it appears earlier in tar)
		// link1 points to original.txt, which hasn't been extracted yet, so original is in remaining
		Expect(state[link2Name]).To(BeTrue(), "link2.txt should still be in state after pass 2")
		Expect(state[link1Name]).To(BeTrue(), "link1.txt should be in state after pass 2")
		Expect(state[originalName]).To(BeFalse(), "original.txt should not be in state after pass 2")
		Expect(len(state)).To(Equal(2), "state should only contain two extracted files")

		_, err = os.Lstat(link1File)
		Expect(err).ToNot(HaveOccurred(), "link1.txt should exist on disk after pass 2")

		originalFile := filepath.Join(tmpDir, originalName)
		_, err = os.Stat(originalFile)
		Expect(os.IsNotExist(err)).To(BeTrue(), "original.txt should not exist on disk after pass 2")

		// remaining should include original.txt (the unresolved target of link1)
		Expect(remaining).To(ContainElement(originalName), "original.txt should be in remaining after pass 2")
		Expect(len(remaining)).To(Equal(1), "only original.txt should be in remaining after pass 2")

		// Third pass: Use the remaining list from second pass
		filterPatterns = remaining
		remaining, err = untarOnce(context.Background(), tmpDir, img, filterPatterns, state)
		Expect(err).ToNot(HaveOccurred())

		// After third pass: all files should be created and resolved
		Expect(state[link2Name]).To(BeTrue(), "link2.txt should be in state after pass 3")
		Expect(state[link1Name]).To(BeTrue(), "link1.txt should be in state after pass 3")
		Expect(state[originalName]).To(BeTrue(), "original.txt should be in state after pass 3")
		Expect(len(state)).To(Equal(3), "state should only contain three extracted files")

		_, err = os.Stat(originalFile)
		Expect(err).ToNot(HaveOccurred(), "original.txt should exist on disk after pass 3")

		// remaining should be empty - all targets resolved
		Expect(len(remaining)).To(Equal(0), "remaining should be empty after pass 3")

		// Verify the complete symlink chain works
		link2Content, err := os.ReadFile(link2File)
		Expect(err).ToNot(HaveOccurred())
		Expect(link2Content).To(Equal(content), "reading through link2 should return original content")
	})
})

// linkType a convenience type just to make the consuming functions more clear.
type linkType = byte

const (
	hardlink linkType = tar.TypeLink
	symlink  linkType = tar.TypeSymlink
)

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
		Typeflag: linkTypeFlag,
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
