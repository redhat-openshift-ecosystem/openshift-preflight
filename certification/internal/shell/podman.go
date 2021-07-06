package shell

import (
	"bytes"
	"os/exec"

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
	return nil
}
