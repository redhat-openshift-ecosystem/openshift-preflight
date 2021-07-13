package shell

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

type ScorecardOlmSuiteCheck struct {
	scorecardCheck
}

const scorecardOlmSuiteResult string = "operator_bundle_scorecard_OlmSuiteCheck.json"

func (p *ScorecardOlmSuiteCheck) Validate(bundleImage string) (bool, error) {
	log.Debug("Running operator-sdk scorecard Check for ", bundleImage)
	selector := []string{"suite=olm"}
	log.Debugf("--selector=%s", selector)
	scorecardReport, err := p.getDataToValidate(bundleImage, selector, scorecardOlmSuiteResult)
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
		Description:      "OLM Test Suite Check",
		Level:            "best",
		KnowledgeBaseURL: "https://sdk.operatorframework.io/docs/advanced-topics/scorecard/scorecard/#overview", // Placeholder
		CheckURL:         "https://sdk.operatorframework.io/docs/advanced-topics/scorecard/scorecard/#olm-test-suite",
	}
}

func (p *ScorecardOlmSuiteCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Operator-sdk scorecard OLM Test Suite. One or more checks failed.",
		Suggestion: "See scorecard output for details, artifacts/operator_bundle_scorecard_OlmSuiteCheck.json",
	}
}
