package shell

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/komish/preflight/certification"
	"github.com/sirupsen/logrus"
)

type RunAsNonRootCheck struct{}

func (p *RunAsNonRootCheck) Validate(image string, logger *logrus.Logger) (bool, error) {
	stdouterr, err := exec.Command("podman", "run", "-it", "--rm", "--entrypoint", "id", image, "-u").CombinedOutput()
	if err != nil {
		logger.Error("unable to get the id of the runtime user of this image")
		logger.Debug(string(stdouterr), err)
		return false, err
	}

	// The output we get from the exec.Command includes returns
	stdouterrString := strings.TrimSuffix(string(stdouterr), "\r\n")
	uid, err := strconv.Atoi(stdouterrString)
	if err != nil {
		logger.Error("unable to determine the runtime user id of the image")
		logger.Debug("expected a value that could be converted to an integer, and got: ", stdouterr)
		return false, err
	}

	logger.Debugf("the runtime user id is %d", uid)

	if uid != 0 {
		return true, nil
	}

	return false, nil
}

func (p *RunAsNonRootCheck) Name() string {
	return "RunAsNonRoot"
}

func (p *RunAsNonRootCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if container runs as the root user",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *RunAsNonRootCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "A container that does not specify a non-root user will fail the automatic certification, and will be subject to a manual review before the container can be approved for publication",
		Suggestion: "Indicate a specific USER in the dockerfile or containerfile",
	}
}
