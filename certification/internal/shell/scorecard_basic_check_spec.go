package shell

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

type ScorecardBasicSpecCheck struct {
	scorecardCheck
}

const scorecardBasicCheckResult string = "operator_bundle_scorecard_BasicSpecCheck.json"

func (p *ScorecardBasicSpecCheck) Validate(bundleImage string) (bool, error) {
	log.Debug("Running operator-sdk scorecard check for ", bundleImage)
	selector := []string{"test=basic-check-spec-test"}
	log.Debugf("--selector=%s", selector)
	scorecardReport, err := p.getDataToValidate(bundleImage, selector, scorecardBasicCheckResult)
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
		Message:    "Operator-sdk scorecard basic spec check failed.",
		Suggestion: "Make sure that all CRs have a spec block",
	}
}
