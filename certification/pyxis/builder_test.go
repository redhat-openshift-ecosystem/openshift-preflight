package pyxis

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pyxis Builder tests", func() {
	var tmpdir string
	var err error

	BeforeEach(func() {
		// create tmpdir to receive extracted fs
		tmpdir, err = os.MkdirTemp(os.TempDir(), "builder-test-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(tmpdir)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("When reading a file with ReadFile", func() {
		It("should be the same size as file.Stat().Size()", func() {
			f := filepath.Join(tmpdir, "test.txt")
			os.WriteFile(f, []byte("\tHello world!\n"), 0o0755)

			file, err := os.Open(f)
			Expect(err).ToNot(HaveOccurred())

			info, err := file.Stat()
			Expect(err).ToNot(HaveOccurred())

			fileBytes, err := os.ReadFile(f)
			Expect(err).ToNot(HaveOccurred())
			Expect(int64(len(fileBytes))).To(Equal(info.Size()))
		})
	})
})
