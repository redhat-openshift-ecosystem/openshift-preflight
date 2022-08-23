package runtime

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Result Writers", func() {
	var resultWriterTestDir string

	BeforeEach(func() {
		// instantiate err to make sure we can equal-assign in the following line.
		var err error
		resultWriterTestDir, err = os.MkdirTemp(os.TempDir(), "rw-test-*")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(resultWriterTestDir)).ToNot(BeZero())
	})

	AfterEach(func() {
		err := os.RemoveAll(resultWriterTestDir)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("File-based Result Writer", func() {
		var rw *ResultWriterFile

		BeforeEach(func() {
			rw = &ResultWriterFile{}
		})

		Context("When using the file-based result writer", func() {
			It("should open, be writeable, and close successfully", func() {
				p := path.Join(resultWriterTestDir, "foo.txt")
				f, err := rw.OpenFile(p)
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()
				Expect(p).To(BeAnExistingFile())

				toWrite := []byte("testing")
				_, err = f.Write(toWrite)
				Expect(err).ToNot(HaveOccurred())

				contents, err := os.ReadFile(p)
				Expect(err).ToNot(HaveOccurred())
				Expect(contents).To(Equal(toWrite))
			})
		})
	})
})
