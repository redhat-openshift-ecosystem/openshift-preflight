package operator

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	log "github.com/sirupsen/logrus"
)

func TestOperator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator Suite")
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.TraceLevel)
}

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
