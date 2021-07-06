package shell

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/sirupsen/logrus"
)

type HasMinimalVulnerabilitiesCheck struct{}

func (p *HasMinimalVulnerabilitiesCheck) Validate(image string, logger *logrus.Logger) (bool, error) {
	return false, errors.ErrFeatureNotImplemented
}
func (p *HasMinimalVulnerabilitiesCheck) Name() string {
	return "HasMinimalVulnerabilities"
}

func (p *HasMinimalVulnerabilitiesCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking for critical or important security vulnerabilites.",
		Level:            "good",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasMinimalVulnerabilitiesCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Components in the container image cannot contain any critical or important vulnerabilities, as defined at https://access.redhat.com/security/updates/classification",
		Suggestion: "Update your UBI image to the latest version or update the packages in your image to the latest versions distrubuted by Red Hat.",
	}
}
