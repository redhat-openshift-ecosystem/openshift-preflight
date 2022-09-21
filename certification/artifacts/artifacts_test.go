package artifacts

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artifacts package utility functions", func() {
	BeforeEach(func() {
		// clean up the artifacts directory that might be created
		// in each of these tests. This removes only the artifacts
		// directory value, not the temporary dir established in
		// BeforeSuite.

		Expect(os.RemoveAll(Path())).To(Succeed())
	})

	Context("With an artifacts directory provided via configuration", func() {
		It("should write the provided contents to the file with the provided name", func() {
			contents := "hello world"
			fullFilePath, err := WriteFile("test.txt", strings.NewReader(contents))
			Expect(err).ToNot(HaveOccurred())

			bcontents, err := os.ReadFile(fullFilePath)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(bcontents)).To(Equal(contents))
		})
		It("should be created by the exported WriteFile() function", func() {
			createdDir := Path()
			Expect(WriteFile("test.txt", strings.NewReader("foo")))
			dirInfo, err := os.Stat(createdDir)
			// if it doesn't exist, this error will capture it.
			Expect(err).ToNot(HaveOccurred())
			Expect(dirInfo.IsDir()).To(BeTrue())
		})
	})
})
