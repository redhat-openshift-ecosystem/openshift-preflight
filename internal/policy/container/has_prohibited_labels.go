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

var _ check.Check = &HasProhibitedLabelsCheck{}

type HasProhibitedLabelsCheck struct{}

func (p *HasProhibitedLabelsCheck) Validate(ctx context.Context, imgRef image.ImageReference) (result bool, err error) {
	labels, err := getDataForValidate(imgRef.ImageInfo)
	if err != nil {
		return false, fmt.Errorf("could not retrieve image labels: %v", err)
	}

	return p.validate(ctx, labels)
}

func (p *HasProhibitedLabelsCheck) validate(ctx context.Context, labels map[string]string) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	trademarkViolationLabels := []string{}
	for _, label := range trademarkLabels {
		if violatesRedHatTrademark(labels[label]) {
			trademarkViolationLabels = append(trademarkViolationLabels, label)
		}
	}

	// TODO: We should be reporting this in the results, not in a log message
	if len(trademarkViolationLabels) > 0 {
		logger.V(log.DBG).Info("labels violate Red Hat trademark", "trademarkViolationLabels", trademarkViolationLabels)
	}

	return len(trademarkViolationLabels) == 0, nil
}

func (p *HasProhibitedLabelsCheck) Name() string {
	return "HasProhibitedLabelsCheck"
}

func (p *HasProhibitedLabelsCheck) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Checking if the labels (name, vendor, maintainer) violate Red Hat trademark.",
		Level:            "good",
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p *HasProhibitedLabelsCheck) Help() check.HelpText {
	return check.HelpText{
		Message:    "Check HasRequiredLabel encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Ensure the following labels to your Dockerfile or Containerfile: name, vendor, maintainer do not violate Red Hat trademark.",
	}
}
