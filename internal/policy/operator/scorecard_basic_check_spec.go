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

var _ check.Check = &ScorecardBasicSpecCheck{}

// ScorecardBasicSpecCheck evaluates the image to ensure it passes the operator-sdk
// scorecard check with the basic-check-spec-test suite selected.
type ScorecardBasicSpecCheck struct {
	scorecardCheck
	fatalError bool
}

const scorecardBasicCheckResult string = "operator_bundle_scorecard_BasicSpecCheck.json"

func NewScorecardBasicSpecCheck(operatorSdk operatorSdk, ns, sa string, kubeconfig []byte, waittime string) *ScorecardBasicSpecCheck {
	return &ScorecardBasicSpecCheck{
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

func (p *ScorecardBasicSpecCheck) Validate(ctx context.Context, bundleRef image.ImageReference) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(log.TRC).Info("running operator-sdk scorecard check", "image", bundleRef.ImageURI)

	selector := []string{"test=basic-check-spec-test"}
	scorecardReport, err := p.getDataToValidate(ctx, bundleRef.ImageFSPath, selector, scorecardBasicCheckResult)
	if err != nil {
		p.fatalError = true
		return false, fmt.Errorf("%v", err)
	}

	return p.validate(ctx, scorecardReport.Items)
}

func (p *ScorecardBasicSpecCheck) Name() string {
	return "ScorecardBasicSpecCheck"
}

func (p *ScorecardBasicSpecCheck) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Check to make sure that all CRs have a spec block.",
		Level:            "best",
		KnowledgeBaseURL: "https://sdk.operatorframework.io/docs/testing-operators/scorecard/#overview",
		CheckURL:         "https://sdk.operatorframework.io/docs/testing-operators/scorecard/#basic-test-suite",
	}
}

func (p *ScorecardBasicSpecCheck) Help() check.HelpText {
	if p.fatalError {
		return check.HelpText{
			Message: "There was a fatal error while running operator-sdk scorecard tests. " +
				"Please see the preflight log for details. If necessary, set logging to be more verbose.",
			Suggestion: "If the logs are showing a context timeout, try setting wait time to a higher value.",
		}
	}
	return check.HelpText{
		Message:    "Check ScorecardBasicSpecCheck encountered an error. Please review the " + scorecardBasicCheckResult + " file in your execution artifacts for more information.",
		Suggestion: "Make sure that all CRs have a spec block",
	}
}

func (p *ScorecardBasicSpecCheck) RequiredFilePatterns() []string {
	return bundle.BundleFiles
}
