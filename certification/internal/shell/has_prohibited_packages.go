package shell

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

// HasProhibitedPackages evaluates that the image does not contain prohibited packages,
// which refers to packages that are not redistributable without an appropriate license.
type HasNoProhibitedPackagesCheck struct{}

func (p *HasNoProhibitedPackagesCheck) Validate(imgRef certification.ImageReference) (bool, error) {
	mounted := true
	result, err := podmanEngine.UnshareWithCheck("HasNoProhibitedPackagesMounted", imgRef.ImageURI, mounted)
	if err != nil {
		log.Trace("unable to execute preflight in the unshare env")
		log.Debugf("Stdout: %s", result.Stdout)
		log.Debugf("Stderr: %s", result.Stderr)
		return false, err
	}

	return result.PassedOverall, nil
}

func (p *HasNoProhibitedPackagesCheck) Name() string {
	return "HasNoProhibitedPackages"
}

func (p *HasNoProhibitedPackagesCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checks to ensure that the image in use does not include prohibited packages, such as Red Hat Enterprise Linux (RHEL) kernel packages.",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasNoProhibitedPackagesCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check HasNoProhibitedPackages encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Remove any RHEL packages that are not distributable outside of UBI",
	}
}
