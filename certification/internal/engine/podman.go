package engine

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	log "github.com/sirupsen/logrus"
)

type podmanEngine struct{}

func NewPodmanEngine() *cli.PodmanEngine {
	var engine cli.PodmanEngine = &podmanEngine{}
	return &engine
}

func (p *podmanEngine) PullImage(imageURI string, options cli.ImagePullOptions) (*cli.PodmanOutput, error) {
	log.Debug(fmt.Sprintf("Pulling image %s from repository", imageURI))
	cmdArgs := []string{"pull"}

	if options.Quiet {
		cmdArgs = append(cmdArgs, "--quiet")
	}

	cmdArgs = append(cmdArgs, imageURI)

	cmd := exec.Command("podman", cmdArgs...)

	log.Debugf("Command being run: %+v", cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error(fmt.Sprintf("unable to pull image %s: ", imageURI), err)
		log.Error("Stderr: ", stderr.String())
		return &cli.PodmanOutput{
			Stderr: stderr.String(),
		}, err
	}
	log.Debug(fmt.Sprintf("Successfully pulled image %s from repository", imageURI))
	return &cli.PodmanOutput{
		Stdout: stdout.String(),
	}, nil
}

func (p *podmanEngine) CreateContainer(imageURI string, createOptions cli.PodmanCreateOption) (*cli.PodmanCreateOutput, error) {
	log.Debug(fmt.Sprintf("Creating container %s with the run options: %+v", imageURI, createOptions))

	if _, err := p.PullImage(imageURI, cli.ImagePullOptions{Quiet: true}); err != nil {
		return nil, err
	}
	cmdArgs := []string{"create"}

	if len(createOptions.Entrypoint) > 0 {
		cmdArgs = append(cmdArgs, "--entrypoint")
		cmdArgs = append(cmdArgs, createOptions.Entrypoint...)
	}

	cmdArgs = append(cmdArgs, imageURI)

	if len(createOptions.Cmd) > 0 {
		cmdArgs = append(cmdArgs, createOptions.Cmd...)
	}

	cmd := exec.Command("podman", cmdArgs...)

	log.Debugf("Command being run: %+v", cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error(fmt.Sprintf("unable to create container %s: ", imageURI), err)
		log.Error("Stderr: ", stderr.String())
		return &cli.PodmanCreateOutput{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, err
	}
	containerId := strings.TrimSpace(stdout.String())
	log.Debug(fmt.Sprintf("Successfully created container %s with options %+v ...", imageURI, createOptions))
	log.Debug(fmt.Sprintf("Container Id is %s.", containerId))

	return &cli.PodmanCreateOutput{
		ContainerId: containerId,
		Stdout:      stdout.String(),
		Stderr:      stderr.String(),
	}, nil
}

func (p *podmanEngine) StartContainer(nameOrId string) (*cli.PodmanOutput, error) {
	log.Debug(fmt.Sprintf("Starting container %s", nameOrId))

	cmdArgs := []string{"start"}
	cmdArgs = append(cmdArgs, nameOrId)
	cmd := exec.Command("podman", cmdArgs...)

	log.Debugf("Command being run: %+v", cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Error(fmt.Sprintf("unable to start container %s: ", nameOrId), err)
		log.Error("Stderr: ", stderr.String())
		return &cli.PodmanOutput{
			Stderr: stderr.String(),
		}, err
	}
	log.Debug(fmt.Sprintf("Successfully started container %s...", nameOrId))
	return &cli.PodmanOutput{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, nil
}

func (p *podmanEngine) RemoveContainer(containerId string) error {
	log.Debug(fmt.Sprintf("Removing container %s", containerId))

	cmdArgs := []string{"rm", "--force"}
	cmdArgs = append(cmdArgs, containerId)
	cmd := exec.Command("podman", cmdArgs...)

	log.Debugf("Command being run: %+v", cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Error(fmt.Sprintf("unable to remove container %s: ", containerId), err)
		log.Error("Stderr: ", stderr.String())
		return err
	}
	return nil
}

func (p *podmanEngine) WaitContainer(containerId string, waitOptions cli.WaitOptions) (bool, error) {
	log.Debug(fmt.Sprintf("Checking for the status of the container %s...", containerId))

	cmdArgs := []string{"wait"}
	if len(waitOptions.Interval) > 0 {
		cmdArgs = append(cmdArgs, "--interval", waitOptions.Interval)
	}

	if len(waitOptions.Condition) > 0 {
		cmdArgs = append(cmdArgs, "--condition", waitOptions.Condition)
	}
	cmdArgs = append(cmdArgs, containerId)

	cmd := exec.Command("podman", cmdArgs...)

	log.Debugf("Command being run: %+v", cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error(fmt.Sprintf("unable to wait for container %s: ", containerId), err)
		log.Error("Stderr: ", stderr.String())
		return false, err
	}

	log.Info("container reached a running state")
	return true, nil
}
