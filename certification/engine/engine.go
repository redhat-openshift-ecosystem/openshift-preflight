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
var runAsNonRootCheck certification.Check = &shell.RunAsNonRootCheck{}
var underLayerMaxCheck certification.Check = &shell.UnderLayerMaxCheck{}
var hasRequiredLabelCheck certification.Check = &shell.HasRequiredLabelsCheck{}
var basedOnUbiCheck certification.Check = &shell.BaseOnUBICheck{}
var deprecatedHasLicenseCheck certification.Check = &shell.HasLicenseCheck{}
var hasMinimalVulnerabilitiesCheck certification.Check = &shell.HasMinimalVulnerabilitiesCheck{}
var hasUniqueTagCheck certification.Check = &shell.HasUniqueTagCheck{}
var hasNoProhibitedCheck certification.Check = &shell.HasNoProhibitedPackagesCheck{}
var validateOperatorBundle certification.Check = &shell.ValidateOperatorBundleCheck{}
var scorecardBasicSpecCheck certification.Check = &shell.ScorecardBasicSpecCheck{}
var scorecardOlmSuiteCheck certification.Check = &shell.ScorecardOlmSuiteCheck{}
var hasNoProhibitedMountedCheck certification.Check = &shell.HasNoProhibitedPackagesMountedCheck{}
var deprecatedRelatedImageManifestSchemaVersionCheck certification.Check = &shell.RelatedImagesAreSchemaVersion2Check{}
var deprecatedOperatorPkgNameIsUniqueMountedCheck certification.Check = &shell.OperatorPkgNameIsUniqueMountedCheck{}
var deprecatedOperatorPkgNameIsUniqueCheck certification.Check = &shell.OperatorPkgNameIsUniqueCheck{}
var hasMinimalVulnerabilitiesUnshareCheck certification.Check = &shell.HasMinimalVulnerabilitiesUnshareCheck{}
var deployableByOlmCheck certification.Check = &k8s.DeployableByOlmCheck{}
var deployableByOlmMountedCheck certification.Check = &k8s.DeployableByOlmMountedCheck{}

// new checks for CraneEngine
var hasLicenseCheck certification.Check = &containerpol.HasLicenseCheck{}
var relatedImageManifestSchemaVersionCheck certification.Check = &operatorpol.RelatedImagesAreSchemaVersion2Check{}
var operatorPkgNameIsUniqueCheck certification.Check = &operatorpol.OperatorPkgNameIsUniqueMountedCheck{}

var operatorPolicy = map[string]certification.Check{
	operatorPkgNameIsUniqueCheck.Name():           operatorPkgNameIsUniqueCheck,
	relatedImageManifestSchemaVersionCheck.Name(): relatedImageManifestSchemaVersionCheck,
	scorecardBasicSpecCheck.Name():                scorecardBasicSpecCheck,
	scorecardOlmSuiteCheck.Name():                 scorecardOlmSuiteCheck,
}

var containerPolicy = map[string]certification.Check{
	hasLicenseCheck.Name(): hasLicenseCheck,
}

var oldContainerPolicy = map[string]certification.Check{
	runAsNonRootCheck.Name():              runAsNonRootCheck,
	underLayerMaxCheck.Name():             underLayerMaxCheck,
	hasRequiredLabelCheck.Name():          hasRequiredLabelCheck,
	basedOnUbiCheck.Name():                basedOnUbiCheck,
	deprecatedHasLicenseCheck.Name():      deprecatedHasLicenseCheck,
	hasMinimalVulnerabilitiesCheck.Name(): hasMinimalVulnerabilitiesCheck,
	hasUniqueTagCheck.Name():              hasUniqueTagCheck,
	hasNoProhibitedCheck.Name():           hasNoProhibitedCheck,
}

var oldOperatorPolicy = map[string]certification.Check{
	validateOperatorBundle.Name():                           validateOperatorBundle,
	scorecardBasicSpecCheck.Name():                          scorecardBasicSpecCheck,
	scorecardOlmSuiteCheck.Name():                           scorecardOlmSuiteCheck,
	deprecatedRelatedImageManifestSchemaVersionCheck.Name(): deprecatedRelatedImageManifestSchemaVersionCheck,
	validateOperatorBundle.Name():                           validateOperatorBundle,
	deprecatedOperatorPkgNameIsUniqueCheck.Name():           operatorPkgNameIsUniqueCheck,
	deployableByOlmCheck.Name():                             deployableByOlmCheck,
}

var unshareChecks = map[string]certification.Check{
	hasNoProhibitedMountedCheck.Name():                   hasNoProhibitedMountedCheck,
	deprecatedOperatorPkgNameIsUniqueMountedCheck.Name(): deprecatedOperatorPkgNameIsUniqueMountedCheck,
	hasMinimalVulnerabilitiesUnshareCheck.Name():         hasMinimalVulnerabilitiesUnshareCheck,
	deployableByOlmMountedCheck.Name():                   deployableByOlmMountedCheck,
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
