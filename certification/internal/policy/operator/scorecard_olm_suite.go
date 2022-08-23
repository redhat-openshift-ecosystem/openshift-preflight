package operator

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"

	log "github.com/sirupsen/logrus"
)

var _ certification.Check = &ScorecardOlmSuiteCheck{}

// ScorecardOlmSuiteCheck evaluates the image to ensure it passes the operator-sdk
// scorecard check with the olm suite selected.
type ScorecardOlmSuiteCheck struct {
	scorecardCheck
	fatalError bool
}

const scorecardOlmSuiteResult string = "operator_bundle_scorecard_OlmSuiteCheck.json"

func NewScorecardOlmSuiteCheck(operatorSdk operatorSdk, ns, sa, kubeconfig, waittime string) *ScorecardOlmSuiteCheck {
	return &ScorecardOlmSuiteCheck{
		scorecardCheck: scorecardCheck{
			OperatorSdk:    operatorSdk,
			namespace:      ns,
			serviceAccount: sa,
			kubeconfig:     kubeconfig,
			waitTime:       waittime,
		},
		fatalError: false,
	}
}

func (p *ScorecardOlmSuiteCheck) Validate(ctx context.Context, bundleRef certification.ImageReference) (bool, error) {
	log.Trace("Running operator-sdk scorecard Check for ", bundleRef.ImageURI)
	selector := []string{"suite=olm"}
	log.Tracef("--selector=%s", selector)
	scorecardReport, err := p.getDataToValidate(ctx, bundleRef.ImageFSPath, selector, scorecardOlmSuiteResult)
	if err != nil {
		p.fatalError = true
		return false, fmt.Errorf("%v", err)
	}

	return p.validate(ctx, scorecardReport.Items)
}

func (p *ScorecardOlmSuiteCheck) Name() string {
	return "ScorecardOlmSuiteCheck"
}

func (p *ScorecardOlmSuiteCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Operator-sdk scorecard OLM Test Suite Check",
		Level:            "best",
		KnowledgeBaseURL: "https://sdk.operatorframework.io/docs/advanced-topics/scorecard/scorecard/#overview", // Placeholder
		CheckURL:         "https://sdk.operatorframework.io/docs/advanced-topics/scorecard/scorecard/#olm-test-suite",
	}
}

func (p *ScorecardOlmSuiteCheck) Help() certification.HelpText {
	if p.fatalError {
		return certification.HelpText{
			Message: "There was a fatal error while running operator-sdk scorecard tests. " +
				"Please see the preflight log for details. If necessary, set logging to be more verbose.",
			Suggestion: "If the logs are showing a context timeout, try setting wait time to a higher value.",
		}
	}
	return certification.HelpText{
		Message: "Check ScorecardOlmSuiteCheck encountered an error. Please review the " +
			artifacts.Path() + "/" + scorecardOlmSuiteResult + " file for more information.",
		Suggestion: "See scorecard output for details, artifacts/operator_bundle_scorecard_OlmSuiteCheck.json",
	}
}
