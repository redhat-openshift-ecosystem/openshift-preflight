package operator

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/migration"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

type FakeOperatorSdkEngine struct {
	OperatorSdkReport   cli.OperatorSdkScorecardReport
	OperatorSdkBVReport cli.OperatorSdkBundleValidateReport
}

func (f FakeOperatorSdkEngine) BundleValidate(image string, opts cli.OperatorSdkBundleValidateOptions) (*cli.OperatorSdkBundleValidateReport, error) {
	return &f.OperatorSdkBVReport, nil
}

func (f FakeOperatorSdkEngine) Scorecard(image string, opts cli.OperatorSdkScorecardOptions) (*cli.OperatorSdkScorecardReport, error) {
	return &f.OperatorSdkReport, nil
}

type BadOperatorSdkEngine struct{}

func (bose BadOperatorSdkEngine) Scorecard(bundleImage string, opts cli.OperatorSdkScorecardOptions) (*cli.OperatorSdkScorecardReport, error) {
	operatorSdkReport := cli.OperatorSdkScorecardReport{
		Stdout: "Bad Stdout",
		Stderr: "Bad Stderr",
		Items:  []cli.OperatorSdkScorecardItem{},
	}
	return &operatorSdkReport, errors.New("the Operator Sdk Scorecard has failed")
}

func (bose BadOperatorSdkEngine) BundleValidate(bundleImage string, opts cli.OperatorSdkBundleValidateOptions) (*cli.OperatorSdkBundleValidateReport, error) {
	operatorSdkReport := cli.OperatorSdkBundleValidateReport{
		Stdout:  "Bad Stdout",
		Stderr:  "Bad Stderr",
		Passed:  false,
		Outputs: []cli.OperatorSdkBundleValidateOutput{},
	}
	return &operatorSdkReport, errors.New("the Operator Sdk Bundle Validate has failed")
}

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
		bundleValidateCheck = *NewValidateOperatorBundleCheck(&fakeEngine)
	})
	Describe("Operator Bundle Validate", func() {
		Context("When Operator Bundle Validate passes", func() {
			It("Should pass Validate", func() {
				ok, err := bundleValidateCheck.Validate(migration.ImageToImageReference("dummy/image"))
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
				ok, err := bundleValidateCheck.Validate(migration.ImageToImageReference("dummy/image"))
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
				ok, err := bundleValidateCheck.Validate(migration.ImageToImageReference("dummy/image"))
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
