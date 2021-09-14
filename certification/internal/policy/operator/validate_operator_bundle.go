package operator

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
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

func (p ValidateOperatorBundleCheck) Validate(bundleRef certification.ImageReference) (bool, error) {
	report, err := p.getDataToValidate(bundleRef.ImageFSPath)
	if err != nil {
		log.Error("Error while executing operator-sdk bundle validate: ", err)
		return false, err
	}

	return p.validate(report)
}

func (p ValidateOperatorBundleCheck) getDataToValidate(imagePath string) (*cli.OperatorSdkBundleValidateReport, error) {
	selector := []string{"community", "operatorhub"}
	opts := cli.OperatorSdkBundleValidateOptions{
		Selector:        selector,
		Verbose:         true,
		ContainerEngine: "none",
		OutputFormat:    "json-alpha1",
	}

	return p.OperatorSdkEngine.BundleValidate(imagePath, opts)
}

func (p ValidateOperatorBundleCheck) validate(report *cli.OperatorSdkBundleValidateReport) (bool, error) {
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
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p ValidateOperatorBundleCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check ValidateOperatorBundle encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Valid bundles are defined by bundle spec, so make sure that this bundle conforms to that spec. More Information: https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md",
	}
}
