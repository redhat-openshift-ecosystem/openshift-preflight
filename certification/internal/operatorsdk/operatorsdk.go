package operatorsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"

	log "github.com/sirupsen/logrus"
)

func New(userProvidedScorecardImage string, cmdContext execContext) *operatorSdk {
	engine := operatorSdk{scorecardImage: userProvidedScorecardImage, cmdContext: cmdContext}
	return &engine
}

type operatorSdk struct {
	scorecardImage string
	cmdContext     execContext
}

// Define a type that is the signature of the exec.Command function.
// This allows us to override that function with our own for
// testing purposes. This type is only used directly in the New() function.
type execContext = func(name string, arg ...string) *exec.Cmd

func (o operatorSdk) Scorecard(ctx context.Context, image string, opts OperatorSdkScorecardOptions) (*OperatorSdkScorecardReport, error) {
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
	if opts.WaitTime != "" {
		cmdArgs = append(cmdArgs, "--wait-time", opts.WaitTime)
	}
	if opts.Namespace != "" {
		cmdArgs = append(cmdArgs, "--namespace", opts.Namespace)
	}
	if opts.ServiceAccount != "" {
		cmdArgs = append(cmdArgs, "--service-account", opts.ServiceAccount)
	}

	configFile, err := o.createScorecardConfigFile()
	defer os.Remove(configFile)
	if err != nil {
		return nil, fmt.Errorf("could not create scorecard config file: %v", err)
	}
	cmdArgs = append(cmdArgs, "--config", configFile)
	if opts.Verbose {
		cmdArgs = append(cmdArgs, "--verbose")
	}

	cmdArgs = append(cmdArgs, image)

	cmd := o.cmdContext("operator-sdk", cmdArgs...)
	log.Trace("running scorecard with the following invocation", cmd.Args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// This is a workaround due to operator-sdk scorecard always returning a 1 exit code
		// whether a test failed or the tool encountered a fatal error.
		//
		// Until resolved, we are concluding/assuming that non-zero exit codes with len(stderr) == 0
		// means that we failed a test, but the command execution succeeded.
		//
		// We also conclude/assume that "FATA" being in stderr would indicate an error in the
		// check execution itself.
		if stderr.Len() != 0 && strings.Contains(strings.ToUpper(stderr.String()), "FATA") {
			log.Debug("operator-sdk scorecard failed to run properly.")
			log.Debugf("stderr: %s", stderr.String())

			return nil, fmt.Errorf("failed to run operator-sdk scorecard: %v", err)
		}
	}

	if err := o.writeScorecardFile(opts.ResultFile, stdout.String()); err != nil {
		return nil, fmt.Errorf("unable to copy result to artifacts directory: %v", err)
	}

	var scorecardData OperatorSdkScorecardReport
	if err := json.Unmarshal(stdout.Bytes(), &scorecardData); err != nil {
		return nil, fmt.Errorf("failed to run operator-sdk scorecard: %v", err)
	}
	scorecardData.Stdout = stdout.String()
	scorecardData.Stderr = stderr.String()
	return &scorecardData, nil
}

func (o operatorSdk) BundleValidate(ctx context.Context, image string, opts OperatorSdkBundleValidateOptions) (*OperatorSdkBundleValidateReport, error) {
	cmdArgs := []string{"bundle", "validate"}
	if opts.ContainerEngine == "" {
		opts.ContainerEngine = "none"
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
	if opts.OptionalValues != nil {
		for key, value := range opts.OptionalValues {
			cmdArgs = append(cmdArgs, "--optional-values", fmt.Sprintf("%s=%s", key, value))
		}
	}
	if opts.Verbose {
		cmdArgs = append(cmdArgs, "--verbose")
	}
	cmdArgs = append(cmdArgs, image)

	cmd := o.cmdContext("operator-sdk", cmdArgs...)
	log.Debugf("Command being run: %s", cmd.Args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// This is a workaround due to operator-sdk scorecard always returning a 1 exit code
		// whether a test failed or the tool encountered a fatal error.
		//
		// Until resolved, we are concluding/assuming that non-zero exit codes with len(stderr) == 0
		// means that we failed a test, but the command execution succeeded.
		//
		// We also conclude/assume that "FATA" being in stderr would indicate an error in the
		// check execution itself.
		if stderr.Len() != 0 && strings.Contains(stderr.String(), "FATA") {
			log.Debugf("stdout: %s", stdout.String())
			log.Debugf("stderr: %s", stderr.String())
			return nil, fmt.Errorf("failed to run operator-sdk bundle validate: %v", err)
		}
	}

	var bundleValidateData OperatorSdkBundleValidateReport
	if strings.Contains(opts.OutputFormat, "json") {
		if err := json.Unmarshal(stdout.Bytes(), &bundleValidateData); err != nil {
			return nil, fmt.Errorf("failed to run operator-sdk bundle validate: %v", err)
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

func (o operatorSdk) writeScorecardFile(resultFile, stdout string) error {
	_, err := artifacts.WriteFile(resultFile, strings.NewReader(stdout))
	return err
}

func (o operatorSdk) createScorecardConfigFile() (string, error) {
	img := runtime.ScorecardImage(o.scorecardImage)
	configTemplate := fmt.Sprintf(`kind: Configuration
apiversion: scorecard.operatorframework.io/v1alpha3
metadata:
  name: config
stages:
- parallel: true
  tests:
  - image: %s
    entrypoint:
      - scorecard-test
      - basic-check-spec
    labels:
      suite: basic
      test: basic-check-spec-test
  - image: %s
    entrypoint:
      - scorecard-test
      - olm-bundle-validation
    labels:
      suite: olm
      test: olm-bundle-validation-test
`, img, img)

	tempConfigFile, err := os.CreateTemp("", "scorecard-test-config-*.yaml")
	if err != nil {
		return "", fmt.Errorf("could not create temp config file: %w", err)
	}
	_, err = tempConfigFile.WriteString(configTemplate)
	return tempConfigFile.Name(), err
}
