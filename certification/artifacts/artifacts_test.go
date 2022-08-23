package artifacts

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artifacts package utility functions", func() {
	BeforeEach(func() {
		// clean up the artifacts directory that might be created
		// in each of these tests. This removes only the artifacts
		// directory value, not the temporary dir established in
		// BeforeSuite.

		Expect(os.RemoveAll(ads)).To(Succeed())
	})

	Context("With an artifacts directory provided via configuration", func() {
		It("should write the provided contents to the file with the provided name", func() {
			contents := "hello world"
			fullFilePath, err := WriteFile("test.txt", contents)
			Expect(err).ToNot(HaveOccurred())

			bcontents, err := os.ReadFile(fullFilePath)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(bcontents)).To(Equal(contents))
		})

		It("should be created when explicitly calling the createArtifactsDir function", func() {
			createdDir, err := createArtifactsDir(artifactsPkgTestBaseDir)
			Expect(err).ToNot(HaveOccurred())
			dirInfo, err := os.Stat(createdDir)
			// if it doesn't exist, this error will capture it.
			Expect(err).ToNot(HaveOccurred())
			Expect(dirInfo.IsDir()).To(BeTrue())
		})

		It("should be created by the exported WriteFile() function", func() {
			createdDir := Path()
			Expect(WriteFile("test.txt", "foo"))
			dirInfo, err := os.Stat(createdDir)
			// if it doesn't exist, this error will capture it.
			Expect(err).ToNot(HaveOccurred())
			Expect(dirInfo.IsDir()).To(BeTrue())
		})
	})
})
