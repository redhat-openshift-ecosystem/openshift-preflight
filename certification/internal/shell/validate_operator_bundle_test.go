package shell

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

var _ = Describe("BundleValidateCheck", func() {
	var (
		bundleValidateCheck ValidateOperatorBundleCheck
		fakeEngine          cli.OperatorSdkEngine
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
		operatorSdkEngine = fakeEngine
	})
	Describe("Operator Bundle Validate", func() {
		Context("When Operator Bundle Validate passes", func() {
			It("Should pass Validate", func() {
				ok, err := bundleValidateCheck.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When Operator Bundle Validate does not Pass", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakeOperatorSdkEngine)
				engine.OperatorSdkBVReport.Passed = false
				engine.OperatorSdkBVReport.Outputs = []cli.OperatorSdkBundleValidateOutput{
					cli.OperatorSdkBundleValidateOutput{Type: "warning", Message: "This is a warning"},
					cli.OperatorSdkBundleValidateOutput{Type: "error", Message: "This is an error"},
				}
				operatorSdkEngine = engine
			})
			It("Should not pass Validate", func() {
				ok, err := bundleValidateCheck.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
	Describe("Checking that OperatorSdkEngine errors are handled correctly", func() {
		BeforeEach(func() {
			fakeEngine = BadOperatorSdkEngine{}
			operatorSdkEngine = fakeEngine
		})
		Context("When OperatorSdk throws an error", func() {
			It("should fail Validate and return an error", func() {
				ok, err := bundleValidateCheck.Validate("dummy/image")
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
