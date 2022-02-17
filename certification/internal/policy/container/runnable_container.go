package container

import (
	"github.com/containers/podman/v3/libpod/define"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	log "github.com/sirupsen/logrus"
)

// RunnableContainerCheck starts a container and
// ensures it reaches a Running state within a specified timeframe
type RunnableContainerCheck struct {
	PodmanEngine cli.PodmanEngine
}

func NewRunnableContainerCheck(podmanEngine *cli.PodmanEngine) *RunnableContainerCheck {
	return &RunnableContainerCheck{
		PodmanEngine: *podmanEngine,
	}
}

func (p *RunnableContainerCheck) Validate(imgRef certification.ImageReference) (bool, error) {
	runOptions := &cli.PodmanCreateOption{
		Cmd: []string{"sleep", checkContainerTimeout.String()},
	}

	creationOutput, err := p.PodmanEngine.CreateContainer(imgRef.ImageURI, *runOptions)
	if err != nil {
		return false, err
	}

	_, err = p.PodmanEngine.StartContainer(creationOutput.ContainerId)
	defer func() {
		if err := p.PodmanEngine.RemoveContainer(creationOutput.ContainerId); err != nil {
			log.Errorf("unable to stop container %s: %s", creationOutput.ContainerId, err)
		}
	}()
	if err != nil {
		return false, err
	}

	return p.PodmanEngine.WaitContainer(creationOutput.ContainerId, cli.WaitOptions{
		Interval:  waitContainer.String(),
		Condition: define.ContainerStateRunning.String(),
		Timeout:   checkContainerTimeout,
	})
}

func (p *RunnableContainerCheck) Name() string {
	return "RunnableContainer"
}

func (p *RunnableContainerCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if container runs within a pre-configured timeframe",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *RunnableContainerCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check RunnableContainer encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Ensure that the container can be launched",
	}
}
