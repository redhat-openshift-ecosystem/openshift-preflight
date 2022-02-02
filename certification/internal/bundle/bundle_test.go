package bundle

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

var _ = Describe("BundleValidateCheck", func() {
	const (
		metadataDir        = "metadata"
		annotationFilename = "annotations.yaml"
		annotations        = `annotations:
  com.redhat.openshift.versions: "v4.6-v4.9"
  operators.operatorframework.io.bundle.package.v1: testPackage
  operators.operatorframework.io.bundle.channel.default.v1: testChannel
`
	)

	var (
		// fakeEngine cli.OperatorSdkEngine
		imageRef certification.ImageReference
	)

	Describe("While ensuring that container util is working", func() {

		// tests: extractAnnotationsBytes
		Context("with an annotations yaml data read from disk", func() {
			Context("with the correct format", func() {
				data := []byte("annotations:\n foo: bar")

				It("should properly marshal to a map[string]string", func() {
					annotations, err := ExtractAnnotationsBytes(data)
					Expect(err).ToNot(HaveOccurred())
					Expect(annotations["foo"]).To(Equal("bar"))
				})
			})

			Context("containing no data read in from the yaml file", func() {
				data := []byte{}

				It("should return an error", func() {
					_, err := ExtractAnnotationsBytes(data)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("containing malformed or unexpected data", func() {
				data := []byte(`malformed`)

				It("should return an error", func() {
					_, err := ExtractAnnotationsBytes(data)
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})

	BeforeEach(func() {
		// 		stdout := `{
		// 	"passed": true,
		// 	"outputs": null
		// }`
		// 		stderr := ""
		// report := cli.OperatorSdkBundleValidateReport{
		// 	Stdout:  stdout,
		// 	Stderr:  stderr,
		// 	Passed:  true,
		// 	Outputs: []cli.OperatorSdkBundleValidateOutput{},
		// }
		// fakeEngine = FakeOperatorSdkEngine{
		// 	OperatorSdkBVReport: report,
		// }

		// mock bundle directory
		tmpDir, err := os.MkdirTemp("", "bundle-metadata-*")
		Expect(err).ToNot(HaveOccurred())

		err = os.Mkdir(filepath.Join(tmpDir, metadataDir), 0755)
		Expect(err).ToNot(HaveOccurred())

		err = os.WriteFile(filepath.Join(tmpDir, metadataDir, annotationFilename), []byte(annotations), 0644)
		Expect(err).ToNot(HaveOccurred())

		imageRef.ImageFSPath = tmpDir
	})

	DescribeTable("Image Registry validation",
		func(versions string, expected bool) {
			ok := isTarget49OrGreater(versions)
			Expect(ok).To(Equal(expected))
		},
		Entry("range 4.6 to 4.8", "v4.6-v4.8", false),
		Entry("exactly 4.8", "=v4.8", false),
		Entry("exactly 4.9", "=v4.9", true),
		Entry("range 4.6 to 4.9", "v4.6-v4.9", true),
		Entry(">= 4.8", "v4.8", true),
		Entry(">= 4.9", "v4.9", true),
	)
	AfterEach(func() {
		err := os.RemoveAll(imageRef.ImageFSPath)
		Expect(err).ToNot(HaveOccurred())
	})
})
