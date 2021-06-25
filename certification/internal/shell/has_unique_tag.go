package shell

import (
	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/certification/errors"
	"github.com/sirupsen/logrus"
)

type HasUniqueTagCheck struct{}

func (p *HasUniqueTagCheck) Validate(image string, logger *logrus.Logger) (bool, error) {
	return false, errors.ErrFeatureNotImplemented
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
