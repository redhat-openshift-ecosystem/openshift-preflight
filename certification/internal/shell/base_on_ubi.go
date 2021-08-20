package shell

import (
	"fmt"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

// BasedOnUBICheck evaluates if the provided image is based on the Red Hat Universal Base Image
// by inspecting the contents of the `/etc/os-release` and identifying if the ID is `rhel` and the
// Name value is `Red Hat Enterprise Linux`
type BaseOnUBICheck struct{}

func (p *BaseOnUBICheck) Validate(imgRef certification.ImageReference) (bool, error) {
	lines, err := p.getDataToValidate(imgRef.ImageURI)
	if err != nil {
		log.Debugf("Stdout: %s", lines)
		return false, fmt.Errorf("%w: %s", errors.ErrRunContainerFailed, err)
	}

	return p.validate(lines)
}

func (p *BaseOnUBICheck) getDataToValidate(image string) ([]string, error) {
	runOpts := cli.ImageRunOptions{
		EntryPoint:     "cat",
		EntryPointArgs: []string{"/etc/os-release"},
		LogLevel:       "debug",
		Image:          image,
	}
	runReport, err := podmanEngine.Run(runOpts)
	if err != nil {
		log.Error("unable to inspect the os-release file in the target container: ", err)
		log.Debugf("Stdout: %s", runReport.Stdout)
		log.Debugf("Stderr: %s", runReport.Stderr)
		return nil, err
	}

	return strings.Split(runReport.Stdout, "\n"), nil
}

func (p *BaseOnUBICheck) validate(lines []string) (bool, error) {
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
		Description:      "Checking if the container's base image is based upon the Red Hat Universal Base Image (UBI)",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide", // Placeholder
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *BaseOnUBICheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check BasedOnUbi encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Change the FROM directive in your Dockerfile or Containerfile to FROM registry.access.redhat.com/ubi8/ubi",
	}
}
