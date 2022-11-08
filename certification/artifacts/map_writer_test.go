package artifacts

import (
	"bytes"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Map Artifact Writer", func() {
	Context("With a Map Artifact Writer", func() {
		var aw *MapWriter
		filename := "testfile.txt"
		contents := []byte("testcontents")

		BeforeEach(func() {
			var err error
			aw, err = NewMapWriter()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should write to the input filename in the map with the provided content", func() {
			_, err := aw.WriteFile(filename, bytes.NewBuffer(contents))

			Expect(err).ToNot(HaveOccurred())

			written := aw.Files()
			val, ok := written[filename]
			Expect(ok).To(BeTrue())
			readin, err := io.ReadAll(val)
			Expect(err).ToNot(HaveOccurred())
			Expect(readin).To(Equal(contents))
		})

		It("Should reject subsequent writes to the same file", func() {
			_, err := aw.WriteFile(filename, bytes.NewBuffer(contents))
			Expect(err).ToNot(HaveOccurred())

			_, err = aw.WriteFile(filename, bytes.NewBuffer([]byte("rejected")))
			Expect(err).To(Equal(ErrFileAlreadyExists))
		})
	})
})
