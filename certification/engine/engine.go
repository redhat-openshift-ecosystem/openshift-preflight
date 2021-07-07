package engine

import (
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/shell"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/sirupsen/logrus"
)

type CheckEngine interface {
	ContainerFileManager
	CheckRunner
}

// ContainerFileManager describes the functionality necessary to interact
// with a container image tarball on disk.
type ContainerFileManager interface {
	// IsRemote will check user-provided path and determine if that path is
	// local or remote. Here local means that it's a location on the filesystem, and
	// remote means that it's an image in a registry.
	ContainerIsRemote(path string, logger *logrus.Logger) (isRemote bool, remotecheckErr error)
	// ExtractContainerTar will accept a path on the filesystem and extract it.
	ExtractContainerTar(path string, logger *logrus.Logger) (tarballPath string, extractionErr error)
	// GetContainerFromRegistry will accept a container location and write it locally
	// as a tarball as done by `podman save`
	GetContainerFromRegistry(containerLoc string, logger *logrus.Logger) (containerDownloadPath string, containerDownloadErro error)
}

type CheckRunner interface {
	ExecuteChecks(logger *logrus.Logger)
	// StoreChecks(...[]certification.Check)
	Results() runtime.Results
}

func NewForConfig(config runtime.Config) (CheckRunner, error) {
	if len(config.EnabledChecks) == 0 {
		// refuse to run if the user has not specified any checks
		return nil, errors.ErrNoChecksEnabled
	}

	checks := make([]certification.Check, len(config.EnabledChecks))
	for i, checkString := range config.EnabledChecks {
		check, exists := nameToChecksMap[checkString]
		if !exists {
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

// Register all checks
var runAsNonRootCheck certification.Check = &shell.RunAsNonRootCheck{}
var underLayerMaxCheck certification.Check = &shell.UnderLayerMaxCheck{}
var hasRequiredLabelCheck certification.Check = &shell.HasRequiredLabelsCheck{}
var basedOnUbiCheck certification.Check = &shell.BaseOnUBICheck{}
var hasLicenseCheck certification.Check = &shell.HasLicenseCheck{}
var hasMinimalVulnerabilitiesCheck certification.Check = &shell.HasMinimalVulnerabilitiesCheck{}
var hasUniqueTagCheck certification.Check = &shell.HasUniqueTagCheck{}
var hasNoProhibitedCheck certification.Check = &shell.HasNoProhibitedPackagesCheck{}
var validateOperatorBundle certification.Check = &shell.ValidateOperatorBundlePolicy{}

var nameToChecksMap = map[string]certification.Check{
	// NOTE(komish): these checks do not all apply to bundles, which is the current
	// scope. Eventually, I expect we'll split out container checks to their
	// on map and pass it to the CheckEngine when the right cobra command is invoked.
	runAsNonRootCheck.Name():              runAsNonRootCheck,
	underLayerMaxCheck.Name():             underLayerMaxCheck,
	hasRequiredLabelCheck.Name():          hasRequiredLabelCheck,
	basedOnUbiCheck.Name():                basedOnUbiCheck,
	hasLicenseCheck.Name():                hasLicenseCheck,
	hasMinimalVulnerabilitiesCheck.Name(): hasMinimalVulnerabilitiesCheck,
	hasUniqueTagCheck.Name():              hasUniqueTagCheck,
	hasNoProhibitedCheck.Name():           hasNoProhibitedCheck,
	validateOperatorBundle.Name():         validateOperatorBundle,
}

var containerPolicyChecks = map[string]certification.Check{
	runAsNonRootCheck.Name():              runAsNonRootCheck,
	underLayerMaxCheck.Name():             underLayerMaxCheck,
	hasRequiredLabelCheck.Name():          hasRequiredLabelCheck,
	basedOnUbiCheck.Name():                basedOnUbiCheck,
	hasLicenseCheck.Name():                hasLicenseCheck,
	hasMinimalVulnerabilitiesCheck.Name(): hasMinimalVulnerabilitiesCheck,
	hasUniqueTagCheck.Name():              hasUniqueTagCheck,
	hasNoProhibitedCheck.Name():           hasNoProhibitedCheck,
}

var operatorPolicyChecks = map[string]certification.Check{
	validateOperatorBundle.Name(): validateOperatorBundle,
}

func AllChecks() []string {
	all := make([]string, len(nameToChecksMap))
	i := 0

	for k := range nameToChecksMap {
		all[i] = k
		i++
	}
	return all
}

func OperatorPolicy() []string {
	checks := make([]string, len(operatorPolicyChecks))
	i := 0

	for k := range operatorPolicyChecks {
		checks[i] = k
		i++
	}
	return checks
}

func ContainerPolicy() []string {
	checks := make([]string, len(containerPolicyChecks))
	i := 0

	for k := range containerPolicyChecks {
		checks[i] = k
		i++
	}
	return checks
}
