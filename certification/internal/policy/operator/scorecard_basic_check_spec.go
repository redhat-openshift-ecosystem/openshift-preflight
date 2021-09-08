package operator

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	log "github.com/sirupsen/logrus"
)

// ScorecardBasicSpecCheck evaluates the image to ensure it passes the operator-sdk
// scorecard check with the basic-check-spec-test suite selected.
type ScorecardBasicSpecCheck struct {
	scorecardCheck
}

const scorecardBasicCheckResult string = "operator_bundle_scorecard_BasicSpecCheck.json"

func NewScorecardBasicSpecCheck(operatorSdkEngine *cli.OperatorSdkEngine) *ScorecardBasicSpecCheck {
	return &ScorecardBasicSpecCheck{
		scorecardCheck{OperatorSdkEngine: *operatorSdkEngine},
	}
}

func (p *ScorecardBasicSpecCheck) Validate(bundleRef certification.ImageReference) (bool, error) {
	log.Debug("Running operator-sdk scorecard check for ", bundleRef.ImageURI)
	selector := []string{"test=basic-check-spec-test"}
	log.Debugf("--selector=%s", selector)
	scorecardReport, err := p.getDataToValidate(bundleRef.ImageFSPath, selector, scorecardBasicCheckResult)
	if err != nil {
		return false, err
	}

	return p.validate(scorecardReport.Items)
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
	return certification.HelpText{
		Message:    "Check ScorecardBasicSpecCheck encountered an error. Please review the artifacts/operator_bundle_scorecard_BasicSpecCheck.json file for more information.",
		Suggestion: "Make sure that all CRs have a spec block",
	}
}
