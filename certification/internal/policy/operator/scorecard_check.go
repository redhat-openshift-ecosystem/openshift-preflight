package operator

import (
	"context"
	"fmt"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/operatorsdk"

	log "github.com/sirupsen/logrus"
)

type scorecardCheck struct {
	OperatorSdk operatorSdk

	namespace      string
	serviceAccount string
	kubeconfig     string
	waitTime       string
}

//nolint:unparam // ctx is unused. Keep for future use.
func (p *scorecardCheck) validate(ctx context.Context, items []operatorsdk.OperatorSdkScorecardItem) (bool, error) {
	foundTestFailed := false
	var err error

	if len(items) == 0 {
		log.Warn("Did not receive any test result information from scorecard output")
	}
	for _, item := range items {
		for _, result := range item.Status.Results {
			if strings.Contains(result.State, "fail") {
				err = fmt.Errorf("result log: %s", result.Log)
				foundTestFailed = true
			}
		}
	}
	return !foundTestFailed, err
}

func (p *scorecardCheck) getDataToValidate(ctx context.Context, bundleImage string, selector []string, resultFile string) (*operatorsdk.OperatorSdkScorecardReport, error) {
	opts := operatorsdk.OperatorSdkScorecardOptions{
		OutputFormat:   "json",
		Selector:       selector,
		ResultFile:     resultFile,
		Kubeconfig:     p.kubeconfig,
		Namespace:      p.namespace,
		ServiceAccount: p.serviceAccount,
		Verbose:        true,
		WaitTime:       fmt.Sprintf("%ss", p.waitTime),
	}
	result, err := p.OperatorSdk.Scorecard(ctx, bundleImage, opts)
	if err != nil {
		return result, fmt.Errorf("%v", err)
	}
	return result, nil
}
