package operator

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/bundle"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"

	"github.com/go-logr/logr"
)

var _ check.Check = &ScorecardOlmSuiteCheck{}

// ScorecardOlmSuiteCheck evaluates the image to ensure it passes the operator-sdk
// scorecard check with the olm suite selected.
type ScorecardOlmSuiteCheck struct {
	scorecardCheck
	fatalError bool
}

const scorecardOlmSuiteResult string = "operator_bundle_scorecard_OlmSuiteCheck.json"

func NewScorecardOlmSuiteCheck(operatorSdk operatorSdk, ns, sa string, kubeconfig []byte, waittime string) *ScorecardOlmSuiteCheck {
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

func (p *ScorecardOlmSuiteCheck) Validate(ctx context.Context, bundleRef image.ImageReference) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(log.TRC).Info("running operator-sdk scorecard check", "image", bundleRef.ImageURI)

	selector := []string{"suite=olm"}
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

func (p *ScorecardOlmSuiteCheck) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Operator-sdk scorecard OLM Test Suite Check",
		Level:            "best",
		KnowledgeBaseURL: "https://sdk.operatorframework.io/docs/testing-operators/scorecard/#overview",
		CheckURL:         "https://sdk.operatorframework.io/docs/testing-operators/scorecard/#olm-test-suite",
	}
}

func (p *ScorecardOlmSuiteCheck) Help() check.HelpText {
	if p.fatalError {
		return check.HelpText{
			Message: "There was a fatal error while running operator-sdk scorecard tests. " +
				"Please see the preflight log for details. If necessary, set logging to be more verbose.",
			Suggestion: "If the logs are showing a context timeout, try setting wait time to a higher value.",
		}
	}
	return check.HelpText{
		Message:    "Check ScorecardOlmSuiteCheck encountered an error. Please review the " + scorecardOlmSuiteResult + " file in your execution artifacts for more information.",
		Suggestion: "See scorecard output for details, artifacts/operator_bundle_scorecard_OlmSuiteCheck.json",
	}
}

func (p *ScorecardOlmSuiteCheck) RequiredFilePatterns() []string {
	return bundle.BundleFiles
}
