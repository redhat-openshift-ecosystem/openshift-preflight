package operator

import (
	"fmt"
	"os"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type scorecardCheck struct {
	OperatorSdkEngine cli.OperatorSdkEngine
}

func (p *scorecardCheck) validate(items []cli.OperatorSdkScorecardItem) (bool, error) {
	foundTestFailed := false

	if len(items) == 0 {
		log.Warn("Did not receive any test result information from scorecard output")
	}
	for _, item := range items {
		for _, result := range item.Status.Results {
			if strings.Contains(result.State, "fail") {
				log.Error(result.Log)
				foundTestFailed = true
			}
		}
	}
	return !foundTestFailed, nil
}

func (p *scorecardCheck) getDataToValidate(bundleImage string, selector []string, resultFile string) (*cli.OperatorSdkScorecardReport, error) {
	namespace := viper.GetString("namespace")
	serviceAccount := viper.GetString("serviceaccount")
	waitTime := viper.GetString("scorecard_wait_time")
	kubeconfig := os.Getenv("KUBECONFIG")

	opts := cli.OperatorSdkScorecardOptions{
		OutputFormat:   "json",
		Selector:       selector,
		ResultFile:     resultFile,
		Kubeconfig:     kubeconfig,
		Namespace:      namespace,
		ServiceAccount: serviceAccount,
		Verbose:        true,
		WaitTime:       fmt.Sprintf("%ss", waitTime),
	}
	return p.OperatorSdkEngine.Scorecard(bundleImage, opts)
}
