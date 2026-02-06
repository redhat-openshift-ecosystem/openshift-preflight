package operator

import (
	"context"
	"errors"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/bundle"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"

	"github.com/go-logr/logr"
)

var _ check.Check = &ValidateOperatorBundleCheck{}

// ValidateOperatorBundleCheck evaluates the image and ensures that it passes bundle validation
// as executed by `operator-sdk bundle validate`
type ValidateOperatorBundleCheck struct{}

func NewValidateOperatorBundleCheck() *ValidateOperatorBundleCheck {
	return &ValidateOperatorBundleCheck{}
}

func (p *ValidateOperatorBundleCheck) Validate(ctx context.Context, bundleRef image.ImageReference) (bool, error) {
	report, err := p.dataToValidate(ctx, bundleRef.ImageFSPath)
	if err != nil {
		return false, fmt.Errorf("error while executing operator-sdk bundle validate: %v", err)
	}

	return p.validate(ctx, report)
}

func (p *ValidateOperatorBundleCheck) dataToValidate(ctx context.Context, imagePath string) (*bundle.Report, error) {
	return bundle.Validate(ctx, imagePath)
}

func (p *ValidateOperatorBundleCheck) validate(ctx context.Context, report *bundle.Report) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	if !report.Passed || len(report.Results) > 0 {
		for _, output := range report.Results {
			for _, result := range output.Errors {
				logger.Error(errors.New("validate operator bundle error"), result.Error())
			}
			for _, result := range output.Warnings {
				logger.Info(fmt.Sprintf("warning: %s", result.Error()))
			}
		}
	}
	return report.Passed, nil
}

func (p *ValidateOperatorBundleCheck) Name() string {
	return "ValidateOperatorBundle"
}

func (p *ValidateOperatorBundleCheck) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Validating Bundle image that checks if it can validate the content and format of the operator bundle",
		Level:            "best",
		KnowledgeBaseURL: "https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/",
		CheckURL:         "https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/",
	}
}

func (p *ValidateOperatorBundleCheck) Help() check.HelpText {
	return check.HelpText{
		Message:    "Check ValidateOperatorBundle encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Valid bundles are defined by bundle spec, so make sure that this bundle conforms to that spec. More Information: https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md",
	}
}

func (p *ValidateOperatorBundleCheck) RequiredFilePatterns() []string {
	return bundle.BundleFiles
}
