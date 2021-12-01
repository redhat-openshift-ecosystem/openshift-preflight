package operator

import (
	"strings"

	"github.com/blang/semver"
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

const ocpVerV1beta1Unsupported = "4.9"

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

	annotations, err := getAnnotationsFromBundle(imagePath)
	if err != nil {
		log.Errorf("unable to get annotations.yaml from the bundle")
		return nil, err
	}

	if versions, ok := annotations[versionsKey]; ok {
		// Check that the label range contains >= 4.9
		if isTarget49OrGreater(versions) {
			log.Debug("OpenShift 4.9 detected in annotations. Running with additional checks enabled.")
			opts.OptionalValues = make(map[string]string)
			opts.OptionalValues["k8s-version"] = "1.22"
		}
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

func isTarget49OrGreater(ocpLabelIndex string) bool {
	semVerOCPV1beta1Unsupported, _ := semver.ParseTolerant(ocpVerV1beta1Unsupported)
	// the OCP range informed cannot allow carry on to OCP 4.9+
	beginsEqual := strings.HasPrefix(ocpLabelIndex, "=")
	// It means that the OCP label is =OCP version
	if beginsEqual {
		version := cleanStringToGetTheVersionToParse(strings.Split(ocpLabelIndex, "=")[1])
		verParsed, err := semver.ParseTolerant(version)
		if err != nil {
			log.Errorf("unable to parse the value (%s) on (%s)", version, ocpLabelIndex)
			return false
		}

		if verParsed.GE(semVerOCPV1beta1Unsupported) {
			return true
		}
		return false
	}
	indexRange := cleanStringToGetTheVersionToParse(ocpLabelIndex)
	if len(indexRange) > 1 {
		// Bare version
		if !strings.Contains(indexRange, "-") {
			verParsed, err := semver.ParseTolerant(indexRange)
			if err != nil {
				log.Error("unable to parse the version")
				return false
			}
			if verParsed.GE(semVerOCPV1beta1Unsupported) {
				return true
			}
		}

		versions := strings.Split(indexRange, "-")
		version := versions[0]
		if len(versions) > 1 {
			version = versions[1]
			verParsed, err := semver.ParseTolerant(version)
			if err != nil {
				log.Error("unable to parse the version")
				return false
			}

			if verParsed.GE(semVerOCPV1beta1Unsupported) {
				return true
			}
			return false
		}

		verParsed, err := semver.ParseTolerant(version)
		if err != nil {
			log.Error("unable to parse the version")
			return false
		}

		if semVerOCPV1beta1Unsupported.GE(verParsed) {
			return true
		}
	}
	return false
}

// cleanStringToGetTheVersionToParse will remove the expected characters for
// we are able to parse the version informed.
func cleanStringToGetTheVersionToParse(value string) string {
	doubleQuote := "\""
	singleQuote := "'"
	value = strings.ReplaceAll(value, singleQuote, "")
	value = strings.ReplaceAll(value, doubleQuote, "")
	value = strings.ReplaceAll(value, "v", "")
	return value
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
