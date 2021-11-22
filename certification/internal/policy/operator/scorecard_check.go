package operator

import (
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
	kubeconfig := os.Getenv("KUBECONFIG")
	opts := cli.OperatorSdkScorecardOptions{
		OutputFormat:   "json",
		Selector:       selector,
		ResultFile:     resultFile,
		Kubeconfig:     kubeconfig,
		Namespace:      namespace,
		ServiceAccount: serviceAccount,
		Verbose:        true,
		WaitTime:       "240s",
	}
	return p.OperatorSdkEngine.Scorecard(bundleImage, opts)
}
