package bundle

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
)

func TestBundle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bundle Utils Suite")
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

// In order to test some negative paths, this io.Reader will just throw an error
type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}
