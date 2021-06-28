package engine

import (
	"fmt"

	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/certification/errors"
	podmanexec "github.com/komish/preflight/certification/internal/shell"
	"github.com/komish/preflight/certification/runtime"
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

	engine := &podmanexec.CheckEngine{
		Image:  config.Image,
		Checks: checks,
	}

	return engine, nil
}

// Register all checks
var runAsNonRootCheck certification.Check = &podmanexec.RunAsNonRootCheck{}
var underLayerMaxCheck certification.Check = &podmanexec.UnderLayerMaxCheck{}
var hasRequiredLabelCheck certification.Check = &podmanexec.HasRequiredLabelsCheck{}
var basedOnUbiCheck certification.Check = &podmanexec.BaseOnUBICheck{}
var hasLicenseCheck certification.Check = &podmanexec.HasLicenseCheck{}
var hasMinimalVulnerabilitiesCheck certification.Check = &podmanexec.HasMinimalVulnerabilitiesCheck{}
var hasUniqueTagCheck certification.Check = &podmanexec.HasUniqueTagCheck{}
var hasNoProhibitedCheck certification.Check = &podmanexec.HasNoProhibitedPackagesCheck{}
var validateOperatorBundle certification.Check = &podmanexec.ValidateOperatorBundlePolicy{}

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

func AllChecks() []string {
	all := make([]string, len(nameToChecksMap))
	i := 0

	for k := range nameToChecksMap {
		all[i] = k
		i++
	}
	return all
}
