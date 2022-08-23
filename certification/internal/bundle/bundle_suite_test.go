package bundle

import (
	"context"
	"errors"
	"testing"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/operatorsdk"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

func TestBundle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bundle Utils Suite")
}

type FakeOperatorSdk struct {
	OperatorSdkReport   operatorsdk.OperatorSdkScorecardReport
	OperatorSdkBVReport operatorsdk.OperatorSdkBundleValidateReport
}

func (f FakeOperatorSdk) BundleValidate(ctx context.Context, image string, opts operatorsdk.OperatorSdkBundleValidateOptions) (*operatorsdk.OperatorSdkBundleValidateReport, error) {
	return &f.OperatorSdkBVReport, nil
}

// In order to test some negative paths, this io.Reader will just throw an error
type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}
