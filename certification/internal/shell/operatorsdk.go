package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	fileutils "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/file"
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
	if opts.Kubeconfig != "" {
		cmdArgs = append(cmdArgs, "--kubeconfig", opts.Kubeconfig)
	}
	if opts.Namespace != "" {
		cmdArgs = append(cmdArgs, "--namespace", opts.Namespace)
	}
	if opts.ServiceAccount != "" {
		cmdArgs = append(cmdArgs, "--service-account", opts.ServiceAccount)
	}

	configFile, err := createScorecardConfigFile()
	if err != nil {
		log.Error("could not create scorecard config file", err)
		return nil, err
	}
	defer os.Remove(configFile)
	cmdArgs = append(cmdArgs, "--config", configFile)

	cmdArgs = append(cmdArgs, image)

	cmd := exec.Command("operator-sdk", cmdArgs...)
	log.Trace("running scorecard with the following invocation", cmd.Args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		// This is a workaround due to operator-sdk scorecard always returning a 1 exit code
		// whether a test failed or the tool encountered a fatal error.
		//
		// Until resolved, we are concluding/assuming that non-zero exit codes with len(stderr) == 0
		// means that we failed a test, but the command execution succeeded.
		//
		// We also conclude/assume that anything being in stderr would indicate an error in the
		// check execution itself.
		if stderr.Len() != 0 {
			log.Error("stderr: ", stdout.String())
			return nil, fmt.Errorf("%w: %s", errors.ErrOperatorSdkScorecardFailed, err)
		}
	}

	err = o.writeScorecardFile(opts.ResultFile, stdout.String())
	if err != nil {
		log.Error("unable to copy result to artifacts directory: ", err)
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

func (o OperatorSdkCLIEngine) BundleValidate(image string, opts cli.OperatorSdkBundleValidateOptions) (*cli.OperatorSdkBundleValidateReport, error) {
	cmdArgs := []string{"bundle", "validate"}
	if opts.ContainerEngine == "" {
		opts.ContainerEngine = "podman"
	}
	cmdArgs = append(cmdArgs, "-b", opts.ContainerEngine)
	if opts.OutputFormat == "" {
		opts.OutputFormat = "json-alpha1"
	}
	cmdArgs = append(cmdArgs, "--output", opts.OutputFormat)
	if opts.Selector != nil {
		for _, selector := range opts.Selector {
			cmdArgs = append(cmdArgs, "--select-optional", fmt.Sprintf("name=%s", selector))
		}
	}
	if opts.Verbose {
		cmdArgs = append(cmdArgs, "--verbose")
	}
	cmdArgs = append(cmdArgs, image)

	cmd := exec.Command("operator-sdk", cmdArgs...)
	log.Debugf("Command being run: %s", cmd.Args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Debugf("Stderr: %s", stderr.String())
		return &cli.OperatorSdkBundleValidateReport{Stderr: stderr.String()}, fmt.Errorf("%w: %s", errors.ErrOperatorSdkBundleValidateFailed, err)
	}

	var bundleValidateData cli.OperatorSdkBundleValidateReport
	if strings.Contains(opts.OutputFormat, "json") {
		err = json.Unmarshal(stdout.Bytes(), &bundleValidateData)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errors.ErrOperatorSdkBundleValidateFailed, err)
		}
	} else {
		if strings.Contains(stdout.String(), "ERRO") || strings.Contains(stdout.String(), "FATA") {
			bundleValidateData.Passed = false
		}
	}
	bundleValidateData.Stdout = stdout.String()
	bundleValidateData.Stderr = stderr.String()

	return &bundleValidateData, nil
}

func (o OperatorSdkCLIEngine) writeScorecardFile(resultFile, stdout string) error {
	scorecardFile := fileutils.ArtifactPath(resultFile)

	err := ioutil.WriteFile(scorecardFile, []byte(stdout), 0644)
	if err != nil {
		return err
	}
	return nil
}

func createScorecardConfigFile() (string, error) {
	configTemplate := `kind: Configuration
	apiversion: scorecard.operatorframework.io/v1alpha3
	metadata:
	  name: config
	stages:
	  - parallel: true
		tests:
		  - image: quay.io/operator-framework/scorecard-test:v1.9.0
			entrypoint:
			  - scorecard-test
			  - basic-check-spec
			labels:
			  suite: basic
			  test: basic-check-spec-test
		  - image: quay.io/operator-framework/scorecard-test:v1.9.0
			entrypoint:
			  - scorecard-test
			  - olm-bundle-validation
			labels:
			  suite: olm
			  test: olm-bundle-validation-test
`

	tempConfigFile, err := os.CreateTemp("", "scorecard-test-config-*.yaml")
	if err != nil {
		log.Error("could not create temp config file", err)
		return "", err
	}
	_, err = tempConfigFile.WriteString(configTemplate)
	return tempConfigFile.Name(), err
}
