package lib

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"
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

// GetContainerPolicyExceptions will query Pyxis to determine if
// a given project has a certification excemptions, such as root or scratch.
// This will then return the corresponding policy.
//
// If no policy exception flags are found on the project, the standard
// container policy is returned.
func GetContainerPolicyExceptions(ctx context.Context, pc PyxisClient) (policy.Policy, error) {
	certProject, err := pc.GetProject(ctx)
	if err != nil {
		return "", fmt.Errorf("could not retrieve project: %w", err)
	}
	// log.Debugf("Certification project name is: %s", certProject.Name)
	if certProject.Container.Type == "scratch" {
		return policy.PolicyScratch, nil
	}

	// if a partner sets `Host Level Access` in connect to `Privileged`, enable RootExceptionContainerPolicy checks
	if certProject.Container.Privileged {
		return policy.PolicyRoot, nil
	}
	return policy.PolicyContainer, nil
}
