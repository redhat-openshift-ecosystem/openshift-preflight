package shell

import (
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

type scorecardCheck struct{}

func (p *scorecardCheck) validate(items []cli.OperatorSdkScorecardItem) (bool, error) {
	foundTestFailed := false

	if len(items) == 0 {
		log.Warn("Did not receive any test result information from scorecard output")
	}
	for _, item := range items {
		for _, result := range item.Status.Results {
			if strings.Contains(result.State, "fail") {
				foundTestFailed = true
			}
		}
	}
	return !foundTestFailed, nil
}

func (p *scorecardCheck) getDataToValidate(bundleImage string, selector []string, resultFile string) (*cli.OperatorSdkScorecardReport, error) {
	opts := cli.OperatorSdkScorecardOptions{
		LogLevel:     "warning",
		OutputFormat: "json",
		Selector:     selector,
		ResultFile:   resultFile,
	}
	return operatorSdkEngine.Scorecard(bundleImage, opts)
}
