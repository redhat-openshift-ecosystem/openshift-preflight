package shell

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/sirupsen/logrus"
)

type HasUniqueTagCheck struct{}

func (p *HasUniqueTagCheck) Validate(image string, logger *logrus.Logger) (bool, error) {
	imageName := strings.Split(image, ":")[0]
	stdouterr, err := exec.Command("skopeo", "list-tags", "docker://"+imageName).CombinedOutput()
	if err != nil {
		logger.Error("unable to execute skopeo on the image: ", err)
		return false, err
	}

	// we must send gojq a []interface{}, so we have to convert our inspect output to that type
	var skopeoData map[string]interface{}
	err = json.Unmarshal(stdouterr, &skopeoData)
	if err != nil {
		logger.Error("unable to parse skopeo list-tags data for image", err)
		logger.Debug("error marshaling skopeo list-tags data: ", err)
		logger.Trace("failure in attempt to convert the raw bytes from `skopeo list-tags` to a [map[string]interface{}")
		return false, err
	}
	tags := skopeoData["Tags"].([]interface{})

	var tagsString string
	for _, tag := range tags {
		tagsString = tagsString + tag.(string) + " "
	}
	logger.Debugf(fmt.Sprintf("detected these tags for %s image: %s", imageName, tagsString))

	if len(tags) > 1 || len(tags) == 1 && strings.ToLower(tags[0].(string)) != "latest" {
		return true, nil
	}
	return false, nil
}

func (p *HasUniqueTagCheck) Name() string {
	return "HasUniqueTag"
}

func (p *HasUniqueTagCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if container has a tag other than 'latest'.",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasUniqueTagCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Containers should have a tag other than latest, so that the image can be uniquely identfied.",
		Suggestion: "Add a tag to your image. Consider using Semantic Versioning. https://semver.org/",
	}
}
