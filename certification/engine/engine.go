// Package engine contains the interfaces necessary to implement policy execution.
package engine

import (
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	internal "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/engine"
	containerpol "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/policy/container"
	operatorpol "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/policy/operator"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
)

// CheckEngine defines the functionality necessary to run all checks for a policy,
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

	engine := &internal.CraneEngine{
		Image:    config.Image,
		Checks:   checks,
		IsBundle: config.Bundle,
	}

	return engine, nil
}

// queryChecks queries Operator and Container checks by name, and return certification.Check
// if found; nil otherwise. This will be collapsed when old checks are all deprecated.
func queryChecks(checkName string) certification.Check {
	// query Operator checks
	if check, exists := operatorPolicy[checkName]; exists {
		return check
	}
	// if not found in Operator Policy, query container policy
	if check, exists := containerPolicy[checkName]; exists {
		return check
	}

	return nil
}

// Register all checks

// Operator checks
var operatorPkgNameIsUniqueCheck certification.Check = &operatorpol.OperatorPkgNameIsUniqueCheck{}
var scorecardBasicSpecCheck certification.Check = operatorpol.NewScorecardBasicSpecCheck(internal.NewOperatorSdkEngine())
var scorecardOlmSuiteCheck certification.Check = operatorpol.NewScorecardOlmSuiteCheck(internal.NewOperatorSdkEngine())
var deployableByOlmCheck certification.Check = operatorpol.NewDeployableByOlmCheck(internal.NewOpenshiftEngine())
var validateOperatorBundle certification.Check = operatorpol.NewValidateOperatorBundleCheck(internal.NewOperatorSdkEngine())

// Container checks
var hasLicenseCheck certification.Check = &containerpol.HasLicenseCheck{}
var hasUniqueTagCheck certification.Check = containerpol.NewHasUniqueTagCheck(internal.NewCraneEngine())
var maxLayersCheck certification.Check = &containerpol.MaxLayersCheck{}
var hasNoProhibitedCheck certification.Check = &containerpol.HasNoProhibitedPackagesCheck{}
var hasRequiredLabelsCheck certification.Check = &containerpol.HasRequiredLabelsCheck{}
var runAsRootCheck certification.Check = &containerpol.RunAsNonRootCheck{}
var basedOnUbiCheck certification.Check = &containerpol.BasedOnUBICheck{}

var operatorPolicy = map[string]certification.Check{
	//operatorPkgNameIsUniqueCheck.Name(): operatorPkgNameIsUniqueCheck,
	scorecardBasicSpecCheck.Name(): scorecardBasicSpecCheck,
	scorecardOlmSuiteCheck.Name():  scorecardOlmSuiteCheck,
	deployableByOlmCheck.Name():    deployableByOlmCheck,
	validateOperatorBundle.Name():  validateOperatorBundle,
}

var containerPolicy = map[string]certification.Check{
	hasLicenseCheck.Name():        hasLicenseCheck,
	hasUniqueTagCheck.Name():      hasUniqueTagCheck,
	maxLayersCheck.Name():         maxLayersCheck,
	hasNoProhibitedCheck.Name():   hasNoProhibitedCheck,
	hasRequiredLabelsCheck.Name(): hasRequiredLabelsCheck,
	runAsRootCheck.Name():         runAsRootCheck,
	basedOnUbiCheck.Name():        basedOnUbiCheck,
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
