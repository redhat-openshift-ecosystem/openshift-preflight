package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
)

var _ check.Check = &HasProhibitedContainerName{}

type HasProhibitedContainerName struct{}

func (p HasProhibitedContainerName) Validate(ctx context.Context, imageReference image.ImageReference) (result bool, err error) {
	return p.validate(ctx, p.getDataForValidate(imageReference.ImageRepository))
}

func (p HasProhibitedContainerName) getDataForValidate(imageRepository string) string {
	// splitting on '/' to get container name, at this point we know that
	// crane's ParseReference has set ImageReference.imageRepository in a valid format
	repository := strings.Split(imageRepository, "/")

	// always return last element which is container name
	return repository[len(repository)-1]
}

func (p HasProhibitedContainerName) validate(ctx context.Context, containerName string) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	result, err := violatesRedHatTrademark(containerName)
	if err != nil {
		return false, fmt.Errorf("error while validating container name: %w", err)
	}

	if result {
		logger.V(log.DBG).Info("container name violate Red Hat trademark", "container-name", containerName)
		return false, nil
	}

	return true, nil
}

func (p HasProhibitedContainerName) Name() string {
	return "HasProhibitedContainerName"
}

func (p HasProhibitedContainerName) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Checking if the container-name violates Red Hat trademark.",
		Level:            "good",
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p HasProhibitedContainerName) Help() check.HelpText {
	return check.HelpText{
		Message:    "Check HasProhibitedContainerName encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Update container-name ie (quay.io/repo-name/container-name) to not violate Red Hat trademark.",
	}
}

func (p HasProhibitedContainerName) RequiredFilePatterns() []string {
	return nil
}
