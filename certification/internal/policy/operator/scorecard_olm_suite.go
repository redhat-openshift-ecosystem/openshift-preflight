package operator

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	log "github.com/sirupsen/logrus"
)

// ScorecardOlmSuiteCheck evaluates the image to ensure it passes the operator-sdk
// scorecard check with the olm suite selected.
type ScorecardOlmSuiteCheck struct {
	scorecardCheck
}

const scorecardOlmSuiteResult string = "operator_bundle_scorecard_OlmSuiteCheck.json"

func NewScorecardOlmSuiteCheck(operatorSdkEngine *cli.OperatorSdkEngine) *ScorecardOlmSuiteCheck {
	return &ScorecardOlmSuiteCheck{
		scorecardCheck{OperatorSdkEngine: *operatorSdkEngine},
	}
}

func (p *ScorecardOlmSuiteCheck) Validate(bundleRef certification.ImageReference) (bool, error) {
	log.Debug("Running operator-sdk scorecard Check for ", bundleRef.ImageURI)
	selector := []string{"suite=olm"}
	log.Debugf("--selector=%s", selector)
	scorecardReport, err := p.getDataToValidate(bundleRef.ImageFSPath, selector, scorecardOlmSuiteResult)
	if err != nil {
		return false, err
	}

	return p.validate(scorecardReport.Items)
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
	return certification.HelpText{
		Message:    "Check ScorecardOlmSuiteCheck encountered an error. Please review the artifacts/operator_bundle_scorecard_OlmSuiteCheck.json file for more information.",
		Suggestion: "See scorecard output for details, artifacts/operator_bundle_scorecard_OlmSuiteCheck.json",
	}
}
