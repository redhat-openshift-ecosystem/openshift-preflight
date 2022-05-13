// Package engine contains the interfaces necessary to implement policy execution.
package engine

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	internal "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/engine"
	containerpol "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/policy/container"
	operatorpol "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/policy/operator"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/pyxis"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/spf13/viper"
)

// CheckEngine defines the functionality necessary to run all checks for a policy,
// and return the results of that check execution.
type CheckEngine interface {
	// ExecuteChecks should execute all checks in a policy and internally
	// store the results. Errors returned by ExecuteChecks should reflect
	// errors in pre-validation tasks, and not errors in individual check
	// execution itself.
	ExecuteChecks(context.Context) error
	// Results returns the outcome of executing all checks.
	Results(context.Context) runtime.Results
}

func NewForConfig(config runtime.Config) (CheckEngine, error) {
	if len(config.EnabledChecks) == 0 {
		// refuse to run if the user has not specified any checks
		return nil, errors.ErrNoChecksEnabled
	}

	checks := make([]certification.Check, 0, len(config.EnabledChecks))
	for _, checkString := range config.EnabledChecks {
		check := queryChecks(checkString)
		if check == nil {
			err := fmt.Errorf("%w: %s",
				errors.ErrRequestedCheckNotFound,
				checkString)
			return nil, err
		}

		checks = append(checks, check)
	}

	engine := &internal.CraneEngine{
		Image:     config.Image,
		Checks:    checks,
		IsBundle:  config.Bundle,
		IsScratch: config.Scratch,
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
	// if not found in Operator Policy, query container policy.
	// No need to check scratch container policy since this is
	// a superset.
	if check, exists := containerPolicy[checkName]; exists {
		return check
	}

	return nil
}

// Register all checks

// Operator checks
var (
	// operatorPkgNameIsUniqueCheck certification.Check = &operatorpol.OperatorPkgNameIsUniqueCheck{}
	scorecardBasicSpecCheck certification.Check = operatorpol.NewScorecardBasicSpecCheck(internal.NewOperatorSdkEngine())
	scorecardOlmSuiteCheck  certification.Check = operatorpol.NewScorecardOlmSuiteCheck(internal.NewOperatorSdkEngine())
	deployableByOlmCheck    certification.Check = operatorpol.NewDeployableByOlmCheck(internal.NewOperatorSdkEngine())
	validateOperatorBundle  certification.Check = operatorpol.NewValidateOperatorBundleCheck(internal.NewOperatorSdkEngine())
)

// Container checks
var (
	hasLicenseCheck        certification.Check = &containerpol.HasLicenseCheck{}
	hasUniqueTagCheck      certification.Check = containerpol.NewHasUniqueTagCheck()
	maxLayersCheck         certification.Check = &containerpol.MaxLayersCheck{}
	hasNoProhibitedCheck   certification.Check = &containerpol.HasNoProhibitedPackagesCheck{}
	hasRequiredLabelsCheck certification.Check = &containerpol.HasRequiredLabelsCheck{}
	runAsRootCheck         certification.Check = &containerpol.RunAsNonRootCheck{}
	hasModifiedFilesCheck  certification.Check = &containerpol.HasModifiedFilesCheck{}

	// Since the Pyxis data for checking UBI is only correct in prod, force the use of external prod
	basedOnUbiCheck certification.Check = containerpol.NewBasedOnUbiCheck(pyxis.NewPyxisClient(
		certification.DefaultPyxisHost,
		viper.GetString("pyxis_api_token"),
		viper.GetString("certification_project_id"),
		&http.Client{Timeout: 60 * time.Second}))
	// runnableContainerCheck  certification.Check = containerpol.NewRunnableContainerCheck(internal.NewPodmanEngine())
	// runSystemContainerCheck certification.Check = containerpol.NewRunSystemContainerCheck(internal.NewPodmanEngine())
)

var operatorPolicy = map[string]certification.Check{
	// operatorPkgNameIsUniqueCheck.Name(): operatorPkgNameIsUniqueCheck,
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
	hasModifiedFilesCheck.Name():  hasModifiedFilesCheck,
	// runnableContainerCheck.Name():  runnableContainerCheck,
	// runSystemContainerCheck.Name(): runSystemContainerCheck,
}

var scratchContainerPolicy = map[string]certification.Check{
	hasLicenseCheck.Name():        hasLicenseCheck,
	hasUniqueTagCheck.Name():      hasUniqueTagCheck,
	maxLayersCheck.Name():         maxLayersCheck,
	hasRequiredLabelsCheck.Name(): hasRequiredLabelsCheck,
	runAsRootCheck.Name():         runAsRootCheck,
	// runnableContainerCheck.Name():  runnableContainerCheck,
	// runSystemContainerCheck.Name(): runSystemContainerCheck,
}

var rootExceptionContainerPolicy = map[string]certification.Check{
	hasLicenseCheck.Name():        hasLicenseCheck,
	hasUniqueTagCheck.Name():      hasUniqueTagCheck,
	maxLayersCheck.Name():         maxLayersCheck,
	hasNoProhibitedCheck.Name():   hasNoProhibitedCheck,
	hasRequiredLabelsCheck.Name(): hasRequiredLabelsCheck,
	basedOnUbiCheck.Name():        basedOnUbiCheck,
	hasModifiedFilesCheck.Name():  hasModifiedFilesCheck,
	// runnableContainerCheck.Name():  runnableContainerCheck,
	// runSystemContainerCheck.Name(): runSystemContainerCheck,
}

func makeCheckList(checkMap map[string]certification.Check) []string {
	checks := make([]string, 0, len(checkMap))

	for key := range checkMap {
		checks = append(checks, key)
	}

	return checks
}

func OperatorPolicy() []string {
	return makeCheckList(operatorPolicy)
}

func ContainerPolicy() []string {
	return makeCheckList(containerPolicy)
}

func ScratchContainerPolicy() []string {
	return makeCheckList(scratchContainerPolicy)
}

func RootExceptionContainerPolicy() []string {
	return makeCheckList(rootExceptionContainerPolicy)
}
