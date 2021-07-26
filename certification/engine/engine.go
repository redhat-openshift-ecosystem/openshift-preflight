package engine

import (
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/shell"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
)

// CheckEngine defines the functonality necessary to run all checks for a policy,
// and return the results of that check execution.
type CheckEngine interface {
	// ExecuteChecks should execute all checks in a policy and internally
	// store the results. Errors returned by ExecuteChecks should reflect
	// errors in pre-validation tasks, and not errors in individual check
	// execution itself.
	ExecuteChecks() error
	// Results returns the outcome of executing all checks.
	Results() runtime.Results
}

func NewForConfig(config runtime.Config) (CheckEngine, error) {
	if len(config.EnabledChecks) == 0 {
		// refuse to run if the user has not specified any checks
		return nil, errors.ErrNoChecksEnabled
	}

	checks := make([]certification.Check, len(config.EnabledChecks))
	for i, checkString := range config.EnabledChecks {
		check := queryChecks(checkString)
		if check == nil {
			err := fmt.Errorf("%w: %s",
				errors.ErrRequestedCheckNotFound,
				checkString)
			return nil, err
		}

		checks[i] = check
	}

	engine := &shell.CheckEngine{
		Image:  config.Image,
		Checks: checks,
	}

	return engine, nil
}

// queryChecks queries Operator and Container checks by name, and return certification.Check
// if found; nil otherwise
func queryChecks(checkName string) certification.Check {
	// query Operator checks
	check, exists := operatorPolicy[checkName]
	if exists {
		return check
	}
	// if not found in Operator Policy, query container policy
	check, exists = containerPolicy[checkName]
	if exists {
		return check
	}

	return nil
}

// Register all checks
var runAsNonRootCheck certification.Check = &shell.RunAsNonRootCheck{}
var underLayerMaxCheck certification.Check = &shell.UnderLayerMaxCheck{}
var hasRequiredLabelCheck certification.Check = &shell.HasRequiredLabelsCheck{}
var basedOnUbiCheck certification.Check = &shell.BaseOnUBICheck{}
var hasLicenseCheck certification.Check = &shell.HasLicenseCheck{}
var hasMinimalVulnerabilitiesCheck certification.Check = &shell.HasMinimalVulnerabilitiesCheck{}
var hasUniqueTagCheck certification.Check = &shell.HasUniqueTagCheck{}
var hasNoProhibitedCheck certification.Check = &shell.HasNoProhibitedPackagesCheck{}
var validateOperatorBundle certification.Check = &shell.ValidateOperatorBundleCheck{}
var scorecardBasicSpecCheck certification.Check = &shell.ScorecardBasicSpecCheck{}
var scorecardOlmSuiteCheck certification.Check = &shell.ScorecardOlmSuiteCheck{}

var containerPolicy = map[string]certification.Check{
	runAsNonRootCheck.Name():              runAsNonRootCheck,
	underLayerMaxCheck.Name():             underLayerMaxCheck,
	hasRequiredLabelCheck.Name():          hasRequiredLabelCheck,
	basedOnUbiCheck.Name():                basedOnUbiCheck,
	hasLicenseCheck.Name():                hasLicenseCheck,
	hasMinimalVulnerabilitiesCheck.Name(): hasMinimalVulnerabilitiesCheck,
	hasUniqueTagCheck.Name():              hasUniqueTagCheck,
	hasNoProhibitedCheck.Name():           hasNoProhibitedCheck,
}

var operatorPolicy = map[string]certification.Check{
	validateOperatorBundle.Name():  validateOperatorBundle,
	scorecardBasicSpecCheck.Name(): scorecardBasicSpecCheck,
	scorecardOlmSuiteCheck.Name():  scorecardOlmSuiteCheck,
}

func makeCheckList(checkMap map[string]certification.Check) []string {
	checks := make([]string, len(checkMap))
	i := 0

	for key := range checkMap {
		checks[i] = key
		i++
	}

	return checks
}

func OperatorPolicy() []string {
	return makeCheckList(operatorPolicy)
}

func ContainerPolicy() []string {
	return makeCheckList(containerPolicy)
}
