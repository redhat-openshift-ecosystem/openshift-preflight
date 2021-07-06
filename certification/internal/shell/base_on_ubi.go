package shell

import (
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	"github.com/sirupsen/logrus"
)

type BaseOnUBICheck struct{}

func (p *BaseOnUBICheck) Validate(image string, logger *logrus.Logger) (bool, error) {
	podmanEngine := PodmanCLIEngine{}
	return p.validate(podmanEngine, image, logger)
}

func (p *BaseOnUBICheck) validate(podmanEngine cli.PodmanEngine, image string, logger *logrus.Logger) (bool, error) {
	runOpts := cli.ImageRunOptions{
		EntryPoint:     "cat",
		EntryPointArgs: []string{"/etc/os-release"},
		LogLevel:       "debug",
		Image:          image,
	}
	runReport, err := podmanEngine.Run(runOpts)
	if err != nil {
		logger.Error("unable to inspect the os-release file in the target container: ", err)
		logger.Debugf("Stdout: %s", runReport.Stdout)
		logger.Debugf("Stderr: %s", runReport.Stderr)
		return false, err
	}

	lines := strings.Split(runReport.Stdout, "\n")

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

func (p *BaseOnUBICheck) Name() string {
	return "BasedOnUbi"
}

func (p *BaseOnUBICheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the container's base image is based on UBI",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide", // Placeholder
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *BaseOnUBICheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "It is recommened that your image be based upon the Red Hat Universal Base Image (UBI)",
		Suggestion: "Change the FROM directive in your Dockerfile or Containerfile to FROM registry.access.redhat.com/ubi8/ubi",
	}
}
