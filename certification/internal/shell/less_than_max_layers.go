package shell

import (
	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/certification/errors"
	"github.com/sirupsen/logrus"
)

type UnderLayerMaxCheck struct {
}

func (p *UnderLayerMaxCheck) Validate(image string, logger *logrus.Logger) (bool, error) {
	return false, errors.ErrFeatureNotImplemented
}

func (p *UnderLayerMaxCheck) Name() string {
	return "Under40Layers"
}

func (p *UnderLayerMaxCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if container has less than 40 layers",
		Level:            "better",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *UnderLayerMaxCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Uncompressed container images should have less than 40 layers. Too many layers within the container images can degrade container performance.",
		Suggestion: "Optimize your Dockerfile to consolidate and minimize the number of layers. Each RUN command will produce a new layer. Try combining RUN commands using && where possible.",
	}
}
