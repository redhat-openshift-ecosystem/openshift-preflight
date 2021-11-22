package operator

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
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
		bundleValidateCheck ValidateOperatorBundleCheck
		fakeEngine          cli.OperatorSdkEngine
		imageRef            certification.ImageReference
	)

	BeforeEach(func() {
		stdout := `{
	"passed": true,
	"outputs": null
}`
		stderr := ""
		report := cli.OperatorSdkBundleValidateReport{
			Stdout:  stdout,
			Stderr:  stderr,
			Passed:  true,
			Outputs: []cli.OperatorSdkBundleValidateOutput{},
		}
		fakeEngine = FakeOperatorSdkEngine{
			OperatorSdkBVReport: report,
		}

		// mock bundle directory
		tmpDir, err := os.MkdirTemp("", "bundle-metadata-*")
		Expect(err).ToNot(HaveOccurred())

		err = os.Mkdir(filepath.Join(tmpDir, metadataDir), 0755)
		Expect(err).ToNot(HaveOccurred())

		err = os.WriteFile(filepath.Join(tmpDir, metadataDir, annotationFilename), []byte(annotations), 0644)
		Expect(err).ToNot(HaveOccurred())

		imageRef.ImageFSPath = tmpDir

		bundleValidateCheck = *NewValidateOperatorBundleCheck(&fakeEngine)
	})
	Describe("Operator Bundle Validate", func() {
		Context("When Operator Bundle Validate passes", func() {
			It("Should pass Validate", func() {
				ok, err := bundleValidateCheck.Validate(imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When Operator Bundle Validate does not Pass", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakeOperatorSdkEngine)
				engine.OperatorSdkBVReport.Passed = false
				engine.OperatorSdkBVReport.Outputs = []cli.OperatorSdkBundleValidateOutput{
					{Type: "warning", Message: "This is a warning"},
					{Type: "error", Message: "This is an error"},
				}
				fakeEngine = engine
				bundleValidateCheck = *NewValidateOperatorBundleCheck(&fakeEngine)
			})
			It("Should not pass Validate", func() {
				ok, err := bundleValidateCheck.Validate(imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
	Describe("Checking that OperatorSdkEngine errors are handled correctly", func() {
		BeforeEach(func() {
			fakeEngine = BadOperatorSdkEngine{}
			bundleValidateCheck = *NewValidateOperatorBundleCheck(&fakeEngine)
		})
		Context("When OperatorSdk throws an error", func() {
			It("should fail Validate and return an error", func() {
				ok, err := bundleValidateCheck.Validate(certification.ImageReference{ImageURI: "dummy/image"})
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
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
