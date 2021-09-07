package container

import (
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

// RunAsNonRootCheck evaluates the image to determine that the runtime UID is not 0,
// which correlates to the root user.
type RunAsNonRootCheck struct{}

func (p *RunAsNonRootCheck) Validate(imgRef certification.ImageReference) (bool, error) {
	user, err := p.getDataToValidate(imgRef.ImageInfo)
	if err != nil {
		return false, err
	}

	return p.validate(user)
}

func (p *RunAsNonRootCheck) getDataToValidate(image cranev1.Image) (string, error) {
	configFile, err := image.ConfigFile()
	if err != nil {
		log.Error("could not retrieve ConfigFile from Image")
		return "", err
	}
	return configFile.Config.User, nil
}

func (p *RunAsNonRootCheck) validate(user string) (bool, error) {
	if user == "" {
		log.Debug("detected empty user. Presumed to be running as root")
		return false, nil
	}

	if user == "0" || user == "root" {
		log.Debugf("detected user specified as root: %s", user)
		return false, nil
	}

	log.Debug("User specified that was not root")
	return true, nil
}

func (p *RunAsNonRootCheck) Name() string {
	return "RunAsNonRoot"
}

func (p *RunAsNonRootCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if container runs as the root user because a container that does not specify a non-root user will fail the automatic certification, and will be subject to a manual review before the container can be approved for publication",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *RunAsNonRootCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check RunAsNonRoot encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Indicate a specific USER in the dockerfile or containerfile",
	}
}
