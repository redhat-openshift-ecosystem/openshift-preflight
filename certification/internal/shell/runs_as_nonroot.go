package shell

import (
	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/certification/errors"
	"github.com/sirupsen/logrus"
)

type RunAsNonRootCheck struct {
}

func (p *RunAsNonRootCheck) Validate(image string, logger *logrus.Logger) (bool, error) {
	return false, errors.ErrFeatureNotImplemented
}

func (p *RunAsNonRootCheck) Name() string {
	return "RunAsNonRoot"
}

func (p *RunAsNonRootCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if container runs as the root user",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *RunAsNonRootCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "A container that does not specify a non-root user will fail the automatic certification, and will be subject to a manual review before the container can be approved for publication",
		Suggestion: "Indicate a specific USER in the dockerfile or containerfile",
	}
}
