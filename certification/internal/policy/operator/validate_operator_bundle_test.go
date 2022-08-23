package operator

import (
	"context"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/operatorsdk"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		fakeEngine          operatorSdk
		imageRef            certification.ImageReference
	)

	BeforeEach(func() {
		stdout := `{
	"passed": true,
	"outputs": null
}`
		stderr := ""
		report := operatorsdk.OperatorSdkBundleValidateReport{
			Stdout:  stdout,
			Stderr:  stderr,
			Passed:  true,
			Outputs: []operatorsdk.OperatorSdkBundleValidateOutput{},
		}
		fakeEngine = FakeOperatorSdk{
			OperatorSdkBVReport: report,
		}

		// mock bundle directory
		tmpDir, err := os.MkdirTemp("", "bundle-metadata-*")
		Expect(err).ToNot(HaveOccurred())

		err = os.Mkdir(filepath.Join(tmpDir, metadataDir), 0o755)
		Expect(err).ToNot(HaveOccurred())

		err = os.WriteFile(filepath.Join(tmpDir, metadataDir, annotationFilename), []byte(annotations), 0o644)
		Expect(err).ToNot(HaveOccurred())

		imageRef.ImageFSPath = tmpDir

		bundleValidateCheck = *NewValidateOperatorBundleCheck(fakeEngine)
	})

	AssertMetaData(&bundleValidateCheck)

	Describe("Operator Bundle Validate", func() {
		Context("When Operator Bundle Validate passes", func() {
			It("Should pass Validate", func() {
				ok, err := bundleValidateCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When Operator Bundle Validate does not Pass", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakeOperatorSdk)
				engine.OperatorSdkBVReport.Passed = false
				engine.OperatorSdkBVReport.Outputs = []operatorsdk.OperatorSdkBundleValidateOutput{
					{Type: "warning", Message: "This is a warning"},
					{Type: "error", Message: "This is an error"},
				}
				fakeEngine = engine
				bundleValidateCheck = *NewValidateOperatorBundleCheck(fakeEngine)
			})
			It("Should not pass Validate", func() {
				ok, err := bundleValidateCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
	Describe("Checking that OperatorSdk errors are handled correctly", func() {
		BeforeEach(func() {
			fakeEngine = BadOperatorSdk{}
			bundleValidateCheck = *NewValidateOperatorBundleCheck(fakeEngine)
		})
		Context("When OperatorSdk throws an error", func() {
			It("should fail Validate and return an error", func() {
				ok, err := bundleValidateCheck.Validate(context.TODO(), certification.ImageReference{ImageURI: "dummy/image"})
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
