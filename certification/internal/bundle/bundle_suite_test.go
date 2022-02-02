package bundle

import "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"

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
