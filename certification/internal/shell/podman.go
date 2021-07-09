package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

type PodmanCLIEngine struct{}

func (pe PodmanCLIEngine) Pull(rawImage string, opts cli.ImagePullOptions) (*cli.ImagePullReport, error) {
	stdouterr, err := exec.Command("podman", "pull", rawImage).CombinedOutput()
	if err != nil {
		return nil, err
	}

	return &cli.ImagePullReport{StdoutErr: string(stdouterr)}, nil
}

func (pe PodmanCLIEngine) Run(opts cli.ImageRunOptions) (*cli.ImageRunReport, error) {
	cmdArgs := []string{"run", "-it", "--rm", "--log-level", opts.LogLevel, "--entrypoint", opts.EntryPoint, opts.Image}
	cmdArgs = append(cmdArgs, opts.EntryPointArgs...)
	cmd := exec.Command("podman", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return &cli.ImageRunReport{Stdout: stdout.String(), Stderr: stderr.String()}, err
	}
	return &cli.ImageRunReport{Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

func (pe PodmanCLIEngine) Save(nameOrID string, tags []string, opts cli.ImageSaveOptions) error {
	cmdArgs := []string{"save", "--output", opts.Destination}
	cmdArgs = append(cmdArgs, nameOrID)
	_, err := exec.Command("podman", cmdArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrSaveContainerFailed, err)
	}
	return nil
}

func (pe PodmanCLIEngine) InspectImage(rawImage string, opts cli.ImageInspectOptions) (*cli.ImageInspectReport, error) {
	cmdArgs := []string{"image", "inspect"}
	if opts.LogLevel != "" {
		cmdArgs = append(cmdArgs, "--log-level", opts.LogLevel)
	}
	cmdArgs = append(cmdArgs, rawImage)

	cmd := exec.Command("podman", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrImageInspectFailed, err)
	}

	var inspectData []cli.PodmanImage
	err = json.Unmarshal(stdout.Bytes(), &inspectData)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrImageInspectFailed, err)
	}
	return &cli.ImageInspectReport{Images: inspectData}, nil
}
