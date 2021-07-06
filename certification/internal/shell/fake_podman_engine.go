package shell

import "github.com/redhat-openshift-ecosystem/openshift-preflight/cli"

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
