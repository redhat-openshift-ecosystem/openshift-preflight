package engine

import (
	"bytes"
	"context"
	"fmt"
	"os"
	exec "os/exec"
	"regexp"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	log "github.com/sirupsen/logrus"
)

const (
	runAsPrivileged = true
)

type podmanEngine struct{}

func NewPodmanEngine() *cli.PodmanEngine {
	var engine cli.PodmanEngine = &podmanEngine{}
	return &engine
}

func (p *podmanEngine) CreateContainer(imageURI string, createOptions cli.PodmanCreateOption) (*cli.PodmanCreateOutput, error) {
	log.Debug(fmt.Sprintf("Creating container %s with the run options: %+v", imageURI, createOptions))

	cmdArgs := []string{"create"}
	if len(createOptions.Entrypoint) > 0 {
		cmdArgs = append(cmdArgs, "--entrypoint")
		cmdArgs = append(cmdArgs, createOptions.Entrypoint...)
	}
	if len(createOptions.ContainerName) > 0 {
		cmdArgs = append(cmdArgs, "--name", createOptions.ContainerName)
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
		log.Debug("Stderr: ", stderr.String())
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

	ctx, cancel := context.WithTimeout(context.Background(), waitOptions.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "podman", cmdArgs...)

	log.Debugf("Command being run: %+v", cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error(fmt.Sprintf("unable to wait for container %s: ", containerId), err)
		log.Error("Stderr: ", stderr.String())
		if strings.Contains(strings.ToLower(err.Error()), "killed") {
			return false, nil
		}
		return false, err
	}

	log.Info("container reached a running state")
	return true, nil
}

func (p *podmanEngine) RunSystemContainer(containerName string) (*cli.PodmanOutput, error) {
	log.Debug(fmt.Sprintf("Generating Systemd file for container %s", containerName))
	cmdArgs := []string{"generate", "systemd", "--new", "--files", "--name", containerName}

	output, err := RunCommand(!runAsPrivileged, "podman", cmdArgs)

	if err != nil {
		return output, err
	}
	serviceFilepath := strings.TrimSpace(output.Stdout)

	r, err := regexp.Compile(".*/(.+.service)")
	if err != nil {
		log.Error("unable to fetch the name of the systemd service: ", err)
		return &cli.PodmanOutput{
			Stderr: err.Error(),
		}, err
	}
	serviceName := r.FindStringSubmatch(serviceFilepath)[1]

	// copy the unit file from working dir to systemd dir
	_, err = copyFile(runAsPrivileged, serviceFilepath, certification.SystemdDir)
	defer os.Remove(serviceFilepath)

	if err != nil {
		log.Error(fmt.Sprintf("unable to copy the unit file into the systemd dir %s: ", certification.SystemdDir), err)
		return &cli.PodmanOutput{
			Stderr: err.Error(),
		}, err
	}

	log.Debug(fmt.Sprintf("Reloading daemon set and start the service %s", serviceName))

	if output, err = RunCommand(runAsPrivileged, "systemctl", []string{"daemon-reload"}); err != nil {
		log.Error("unable to reaload the daemon set: ", err)
		log.Error("Stderr: ", output.Stderr)
		// remove the service file
		RunCommand(runAsPrivileged, "rm", []string{"-f", fmt.Sprintf("%s/%s", certification.SystemdDir, serviceName)})

		return &cli.PodmanOutput{
			Stderr: output.Stderr,
		}, err
	}
	if output, err = RunCommand(runAsPrivileged, "systemctl", []string{"start", serviceName}); err != nil {
		log.Error(fmt.Sprintf("unable to start the service %s: ", serviceName), err)
		log.Error("Stderr: ", output.Stderr)
		// remove the service file
		RunCommand(runAsPrivileged, "rm", []string{"-f", fmt.Sprintf("%s/%s", certification.SystemdDir, serviceName)})

		return &cli.PodmanOutput{
			Stderr: output.Stderr,
		}, err
	}

	return &cli.PodmanOutput{
		Stdout: serviceName,
	}, nil
}

func (p *podmanEngine) IsSystemContainerRunning(serviceName string) (bool, error) {
	log.Debug("Checking the service status of ", serviceName)

	output, err := RunCommand(runAsPrivileged, "systemctl", []string{"is-active", serviceName})
	if err != nil {
		log.Error(fmt.Sprintf("unable to check the status of the service %s: ", serviceName), err)
		log.Error("Stderr: ", output.Stderr)
		return false, err
	}
	serviceStatus := strings.TrimSpace(output.Stdout)
	log.Debug(fmt.Sprintf("The %s status is %s", serviceName, serviceStatus))
	return strings.ToLower(serviceStatus) == "active", nil
}

func (p *podmanEngine) StopSystemContainer(serviceName string) error {
	log.Debug("Stopping the container service ", serviceName)

	if output, err := RunCommand(runAsPrivileged, "systemctl", []string{"stop", serviceName}); err != nil {
		log.Error(fmt.Sprintf("unable to start the service %s: ", serviceName), err)
		log.Error("Stderr: ", output.Stderr)
		return err
	}

	if output, err := RunCommand(runAsPrivileged, "systemctl", []string{"daemon-reload"}); err != nil {
		log.Error("unable to reaload the daemon set: ", err)
		log.Error("Stderr: ", output.Stderr)
		return err
	}
	log.Debug("Successfully stopped the container service ", serviceName)
	return nil
}

func copyFile(isPrivileged bool, src string, dst string) (int64, error) {
	log.Debug(fmt.Sprintf("Copying %s into %s", src, dst))

	if _, err := RunCommand(isPrivileged, "cp", []string{src, dst}); err != nil {
		log.Error(fmt.Sprintf("failed to copy %s to %s", src, dst))
		return -1, err
	}
	log.Debug(fmt.Sprintf("Successfully copied %s to %s", src, dst))
	return 0, nil
}

func RunCommand(isPrivileged bool, command string, args []string) (*cli.PodmanOutput, error) {
	var cmd *exec.Cmd

	if isPrivileged {
		cmdArgs := "sudo " + command + " " + strings.Join(args, " ")
		cmd = exec.Command("sh", "-c", cmdArgs)
	} else {
		cmd = exec.Command(command, args...)
	}

	log.Debugf("Command being run: %+v", cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error(fmt.Sprintf("unable to run command %+v ", command), err)
		log.Error("Stderr: ", stderr.String())
		return &cli.PodmanOutput{
			Stderr: stderr.String(),
		}, err
	}
	return &cli.PodmanOutput{
		Stdout: stdout.String(),
	}, nil
}
