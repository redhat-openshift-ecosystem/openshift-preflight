package container

import (
	"context"
	"fmt"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	log "github.com/sirupsen/logrus"
)

// RunSystemContainerCheck runs a container as a systemd service
// and ensures that the service is up and running.
type RunSystemContainerCheck struct {
	PodmanEngine cli.PodmanEngine
}

func NewRunSystemContainerCheck(podmanEngine *cli.PodmanEngine) *RunSystemContainerCheck {
	return &RunSystemContainerCheck{
		PodmanEngine: *podmanEngine,
	}
}

func (p *RunSystemContainerCheck) Validate(ctx context.Context, imgRef certification.ImageReference) (bool, error) {
	containerName := "podman-test"

	runOptions := &cli.PodmanCreateOption{
		ContainerName: containerName,
	}

	createOutput, err := p.PodmanEngine.CreateContainer(imgRef.ImageURI, *runOptions)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := p.PodmanEngine.RemoveContainer(containerName); err != nil {
			log.Warnf("unable to remove container %s %s", createOutput.ContainerId, err)
		}
	}()

	podmanOutput, err := p.PodmanEngine.RunSystemContainer(containerName)
	if err != nil {
		return false, err
	}

	serviceName := podmanOutput.Stdout
	defer func() {
		if err := p.PodmanEngine.StopSystemContainer(serviceName); err != nil {
			log.Warnf("unable to stop service %s: %s", serviceName, err)
		}
		os.Remove(fmt.Sprintf("%s/%s", certification.SystemdDir, serviceName))
	}()

	status, err := p.PodmanEngine.IsSystemContainerRunning(serviceName)
	if err != nil {
		return false, err
	}

	return status, nil
}

func (p *RunSystemContainerCheck) Name() string {
	return "RunSystemContainer"
}

func (p *RunSystemContainerCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if a container can run as a systemd service",
		Level:            "best",
		KnowledgeBaseURL: certDocumentationURL,
		CheckURL:         certDocumentationURL,
	}
}

func (p *RunSystemContainerCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check RunSystemContainer encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Ensure that the container can be launched as a systemd service",
	}
}
