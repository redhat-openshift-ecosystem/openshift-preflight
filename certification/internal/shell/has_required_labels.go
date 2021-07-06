package shell

import (
	"encoding/json"
	"os/exec"

	"github.com/itchyny/gojq"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/sirupsen/logrus"
)

type HasRequiredLabelsCheck struct{}

func (p *HasRequiredLabelsCheck) Validate(image string, logger *logrus.Logger) (bool, error) {
	// TODO: if we're going have the image json on disk already, we should use it here instead of podman inspect-ing.
	stdouterr, err := exec.Command("podman", "inspect", image).CombinedOutput()
	if err != nil {
		logger.Error("unable to execute inspect on the image: ", err)
		return false, err
	}

	// we must send gojq a []interface{}, so we have to convert our inspect output to that type
	var inspectData []interface{}
	err = json.Unmarshal(stdouterr, &inspectData)
	if err != nil {
		logger.Error("unable to parse podman inspect data for image")
		logger.Debug("error marshaling podman inspect data: ", err)
		logger.Trace("failure in attempt to convert the raw bytes from `podman inspect` to a []interface{}")
		return false, err
	}

	jqQueryString := ".[0] | .Labels | {name: .name, vendor: .vendor, version: .version, release: .release, summary: .summary, description: .description}"

	query, err := gojq.Parse(jqQueryString)
	if err != nil {
		logger.Error("unable to parse podman inspect data for image")
		logger.Debug("unable to successfully parse the gojq query string:", err)
		return false, err
	}

	// gojq expects us to iterate in the event that our query returned multiple matching values, but we only expect one.
	iter := query.Run(inspectData)
	val, nextOk := iter.Next()

	if !nextOk {
		logger.Warn("did not receive any label information when parsing container image")
		// in this case, there was no data returned from jq, so we need to fail the check.
		return false, nil
	}

	// gojq can return an error in iteration, so we need to check for that.
	if err, ok := val.(error); ok {
		logger.Error("unable to parse podman inspect data for image")
		logger.Debug("unable to successfully parse the podman inspect output with the query string provided:", err)
		// this is an error, as we didn't get the proper input from `podman inspect`
		return false, err
	}

	labels := val.(map[string]interface{})

	for _, label := range labels {
		if label == nil {
			logger.Warn("an expected label is missing:", label)
			return false, nil
		}
	}

	return true, nil
}

func (p *HasRequiredLabelsCheck) Name() string {
	return "HasRequiredLabel"
}

func (p *HasRequiredLabelsCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the container's base image is based on UBI",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasRequiredLabelsCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "It is recommened that your image be based upon the Red Hat Universal Base Image (UBI)",
		Suggestion: "Change the FROM directive in your Dockerfile or Containerfile to FROM registry.access.redhat.com/ubi8/ubi",
	}
}
