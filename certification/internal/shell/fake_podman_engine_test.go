package shell

import (
	"errors"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

// This struct is meant to implement cli.PodmanEngine
// It is used for unit testing, allowing the package-level
// variable of podmanEngine to be overridden in test files

type FakePodmanEngine struct {
	RunReportStdout     string
	RunReportStderr     string
	PullReportStdouterr string
}

func (fpe FakePodmanEngine) Run(opts cli.ImageRunOptions) (*cli.ImageRunReport, error) {
	runReport := cli.ImageRunReport{
		Stdout: fpe.RunReportStdout,
		Stderr: fpe.RunReportStderr,
	}
	return &runReport, nil
}

func (fpe FakePodmanEngine) Pull(rawImage string, opts cli.ImagePullOptions) (*cli.ImagePullReport, error) {
	pullReport := cli.ImagePullReport{
		StdoutErr: fpe.PullReportStdouterr,
	}

	return &pullReport, nil
}

func (fpe FakePodmanEngine) Save(nameOrID string, tags []string, opts cli.ImageSaveOptions) error {
	return nil
}

type BadPodmanEngine struct{}

func (bpe BadPodmanEngine) Run(cli.ImageRunOptions) (*cli.ImageRunReport, error) {
	runReport := cli.ImageRunReport{
		Stdout: "Bad stadout",
		Stderr: "Bad stderr",
	}
	return &runReport, errors.New("the Podman Run has failed")
}

func (bpe BadPodmanEngine) Pull(rawImage string, opts cli.ImagePullOptions) (*cli.ImagePullReport, error) {
	return nil, errors.New("the Podman Pull has failed")
}

func (bpe BadPodmanEngine) Save(nameOrID string, tags []string, opts cli.ImageSaveOptions) error {
	return errors.New("the Podman Save has failed")
}
