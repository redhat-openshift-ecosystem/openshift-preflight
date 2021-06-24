package shell

import (
	"os/exec"
	"strings"

	"github.com/komish/preflight/certification"
	"github.com/sirupsen/logrus"
)

type BasedOnUbiPolicy struct{}

func (p *BasedOnUbiPolicy) Validate(image string, logger *logrus.Logger) (bool, error) {
	stdouterr, err := exec.Command("podman", "run", "--rm", "-it", image, "cat", "/etc/os-release").CombinedOutput()
	if err != nil {
		logger.Error("unable to inspect the os-release file in the target container: ", err)
		return false, err
	}

	lines := strings.Split(string(stdouterr), "\n")

	var hasRHELID, hasRHELName bool
	for _, value := range lines {
		if strings.HasPrefix(value, `ID="rhel"`) {
			hasRHELID = true
		} else if strings.HasPrefix(value, `NAME="Red Hat Enterprise Linux"`) {
			hasRHELName = true
		}
	}
	if hasRHELID && hasRHELName {
		return true, nil
	}

	return false, nil
}

func (p *BasedOnUbiPolicy) Name() string {
	return "BasedOnUbi"
}

func (p *BasedOnUbiPolicy) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the container's base image is based on UBI",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide", // Placeholder
		PolicyURL:        "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *BasedOnUbiPolicy) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "It is recommened that your image be based upon the Red Hat Universal Base Image (UBI)",
		Suggestion: "Change the FROM directive in your Dockerfile or Containerfile to FROM registry.access.redhat.com/ubi8/ubi",
	}
}
