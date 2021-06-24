package engine

import (
	"fmt"

	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/certification/errors"
	podmanexec "github.com/komish/preflight/certification/internal/shell"
	"github.com/komish/preflight/certification/runtime"
	"github.com/sirupsen/logrus"
)

type PolicyEngine interface {
	ContainerFileManager
	PolicyRunner
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

type PolicyRunner interface {
	ExecutePolicies(logger *logrus.Logger)
	// StorePolicies(...[]certification.Policy)
	Results() runtime.Results
}

func NewForConfig(config runtime.Config) (PolicyRunner, error) {
	if len(config.EnabledPolicies) == 0 {
		// refuse to run if the user has not specified any policies
		return nil, errors.ErrNoPoliciesEnabled
	}

	policies := make([]certification.Policy, len(config.EnabledPolicies))
	for i, policyString := range config.EnabledPolicies {
		policy, exists := nameToPoliciesMap[policyString]
		if !exists {
			err := fmt.Errorf("%w: %s",
				errors.ErrRequestedPolicyNotFound,
				policyString)
			return nil, err
		}

		policies[i] = policy
	}

	engine := &podmanexec.PolicyEngine{
		Image:    config.Image,
		Policies: policies,
	}

	return engine, nil
}

// Register all policies
var runAsNonRootPolicy certification.Policy = &podmanexec.RunAsNonRootPolicy{}
var underLayerMaxPolicy certification.Policy = &podmanexec.UnderLayerMaxPolicy{}
var hasRequiredLabelPolicy certification.Policy = &podmanexec.HasRequiredLabelPolicy{}
var basedOnUbiPolicy certification.Policy = &podmanexec.BasedOnUbiPolicy{}
var hasLicensePolicy certification.Policy = &podmanexec.HasLicensePolicy{}
var hasMinimalVulnerabilitiesPolicy certification.Policy = &podmanexec.HasMinimalVulnerabilitiesPolicy{}
var hasUniqueTag certification.Policy = &podmanexec.HasUniqueTagPolicy{}
var hasNoProhibitedPackages certification.Policy = &podmanexec.HasNoProhibitedPackagesPolicy{}

var nameToPoliciesMap = map[string]certification.Policy{
	runAsNonRootPolicy.Name():              runAsNonRootPolicy,
	underLayerMaxPolicy.Name():             underLayerMaxPolicy,
	hasRequiredLabelPolicy.Name():          hasRequiredLabelPolicy,
	basedOnUbiPolicy.Name():                basedOnUbiPolicy,
	hasLicensePolicy.Name():                hasLicensePolicy,
	hasMinimalVulnerabilitiesPolicy.Name(): hasMinimalVulnerabilitiesPolicy,
	hasUniqueTag.Name():                    hasUniqueTag,
	hasNoProhibitedPackages.Name():         hasNoProhibitedPackages,
}

func AllPolicies() []string {
	all := make([]string, len(nameToPoliciesMap))
	i := 0

	for k := range nameToPoliciesMap {
		all[i] = k
		i++
	}
	return all
}
