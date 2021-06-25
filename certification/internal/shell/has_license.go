package shell

import (
	"os/exec"
	"strings"

	"github.com/komish/preflight/certification"
	"github.com/sirupsen/logrus"
)

type HasLicensePolicy struct{}

func (p *HasLicensePolicy) Validate(image string, logger *logrus.Logger) (bool, error) {
	stdouterr, err := exec.Command("podman", "run", "--rm", image, "ls", "-A", "/licenses").CombinedOutput()
	result := string(stdouterr)
	if err != nil {
		if strings.Contains(result, "No such file or directory") || result == "" {
			logger.Warn("license not found in the container image at /licenses")
			return false, nil
		}

		logger.Error("some error attempting to identify if /licenses container the license: ", err)
		return false, err
	}

	// sanity check - in case we don't get an error, but also don't have the file.
	if strings.Contains(result, "No such file or directory") || result == "" {
		logger.Warn("license not found in the container image at /licenses")
		return false, nil
	}

	return true, nil
}

func (p *HasLicensePolicy) Name() string {
	return "HasLicense"
}

func (p *HasLicensePolicy) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if terms and conditions for images are present.",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		PolicyURL:        "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *HasLicensePolicy) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Container images must include terms and conditions applicable to the software including open source licensing information.",
		Suggestion: "Create a directory named /licenses and include all relevant licensing and/or terms and conditions as text file(s) in that directory.",
	}
}
