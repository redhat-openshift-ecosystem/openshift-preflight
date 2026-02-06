package container

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"

	"github.com/go-logr/logr"
)

var trademarkLabels = []string{"name", "vendor", "maintainer"}

var _ check.Check = &HasNoProhibitedLabelsCheck{}

type HasNoProhibitedLabelsCheck struct{}

func (p *HasNoProhibitedLabelsCheck) Validate(ctx context.Context, imgRef image.ImageReference) (result bool, err error) {
	labels, err := getContainerLabels(imgRef.ImageInfo)
	if err != nil {
		return false, fmt.Errorf("could not retrieve image labels: %v", err)
	}

	return p.validate(ctx, labels)
}

func (p *HasNoProhibitedLabelsCheck) validate(ctx context.Context, labels map[string]string) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	trademarkViolationLabels := []string{}
	for _, label := range trademarkLabels {
		result, err := violatesRedHatTrademark(labels[label])
		if err != nil {
			return false, fmt.Errorf("error while validating label: %w", err)
		}

		if result {
			trademarkViolationLabels = append(trademarkViolationLabels, label)
		}
	}

	// TODO: We should be reporting this in the results, not in a log message
	if len(trademarkViolationLabels) > 0 {
		logger.V(log.DBG).Info("labels violate Red Hat trademark", "trademarkViolationLabels", trademarkViolationLabels)
	}

	return len(trademarkViolationLabels) == 0, nil
}

func (p *HasNoProhibitedLabelsCheck) Name() string {
	return "HasNoProhibitedLabels"
}

func (p *HasNoProhibitedLabelsCheck) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Checking if the labels (name, vendor, maintainer) violate Red Hat trademark.",
		Level:            "good",
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p *HasNoProhibitedLabelsCheck) Help() check.HelpText {
	return check.HelpText{
		Message:    "Check HasNoProhibitedLabelsCheck encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Ensure the name, vendor, and maintainer label on your image do not violate the Red Hat trademark.",
	}
}

func (p *HasNoProhibitedLabelsCheck) RequiredFilePatterns() []string {
	return nil
}
