package operator

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"

	log "github.com/sirupsen/logrus"
)

var _ certification.Check = &ScorecardBasicSpecCheck{}

// ScorecardBasicSpecCheck evaluates the image to ensure it passes the operator-sdk
// scorecard check with the basic-check-spec-test suite selected.
type ScorecardBasicSpecCheck struct {
	scorecardCheck
	fatalError bool
}

const scorecardBasicCheckResult string = "operator_bundle_scorecard_BasicSpecCheck.json"

func NewScorecardBasicSpecCheck(operatorSdk operatorSdk, ns, sa, kubeconfig, waittime string) *ScorecardBasicSpecCheck {
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

func (p *ScorecardBasicSpecCheck) Validate(ctx context.Context, bundleRef certification.ImageReference) (bool, error) {
	log.Trace("Running operator-sdk scorecard check for ", bundleRef.ImageURI)
	selector := []string{"test=basic-check-spec-test"}
	log.Tracef("--selector=%s", selector)
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

func (p *ScorecardBasicSpecCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Check to make sure that all CRs have a spec block.",
		Level:            "best",
		KnowledgeBaseURL: "https://sdk.operatorframework.io/docs/advanced-topics/scorecard/scorecard/#overview", // Placeholder
		CheckURL:         "https://sdk.operatorframework.io/docs/advanced-topics/scorecard/scorecard/#basic-test-suite",
	}
}

func (p *ScorecardBasicSpecCheck) Help() certification.HelpText {
	if p.fatalError {
		return certification.HelpText{
			Message: "There was a fatal error while running operator-sdk scorecard tests. " +
				"Please see the preflight log for details. If necessary, set logging to be more verbose.",
			Suggestion: "If the logs are showing a context timeout, try setting wait time to a higher value.",
		}
	}
	return certification.HelpText{
		Message: "Check ScorecardBasicSpecCheck encountered an error. Please review the " +
			artifacts.Path() + "/" + scorecardBasicCheckResult + " file for more information.",
		Suggestion: "Make sure that all CRs have a spec block",
	}
}
