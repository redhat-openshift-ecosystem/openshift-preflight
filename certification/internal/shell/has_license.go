package shell

import (
	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/certification/errors"
	"github.com/sirupsen/logrus"
)

type HasLicensePolicy struct{}

func (p *HasLicensePolicy) Validate(image string, logger *logrus.Logger) (bool, error) {
	return false, errors.ErrFeatureNotImplemented
}

func (p *HasLicensePolicy) Name() string {
	return "HasLicense"
}

func (p *HasLicensePolicy) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if terms and conditions for images are present.",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		PolicyURL:        "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasLicensePolicy) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Container images must include terms and conditions applicable to the software including open source licensing information.",
		Suggestion: "Create a directory named /licenses and include all relevant licensing and/or terms and conditions as text file(s) in that directory.",
	}
}
