package shell

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/itchyny/gojq"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/sirupsen/logrus"
)

const (
	acceptableLayerMax = 40
)

type UnderLayerMaxCheck struct{}

func (p *UnderLayerMaxCheck) Validate(image string, logger *logrus.Logger) (bool, error) {
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
		logger.Error("unable to parse podman inspect data for image", err)
		logger.Debug("error marshaling podman inspect data: ", err)
		logger.Trace("failure in attempt to convert the raw bytes from `podman inspect` to a []interface{}")
		return false, err
	}

	query, err := gojq.Parse(".[0].RootFS.Layers")
	if err != nil {
		logger.Error("unable to parse podman inspect data for image", err)
		logger.Debug("unable to successfully parse the gojq query string:", err)
		return false, err
	}

	// gojq expects us to iterate in the event that our query returned multiple matching values, but we only expect one.
	iter := query.Run(inspectData)
	val, nextOk := iter.Next()

	if !nextOk {
		logger.Warn("did not receive any layer information when parsing container image")
		// in this case, there was no data returned from jq, so we need to fail the check.
		return false, nil
	}

	// gojq can return an error in iteration, so we need to check for that.
	if err, ok := val.(error); ok {
		logger.Error("unable to parse podman inspect data for image", err)
		logger.Debug("unable to successfully parse the podman inspect output with the query string provided:", err)
		// this is an error, as we didn't get the proper input from `podman inspect`
		return false, err
	}

	layers := val.([]interface{})

	logger.Debugf("detected %d layers in image", len(layers))
	if len(layers) < acceptableLayerMax {
		return true, nil
	}

	return false, nil
}

func (p *UnderLayerMaxCheck) Name() string {
	return "LayerCountAcceptable"
}

func (p *UnderLayerMaxCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      fmt.Sprintf("Checking if container has less than %d layers", acceptableLayerMax),
		Level:            "better",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *UnderLayerMaxCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    fmt.Sprintf("Uncompressed container images should have less than %d layers. Too many layers within the container images can degrade container performance.", acceptableLayerMax),
		Suggestion: "Optimize your Dockerfile to consolidate and minimize the number of layers. Each RUN command will produce a new layer. Try combining RUN commands using && where possible.",
	}
}
