package container

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"

	"github.com/go-logr/logr"
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
)

var _ check.Check = &RunAsNonRootCheck{}

// RunAsNonRootCheck evaluates the image to determine that the runtime UID is not 0,
// which correlates to the root user.
type RunAsNonRootCheck struct{}

func (p *RunAsNonRootCheck) Validate(ctx context.Context, imgRef image.ImageReference) (bool, error) {
	user, err := p.getDataToValidate(imgRef.ImageInfo)
	if err != nil {
		return false, fmt.Errorf("could not get validation data: %v", err)
	}

	return p.validate(ctx, user)
}

func (p *RunAsNonRootCheck) getDataToValidate(image cranev1.Image) (string, error) {
	configFile, err := image.ConfigFile()
	if err != nil {
		return "", fmt.Errorf("could not retrieve ConfigFile from Image: %w", err)
	}
	return configFile.Config.User, nil
}

func (p *RunAsNonRootCheck) validate(ctx context.Context, user string) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	if user == "" {
		logger.Info("detected empty USER. Presumed to be running as root")
		logger.Info("USER value must be provided and be a non-root value for this check to pass")
		return false, nil
	}

	if user == "0" || user == "root" {
		logger.Info("detected USER specified as root or UID 0")
		logger.Info("USER other than root is required for this check to pass")
		return false, nil
	}

	logger.Info(fmt.Sprintf("USER %s specified that is non-root", user))
	return true, nil
}

func (p *RunAsNonRootCheck) Name() string {
	return "RunAsNonRoot"
}

func (p *RunAsNonRootCheck) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Checking if container runs as the root user because a container that does not specify a non-root user will fail the automatic certification, and will be subject to a manual review before the container can be approved for publication",
		Level:            "best",
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p *RunAsNonRootCheck) Help() check.HelpText {
	return check.HelpText{
		Message:    "Check RunAsNonRoot encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Indicate a specific USER in the dockerfile or containerfile",
	}
}

func (p *RunAsNonRootCheck) RequiredFilePatterns() []string {
	return nil
}
