package artifacts

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artifacts Package Utility Functions", func() {
	Context("When resolving a path of an input directory", func() {
		It("Should return the exact input when it starts with the \"/\" character", func() {
			in := "/foo"
			actual := resolveFullPath(in)
			Expect(actual).To(Equal(in))
		})

		It("Should return the input relative to the current working directory when the first character is not \"/\"", func() {
			in := "foo"
			cwd, err := os.Getwd()
			Expect(err).ToNot(HaveOccurred())
			actual := resolveFullPath(in)
			Expect(actual).To(Equal(filepath.Join(cwd, in)))
		})
	})
})
