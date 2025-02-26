package container

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"

	"github.com/go-logr/logr"
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
)

var requiredLabels = []string{"name", "vendor", "version", "release", "summary", "description", "maintainer"}

var trademarkLabels = []string{"name", "vendor", "maintainer"}

var _ check.Check = &HasRequiredLabelsCheck{}

// HasRequiredLabelsCheck evaluates the image manifest to ensure that the appropriate metadata
// labels are present on the image asset as it exists in its current container registry.
type HasRequiredLabelsCheck struct {
	Policy policy.Policy
}

func (p *HasRequiredLabelsCheck) Validate(ctx context.Context, imgRef image.ImageReference) (bool, error) {
	labels, err := p.getDataForValidate(imgRef.ImageInfo)
	if err != nil {
		return false, fmt.Errorf("could not retrieve image labels: %v", err)
	}

	return p.validate(ctx, labels)
}

func (p *HasRequiredLabelsCheck) getDataForValidate(image cranev1.Image) (map[string]string, error) {
	configFile, err := image.ConfigFile()
	return configFile.Config.Labels, err
}

func (p *HasRequiredLabelsCheck) validate(ctx context.Context, labels map[string]string) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	trademarkViolationLabels := []string{}
	if p.Policy != policy.PolicyKonflux {
		for _, label := range trademarkLabels {
			if violatesRedHatTrademark(labels[label]) {
				trademarkViolationLabels = append(trademarkViolationLabels, label)
			}
		}
	}

	missingLabels := []string{}
	for _, label := range requiredLabels {
		if labels[label] == "" {
			missingLabels = append(missingLabels, label)
		}
	}

	// TODO: We should be reporting this in the results, not in a log message
	if len(missingLabels) > 0 {
		logger.V(log.DBG).Info("expected labels are missing", "missingLabels", missingLabels)
	}

	if len(trademarkViolationLabels) > 0 {
		logger.V(log.DBG).Info("labels violate Red Hat trademark", "trademarkViolationLabels", trademarkViolationLabels)
	}

	return len(missingLabels) == 0 && len(trademarkViolationLabels) == 0, nil
}

func (p *HasRequiredLabelsCheck) Name() string {
	return "HasRequiredLabel"
}

func (p *HasRequiredLabelsCheck) Metadata() check.Metadata {
	return check.Metadata{
		Description: "Checking if the required labels (name, vendor, version, release, summary, description, maintainer) are present in the container metadata " +
			"and that they do not violate Red Hat trademark.",
		Level:            "good",
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p *HasRequiredLabelsCheck) Help() check.HelpText {
	return check.HelpText{
		Message: "Check HasRequiredLabel encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Add the following labels to your Dockerfile or Containerfile: name, vendor, version, release, summary, description, maintainer " +
			"and validate that they do not violate Red Hat trademark.",
	}
}
