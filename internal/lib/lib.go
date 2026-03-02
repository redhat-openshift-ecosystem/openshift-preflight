package lib

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
)

// ResolveSubmitter will build out a ResultSubmitter if the provided pyxisClient, pc, is not nil.
// The pyxisClient is a required component of the submitter. If pc is nil, then a noop submitter
// is returned instead, which does nothing.
func ResolveSubmitter(pc PyxisClient, projectID, dockerconfig, logfile string) ResultSubmitter {
	if pc != nil {
		return &ContainerCertificationSubmitter{
			CertificationProjectID: projectID,
			Pyxis:                  pc,
			DockerConfig:           dockerconfig,
			PreflightLogFile:       logfile,
		}
	}
	return NewNoopSubmitter(true, nil)
}

// GetContainerPolicyExceptions accepts a CertProject to determine if
// a given project has certification exemptions, such as root or scratch.
// This will then return the corresponding policy.
//
// If no policy exception flags are found on the project, the standard
// container policy is returned.
func GetContainerPolicyExceptions(certProject *pyxis.CertProject) policy.Policy {
	// check for nil first so code doesn't panic later
	if certProject == nil {
		return policy.PolicyContainer
	}

	// if the partner has gotten a scratch exception from the business and os_content_type == "Scratch Image"
	// and a partner sets `Host Level Access` in connect to `Privileged`, enable ScratchRootContainerPolicy checks
	if certProject.ScratchProject() && certProject.Container.Privileged {
		return policy.PolicyScratchRoot
	}

	// if the partner has gotten a scratch exception from the business and os_content_type == "Scratch Image",
	// enable ScratchNonRootContainerPolicy checks
	if certProject.ScratchProject() {
		return policy.PolicyScratchNonRoot
	}

	// if a partner sets `Host Level Access` in connect to `Privileged`, enable RootExceptionContainerPolicy checks
	if certProject.Container.Privileged {
		return policy.PolicyRoot
	}
	return policy.PolicyContainer
}
