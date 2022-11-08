package artifacts

import (
	"bytes"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Filesystem Artifact Writer", func() {
	var tempdir string
	BeforeEach(func() {
		var err error
		tempdir, err = os.MkdirTemp(os.TempDir(), "fs-artifact-writer-*")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(tempdir)).ToNot(BeZero())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempdir)).To(Succeed())
	})

	Context("With a Filesystem Artifact Writer configured with a provided Artifact Directory", func() {
		var aw *FilesystemWriter
		filename := "testfile.txt"
		contents := []byte("testcontents")

		BeforeEach(func() {
			var err error
			aw, err = NewFilesystemWriter(WithDirectory(tempdir))
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should write to the input filename in the configured directory with the provided content", func() {
			fullpath, err := aw.WriteFile(filename, bytes.NewBuffer(contents))

			Expect(err).ToNot(HaveOccurred())
			Expect(fullpath).To(Equal(filepath.Join(tempdir, filename)))

			_, err = os.Stat(fullpath)
			Expect(err).ToNot(HaveOccurred())

			readin, err := os.ReadFile(fullpath)
			Expect(err).ToNot(HaveOccurred())
			Expect(readin).To(Equal(contents))
		})
	})
})
