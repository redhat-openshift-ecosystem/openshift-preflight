package shell

import (
	"os/exec"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

type ValidateOperatorBundlePolicy struct {
}

func (p ValidateOperatorBundlePolicy) Validate(bundle string) (bool, error) {
	stdouterr, err := exec.Command("operator-sdk", "bundle", "validate", "-b", "podman", "--verbose", bundle).CombinedOutput()
	if err != nil {
		log.Error("Error will executing operator-sdk validate bundle: ", err)
		return false, err
	}

	lines := strings.Split(string(stdouterr), "time=")

	if strings.Contains(lines[len(lines)-1], "All validation tests have completed successfully") {
		for _, line := range lines {
			if strings.Contains(line, "level=warning") {
				log.Warn("time= ", line)
			}
		}
		return true, nil
	}
	log.Warn("The bundle image did not pass all of the validation tests")
	return false, nil
}

func (p ValidateOperatorBundlePolicy) Name() string {
	return "ValidateOperatorBundle"
}

func (p ValidateOperatorBundlePolicy) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Validating Bundle image that checks if it can validate the content and format of the operator bundle",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p ValidateOperatorBundlePolicy) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check ValidateOperatorBundle encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Valid bundles are definied by bundle spec, so make sure that this bundle conforms to that spec. More Information: https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md",
	}
}
