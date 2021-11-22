package e2e

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestPreflightHelp(t *testing.T) {
	got := preflightHelp()
	want := "A utility that allows you to pre-test your bundles, operators, and container before submitting for Red Hat Certification.\n\nUsage:\n  preflight [command]\n\nAvailable Commands:\n  certify        Submits check results to Red Hat\n  check          Run checks for an operator or container\n  completion     generate the autocompletion script for the specified shell\n  help           Help about any command\n  runtime-assets Returns information about assets used at runtime.\n  support        Submits a support request\n\nFlags:\n  -h, --help      help for preflight\n  -v, --version   version for preflight\n\nUse \"preflight [command] --help\" for more information about a command.\n"

	if got != want {
		t.Errorf("got %q \n\n\n\nwant %q", got, want)
	}
}

func TestPreflightCheck(t *testing.T) {
	got := preflightCheck()
	want := "This command will allow you to execute the Red Hat Certification tests for an operator or a container.\n\nUsage:\n  preflight check [command]\n\nAvailable Commands:\n  container   Run checks for a container\n  operator    Run checks for an Operator\n\nFlags:\n  -h, --help   help for check\n\nUse \"preflight check [command] --help\" for more information about a command.\n"

	if got != want {
		t.Errorf("got %q \n\n\n\nwant %q", got, want)
	}
}

func TestPreflightCheckContainer(t *testing.T) {
	got := preflightCheckContainer()
	want := "Error: not enough positional arguments: A container image positional argument is required\n\nThe checks that will be executed are the following:\n-"

	container_checks := []string{"LayerCountAcceptable", "HasNoProhibitedPackagesMounted", "HasRequiredLabel", "RunAsNonRoot", "BasedOnUbi", "HasLicense", "HasUniqueTag"}
	flag := true
	for _, check := range container_checks {
		if !(strings.Contains(got, check)) {
			flag = false
		}
	}

	if !(strings.Contains(got, want) && flag) {
		t.Errorf("Missing few Container Checks")
	}
}

func TestPreflightCheckOperator(t *testing.T) {
	got := preflightCheckOperator()
	want := "Error: not enough positional arguments: An operator image positional argument is required\n\nThe checks that will be executed are the following:\n-"

	operator_checks := []string{"ValidateOperatorBundle", "ScorecardBasicSpecCheck", "ScorecardOlmSuiteCheck", "DeployableByOLM"}
	flag := true
	for _, check := range operator_checks {
		if !(strings.Contains(got, check)) {
			flag = false
		}
	}

	if !(strings.Contains(got, want) && flag) {
		t.Errorf("Missing few Operator checks")
	}

}

/*func TestPreflightSupport(t *testing.T) {
    got := preflightSupport()
    fmt.Println(got)
    want := "Error: not enough positional arguments: An operator image positional argument is required\n\nThe checks that will be executed are the following:\n-"

    if !(strings.Contains(got,want)) {
        t.Errorf("got %q \n\n\n\nwant %q", got, want)
    }
}*/

func TestPreflightContainerRun(t *testing.T) {
	preflightContainerRun()
	content, err := ioutil.ReadFile("artifacts/results.json")
	if err != nil {
		t.Errorf("Error reading results.json file")
	}
	var payload map[string]interface{}
	err = json.Unmarshal(content, &payload)
	if err != nil {
		t.Errorf("Error reading results.json file")
	}
	var result bool
	result = payload["passed"].(bool)
	fmt.Println(result)
	if !(result == true) {
		t.Errorf("Some Tests failed")
	}
}

func TestContainerArtifcats(t *testing.T) {
	_, err := os.OpenFile("artifacts/results.json", os.O_RDONLY, 0)
	if errors.Is(err, os.ErrNotExist) {
		t.Errorf("No artifacts generated for Container run")
	}
	os.RemoveAll("artifacts")
}

func TestPreflightOperatorRun(t *testing.T) {
	preflightOperatorRun()
	content, err := ioutil.ReadFile("artifacts/results.json")
	if err != nil {
		t.Errorf("Error reading results.json file")
	}
	var payload map[string]interface{}
	err = json.Unmarshal(content, &payload)
	if err != nil {
		t.Errorf("Error reading resuls.json file")
	}
	var result bool
	result = payload["passed"].(bool)
	fmt.Println(result)
	if !(result == true) {
		t.Errorf("Some Tests failed")
	}
}

func TestOperatorArtifcats(t *testing.T) {
	_, err := os.OpenFile("artifacts/results.json", os.O_RDONLY, 0)
	if errors.Is(err, os.ErrNotExist) {
		t.Errorf("No artifacts generated for Operator run")
	}
	os.RemoveAll("artifacts")
}
