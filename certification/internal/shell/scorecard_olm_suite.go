package shell

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

type ScorecardOlmSuiteCheck struct{}

const scorecardOlmSuiteResult string = "operator_bundle_scorecard_OlmSuiteCheck.json"

func (p *ScorecardOlmSuiteCheck) Validate(bundleImage string) (bool, error) {

	currentDir, err := os.Getwd()
	if err != nil {
		log.Error("unable to get current directory: ", err)
		return false, err
	}

	artifactsDir := filepath.Join(currentDir, "/artifacts")

	err = os.MkdirAll(artifactsDir, 0777)
	if err != nil {
		log.Error("unable to create artifactsDir: ", err)
		return false, err
	}

	log.Debug("Running operator-sdk scorecard Check for ", bundleImage)
	log.Debug("--selector=suite=olm")
	stdouterr, err := exec.Command("operator-sdk", "scorecard",
		"--selector=suite=olm",
		"--output", "json", bundleImage).CombinedOutput()

	scorecardFile := filepath.Join(artifactsDir, "/", scorecardOlmSuiteResult)

	err = ioutil.WriteFile(scorecardFile, stdouterr, 0644)
	if err != nil {
		log.Error("unable to copy result to /artifacts subdir: ", err)
		return false, err
	}

	// we must send gojq a interface{}, so we have to convert our inspect output to that type
	var inspectData interface{}

	err = json.Unmarshal(stdouterr, &inspectData)
	if err != nil {
		log.Error("unable to parse scorecard json output")
		log.Debug("error Unmarshaling scorecard json output: ", err)
		log.Trace("failure in attempt to convert the raw bytes from `operator-sdk scorecard` to a interface{}")
		return false, err
	}

	query, err := gojq.Parse(".items[].status.results[] | .name, .state")
	if err != nil {
		log.Error("unable to parse scorecard json output")
		log.Debug("unable to parse :", err)
		return false, err
	}

	// gojq expects us to iterate in the event that our query returned multiple matching values, but we only expect one.
	iter := query.Run(inspectData)

	foundTestFailed := false

	log.Info("scorecard outuput")

	for {
		v, ok := iter.Next()
		if !ok {
			log.Warn("Did not receive any test result information when parsing scorecard output.")
			break
		}
		if err, ok := v.(error); ok {
			log.Error("unable to parse scorecard output")
			log.Debug("unable to successfully parse the scorecard output", err)
			return false, err
		}
		//test fails but keeps going listing out all tests
		s := v.(string)
		log.Info(s)
		if strings.Contains(s, "fail") {
			foundTestFailed = true
		}
	}
	return !foundTestFailed, nil
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
