package operator

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/bundle"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	log "github.com/sirupsen/logrus"
)

// ValidateOperatorBundleCheck evaluates the image and ensures that it passes bundle validation
// as executed by `operator-sdk bundle validate`
type ValidateOperatorBundleCheck struct {
	OperatorSdkEngine cli.OperatorSdkEngine
}

func NewValidateOperatorBundleCheck(operatorSdkEngine *cli.OperatorSdkEngine) *ValidateOperatorBundleCheck {
	return &ValidateOperatorBundleCheck{
		OperatorSdkEngine: *operatorSdkEngine,
	}
}

const ocpVerV1beta1Unsupported = "4.9"

func (p ValidateOperatorBundleCheck) Validate(ctx context.Context, bundleRef certification.ImageReference) (bool, error) {
	report, err := p.getDataToValidate(ctx, bundleRef.ImageFSPath)
	if err != nil {
		return false, fmt.Errorf("error while executing operator-sdk bundle validate: %v", err)
	}

	return p.validate(ctx, report)
}

func (p ValidateOperatorBundleCheck) getDataToValidate(ctx context.Context, imagePath string) (*cli.OperatorSdkBundleValidateReport, error) {
	return bundle.Validate(ctx, p.OperatorSdkEngine, imagePath)
}

func (p ValidateOperatorBundleCheck) validate(ctx context.Context, report *cli.OperatorSdkBundleValidateReport) (bool, error) {
	if !report.Passed || len(report.Outputs) > 0 {
		for _, output := range report.Outputs {
			var logFn func(...interface{})
			switch output.Type {
			case "warning":
				logFn = log.Warn
			case "error":
				logFn = log.Error
			default:
				logFn = log.Debug
			}
			logFn(output.Message)
		}
	}
	return report.Passed, nil
}

func (p ValidateOperatorBundleCheck) Name() string {
	return "ValidateOperatorBundle"
}

func (p ValidateOperatorBundleCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Validating Bundle image that checks if it can validate the content and format of the operator bundle",
		Level:            "best",
		KnowledgeBaseURL: "https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/",
		CheckURL:         "https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/",
	}
}

func (p ValidateOperatorBundleCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check ValidateOperatorBundle encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Valid bundles are defined by bundle spec, so make sure that this bundle conforms to that spec. More Information: https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md",
	}
}
