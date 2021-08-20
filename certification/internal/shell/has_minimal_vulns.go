package shell

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

// HasMinimalVulnerabilitiesCheck calls the HasMinimalVulnerabilitiesUnshareCheck
// In an Unshare environment. It does not require a mounted image.
type HasMinimalVulnerabilitiesCheck struct{}

func (p *HasMinimalVulnerabilitiesCheck) Validate(imgRef certification.ImageReference) (bool, error) {
	mounted := false
	result, err := podmanEngine.UnshareWithCheck("HasMinimalVulnerabilitiesUnshare", imgRef.ImageURI, mounted)
	if err != nil {
		log.Trace("unable to execute preflight in the unshare env")
		log.Debugf("Stdout: %s", result.Stdout)
		log.Debugf("Stderr: %s", result.Stderr)
		return false, err
	}

	return result.PassedOverall, nil
}

func (p *HasMinimalVulnerabilitiesCheck) Name() string {
	return "HasMinimalVulnerabilities"
}

func (p *HasMinimalVulnerabilitiesCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking container image does not contain any critical or important security vulnerabilities, as defined at https://access.redhat.com/security/updates/classification.",
		Level:            "good",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasMinimalVulnerabilitiesCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check HasMinimalVulnerabilities encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Update your UBI image to the latest version or update the packages in your image to the latest versions distributed by Red Hat.",
	}
}
