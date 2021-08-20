package shell

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

type OperatorPkgNameIsUniqueCheck struct{}

func (p *OperatorPkgNameIsUniqueCheck) Validate(imgRef certification.ImageReference) (bool, error) {
	mounted := true
	result, err := podmanEngine.UnshareWithCheck("OperatorPackageNameIsUniqueMounted", imgRef.ImageURI, mounted)
	if err != nil {
		log.Trace("unable to execute preflight in the unshare env")
		log.Debugf("Stdout: %s", result.Stdout)
		log.Debugf("Stderr: %s", result.Stderr)
		return false, err
	}

	return result.PassedOverall, nil
}

func (p *OperatorPkgNameIsUniqueCheck) Name() string {
	return "OperatorPackageNameIsUnique"
}

func (p *OperatorPkgNameIsUniqueCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Validating Bundle image package name uniqueness",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *OperatorPkgNameIsUniqueCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check encountered an error. It is possible that the bundle package name already exist in the RedHat Catalog registry.",
		Suggestion: "Bundle package name must be unique meaning that it doesn't already exist in the RedHat Catalog registry",
	}
}
