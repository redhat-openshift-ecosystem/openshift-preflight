package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

type OperatorSdkCLIEngine struct{}

func (o OperatorSdkCLIEngine) Scorecard(image string, opts cli.OperatorSdkScorecardOptions) (*cli.OperatorSdkScorecardReport, error) {
	cmdArgs := []string{"scorecard"}
	if opts.OutputFormat == "" {
		opts.OutputFormat = "json"
	}
	cmdArgs = append(cmdArgs, "--output", opts.OutputFormat)
	if opts.Selector != nil {
		for _, selector := range opts.Selector {
			cmdArgs = append(cmdArgs, fmt.Sprintf("--selector=%s", selector))
		}
	}
	cmdArgs = append(cmdArgs, image)

	artifactsDir, err := o.createArtifactsDir()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("operator-sdk", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrOperatorSdkScorecardFailed, err)
	}

	err = o.writeScorecardFile(artifactsDir, opts.ResultFile, stdout.String())
	if err != nil {
		log.Error("unable to copy result to /artifacts subdir: ", err)
		return nil, err
	}

	var scorecardData cli.OperatorSdkScorecardReport
	err = json.Unmarshal(stdout.Bytes(), &scorecardData)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrOperatorSdkScorecardFailed, err)
	}
	scorecardData.Stdout = stdout.String()
	scorecardData.Stderr = stderr.String()
	return &scorecardData, nil
}

func (o OperatorSdkCLIEngine) writeScorecardFile(artifactsDir, resultFile, stdout string) error {
	scorecardFile := filepath.Join(artifactsDir, "/", resultFile)

	err := ioutil.WriteFile(scorecardFile, []byte(stdout), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (o OperatorSdkCLIEngine) createArtifactsDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Error("unable to get current directory: ", err)
		return "", err
	}

	artifactsDir := filepath.Join(currentDir, "/artifacts")

	err = os.MkdirAll(artifactsDir, 0777)
	if err != nil {
		log.Error("unable to create artifactsDir: ", err)
		return "", err
	}
	return artifactsDir, nil
}
