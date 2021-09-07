// Package engine contains the interfaces necessary to implement policy execution.
package engine

import (
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	internal "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/k8s"
	containerpol "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/policy/container"
	operatorpol "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/policy/operator"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/shell"
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
		check := queryNewChecks(checkString)
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

func NewShellEngineForConfig(config runtime.Config) (CheckEngine, error) {
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

	var engine CheckEngine
	engine = &shell.CheckEngine{
		Image:  config.Image,
		Checks: checks,
		Bundle: config.Bundle,
	}
	if config.Mounted {
		engine = &shell.MountedCheckEngine{
			Image: config.Image,
			Check: checks[0],
		}
	}
	return engine, nil
}

// queryChecks queries Operator and Container checks by name, and return certification.Check
// if found; nil otherwise
func queryChecks(checkName string) certification.Check {
	// query Operator checks
	if check, exists := oldOperatorPolicy[checkName]; exists {
		return check
	}
	// if not found in Operator Policy, query container policy
	if check, exists := oldContainerPolicy[checkName]; exists {
		return check
	}

	// Lastly, look at the mounted checks
	if check, exists := unshareChecks[checkName]; exists {
		return check
	}

	return nil
}

// queryNewChecks queries Operator and Container checks by name, and return certification.Check
// if found; nil otherwise. This will be collapsed when old checks are all deprecated.
func queryNewChecks(checkName string) certification.Check {
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
var deprecatedRunAsNonRootCheck certification.Check = &shell.RunAsNonRootCheck{}
var deprecatedUnderLayerMaxCheck certification.Check = &shell.UnderLayerMaxCheck{}
var deprecatedHasRequiredLabelCheck certification.Check = &shell.HasRequiredLabelsCheck{}
var basedOnUbiCheck certification.Check = &shell.BaseOnUBICheck{}
var deprecatedHasLicenseCheck certification.Check = &shell.HasLicenseCheck{}
var deprecatedHasUniqueTagCheck certification.Check = &shell.HasUniqueTagCheck{}
var deprecatedValidateOperatorBundle certification.Check = &shell.ValidateOperatorBundleCheck{}
var deprecatedScorecardBasicSpecCheck certification.Check = &shell.ScorecardBasicSpecCheck{}
var deprecatedScorecardOlmSuiteCheck certification.Check = &shell.ScorecardOlmSuiteCheck{}
var deprecatedHasNoProhibitedCheck certification.Check = &shell.HasNoProhibitedPackagesCheck{}
var deprecatedHasNoProhibitedMountedCheck certification.Check = &shell.HasNoProhibitedPackagesMountedCheck{}
var deprecatedOperatorPkgNameIsUniqueMountedCheck certification.Check = &shell.OperatorPkgNameIsUniqueMountedCheck{}
var deprecatedOperatorPkgNameIsUniqueCheck certification.Check = &shell.OperatorPkgNameIsUniqueCheck{}
var hasMinimalVulnerabilitiesUnshareCheck certification.Check = &shell.HasMinimalVulnerabilitiesUnshareCheck{}
var deprecatedDeployableByOlmCheck certification.Check = &k8s.DeployableByOlmCheck{}
var deprecatedDeployableByOlmMountedCheck certification.Check = &k8s.DeployableByOlmMountedCheck{}

// Disabled due to issue #99 and discussions in community meeting
// var hasMinimalVulnerabilitiesCheck certification.Check = &shell.HasMinimalVulnerabilitiesCheck{}

// new checks for CraneEngine
var hasLicenseCheck certification.Check = &containerpol.HasLicenseCheck{}
var hasUniqueTagCheck certification.Check = containerpol.NewHasUniqueTagCheck(internal.NewSkopeoEngine())
var deployableByOlmCheck certification.Check = &operatorpol.DeployableByOlmCheck{}
var operatorPkgNameIsUniqueCheck certification.Check = &operatorpol.OperatorPkgNameIsUniqueCheck{}
var validateOperatorBundle certification.Check = operatorpol.NewValidateOperatorBundleCheck(internal.NewOperatorSdkEngine())
var scorecardBasicSpecCheck certification.Check = operatorpol.NewScorecardBasicSpecCheck(internal.NewOperatorSdkEngine())
var scorecardOlmSuiteCheck certification.Check = operatorpol.NewScorecardOlmSuiteCheck(internal.NewOperatorSdkEngine())
var maxLayersCheck certification.Check = &containerpol.MaxLayersCheck{}
var hasNoProhibitedCheck certification.Check = &containerpol.HasNoProhibitedPackagesCheck{}
var hasRequiredLabelsCheck certification.Check = &containerpol.HasRequiredLabelsCheck{}
var runAsRootCheck certification.Check = &containerpol.RunAsNonRootCheck{}

var operatorPolicy = map[string]certification.Check{
	operatorPkgNameIsUniqueCheck.Name(): operatorPkgNameIsUniqueCheck,
	scorecardBasicSpecCheck.Name():      scorecardBasicSpecCheck,
	scorecardOlmSuiteCheck.Name():       scorecardOlmSuiteCheck,
	deployableByOlmCheck.Name():         deployableByOlmCheck,
	validateOperatorBundle.Name():       validateOperatorBundle,
}

var containerPolicy = map[string]certification.Check{
	hasLicenseCheck.Name():        hasLicenseCheck,
	hasUniqueTagCheck.Name():      hasUniqueTagCheck,
	maxLayersCheck.Name():         maxLayersCheck,
	hasLicenseCheck.Name():        hasLicenseCheck,
	hasNoProhibitedCheck.Name():   hasNoProhibitedCheck,
	hasRequiredLabelsCheck.Name(): hasRequiredLabelsCheck,
	runAsRootCheck.Name():         runAsRootCheck,
}

var oldContainerPolicy = map[string]certification.Check{
	deprecatedRunAsNonRootCheck.Name():     deprecatedRunAsNonRootCheck,
	deprecatedUnderLayerMaxCheck.Name():    deprecatedUnderLayerMaxCheck,
	deprecatedHasRequiredLabelCheck.Name(): deprecatedHasRequiredLabelCheck,
	basedOnUbiCheck.Name():                 basedOnUbiCheck,
	deprecatedHasLicenseCheck.Name():       deprecatedHasLicenseCheck,
	deprecatedHasUniqueTagCheck.Name():     deprecatedHasUniqueTagCheck,
	deprecatedHasNoProhibitedCheck.Name():  deprecatedHasNoProhibitedCheck,
	// Disabled due to issue #99 and discussions in community meeting
	// hasMinimalVulnerabilitiesCheck.Name(): hasMinimalVulnerabilitiesCheck,
}

var oldOperatorPolicy = map[string]certification.Check{
	deprecatedValidateOperatorBundle.Name():       deprecatedValidateOperatorBundle,
	deprecatedScorecardBasicSpecCheck.Name():      deprecatedScorecardBasicSpecCheck,
	deprecatedScorecardOlmSuiteCheck.Name():       deprecatedScorecardOlmSuiteCheck,
	deprecatedOperatorPkgNameIsUniqueCheck.Name(): operatorPkgNameIsUniqueCheck,
	deployableByOlmCheck.Name():                   deployableByOlmCheck,
}

var unshareChecks = map[string]certification.Check{
	deprecatedOperatorPkgNameIsUniqueMountedCheck.Name(): deprecatedOperatorPkgNameIsUniqueMountedCheck,
	deprecatedDeployableByOlmCheck.Name():                deprecatedDeployableByOlmCheck,
	deprecatedHasNoProhibitedMountedCheck.Name():         deprecatedHasNoProhibitedMountedCheck,
	hasMinimalVulnerabilitiesUnshareCheck.Name():         hasMinimalVulnerabilitiesUnshareCheck,
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

func OldOperatorPolicy() []string {
	return makeCheckList(oldOperatorPolicy)
}

func OldContainerPolicy() []string {
	return makeCheckList(oldContainerPolicy)
}

func OperatorPolicy() []string {
	return makeCheckList(operatorPolicy)
}

func ContainerPolicy() []string {
	return makeCheckList(containerPolicy)
}
