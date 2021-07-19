package shell

import (
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

type HasUniqueTagCheck struct{}

func (p *HasUniqueTagCheck) Validate(image string) (bool, error) {
	tags, err := p.getDataToValidate(image)
	if err != nil {
		return false, err
	}
	return p.validate(image, tags)
}

func (p *HasUniqueTagCheck) getDataToValidate(image string) ([]string, error) {

	runReport, err := skopeoEngine.ListTags(image)

	if err != nil {
		log.Error("unable to execute skopeo on the image: ", err)
		return nil, err
	}

	return runReport.Tags, nil
}

func (p *HasUniqueTagCheck) validate(image string, tags []string) (bool, error) {
	// An image passes the check if:
	// 1) it has more than one tag (`latest` is acceptable)
	// OR
	// 2) it has only one tag, and it is not `latest`
	return len(tags) > 1 || len(tags) == 1 && strings.ToLower(tags[0]) != "latest", nil
}

func (p *HasUniqueTagCheck) Name() string {
	return "HasUniqueTag"
}

func (p *HasUniqueTagCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if container has a tag other than 'latest', so that the image can be uniquely identfied.",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasUniqueTagCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check HasUniqueTag encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Add a tag to your image. Consider using Semantic Versioning. https://semver.org/",
	}
}
