package container

import (
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

var requiredLabels = []string{"name", "vendor", "version", "release", "summary", "description"}

// HasRequiredLabelsCheck evaluates the image manifest to ensure that the appropriate metadata
// labels are present on the image asset as it exists in its current container registry.
type HasRequiredLabelsCheck struct{}

func (p *HasRequiredLabelsCheck) Validate(imgRef certification.ImageReference) (bool, error) {
	labels, err := p.getDataForValidate(imgRef.ImageInfo)
	if err != nil {
		return false, err
	}

	return p.validate(labels)
}

func (p *HasRequiredLabelsCheck) getDataForValidate(image cranev1.Image) (map[string]string, error) {
	configFile, err := image.ConfigFile()
	return configFile.Config.Labels, err
}

func (p *HasRequiredLabelsCheck) validate(labels map[string]string) (bool, error) {
	missingLabels := []string{}
	for _, label := range requiredLabels {
		if labels[label] == "" {
			missingLabels = append(missingLabels, label)
		}
	}

	if len(missingLabels) > 0 {
		log.Warn("expected labels are missing:", missingLabels)
	}

	return len(missingLabels) == 0, nil
}

func (p *HasRequiredLabelsCheck) Name() string {
	return "HasRequiredLabel"
}

func (p *HasRequiredLabelsCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the required labels (name, vendor, version, release, summary, description) are present in the container metadata.",
		Level:            "good",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasRequiredLabelsCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check Check HasRequiredLabel encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Add the following labels to your Dockerfile or Containerfile: name, vendor, version, release, summary, description",
	}
}
