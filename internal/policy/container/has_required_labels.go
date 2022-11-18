package container

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	log "github.com/sirupsen/logrus"
)

var requiredLabels = []string{"name", "vendor", "version", "release", "summary", "description"}

var _ certification.Check = &HasRequiredLabelsCheck{}

// HasRequiredLabelsCheck evaluates the image manifest to ensure that the appropriate metadata
// labels are present on the image asset as it exists in its current container registry.
type HasRequiredLabelsCheck struct{}

func (p *HasRequiredLabelsCheck) Validate(ctx context.Context, imgRef certification.ImageReference) (bool, error) {
	labels, err := p.getDataForValidate(imgRef.ImageInfo)
	if err != nil {
		return false, fmt.Errorf("could not retrieve image labels: %v", err)
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

	// TODO: We should be reporting this in the results, not in a log message
	if len(missingLabels) > 0 {
		log.Debugf("expected labels are missing: %+v", missingLabels)
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
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p *HasRequiredLabelsCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check Check HasRequiredLabel encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Add the following labels to your Dockerfile or Containerfile: name, vendor, version, release, summary, description",
	}
}
