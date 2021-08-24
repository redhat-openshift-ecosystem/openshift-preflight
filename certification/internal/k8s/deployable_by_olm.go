package k8s

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

type DeployableByOlmCheck struct{}

func (p *DeployableByOlmCheck) Validate(bundleRef certification.ImageReference) (bool, error) {

	result, err := podmanEngine.UnshareWithCheck("DeployableByOLMMounted", bundleRef.ImageURI, true)

	if err != nil {
		log.Trace("unable to execute preflight in the unshare env")
		log.Debugf("Stdout: %s", result.Stdout)
		log.Debugf("Stderr: %s", result.Stderr)
		return false, err
	}

	return result.PassedOverall, nil
}

func (p *DeployableByOlmCheck) Name() string {
	return "DeployableByOLM"
}

func (p *DeployableByOlmCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the operator could be deployed by OLM",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide", // Placeholder
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *DeployableByOlmCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "It is required that your operator could be deployed by OLM",
		Suggestion: "Follow the guidelines on the operatorsdk website to learn how to package your operator https://sdk.operatorframework.io/docs/olm-integration/cli-overview/",
	}
}
