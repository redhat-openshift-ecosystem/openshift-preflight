package shell

import (
	"strconv"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

// RunAsNonRootCheck evaluates the image to determine that the runtime UID is not 0,
// which correlates to the root user.
type RunAsNonRootCheck struct{}

func (p *RunAsNonRootCheck) Validate(imgRef certification.ImageReference) (bool, error) {

	line, err := p.getDataToValidate(imgRef.ImageURI)
	if err != nil {
		return false, err
	}

	return p.validate(line)

}

func (p *RunAsNonRootCheck) getDataToValidate(image string) (string, error) {
	runOpts := cli.ImageRunOptions{
		EntryPoint:     "id",
		EntryPointArgs: []string{"-u"},
		LogLevel:       "debug",
		Image:          image,
	}
	runReport, err := podmanEngine.Run(runOpts)
	if err != nil {
		log.Error("unable to get the id of the runtime user of this image, error: ", err)
		log.Debugf("stdout: %s", runReport.Stdout)
		log.Debugf("stderr: %s", runReport.Stderr)
		return "", err
	}

	return runReport.Stdout, nil
}

func (p *RunAsNonRootCheck) validate(line string) (bool, error) {
	// The output we get from the exec.Command includes returns
	stdoutString := strings.TrimSpace(line)
	uid, err := strconv.Atoi(stdoutString)
	if err != nil {
		log.Error("unable to determine the runtime user id of the image")
		log.Debug("expected a value that could be converted to an integer, and got: ", stdoutString)
		return false, err
	}

	log.Debugf("the runtime user id is %d", uid)

	return uid != 0, nil

}

func (p *RunAsNonRootCheck) Name() string {
	return "RunAsNonRoot"
}

func (p *RunAsNonRootCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if container runs as the root user because a container that does not specify a non-root user will fail the automatic certification, and will be subject to a manual review before the container can be approved for publication",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *RunAsNonRootCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check RunAsNonRoot encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Indicate a specific USER in the dockerfile or containerfile",
	}
}
