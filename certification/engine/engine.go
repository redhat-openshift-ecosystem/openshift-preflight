// Package engine contains the interfaces necessary to implement policy execution.
package engine

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/pyxis"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	internal "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/operatorsdk"
	containerpol "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy/container"
	operatorpol "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy/operator"
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

func New(ctx context.Context,
	image string,
	checks []certification.Check,
	kubeconfig []byte,
	dockerconfig string,
	isBundle,
	isScratch bool,
	insecure bool,
	platform string,
) (CheckEngine, error) {
	return &internal.CraneEngine{
		Kubeconfig:   kubeconfig,
		DockerConfig: dockerconfig,
		Image:        image,
		Checks:       checks,
		IsBundle:     isBundle,
		IsScratch:    isScratch,
		Platform:     platform,
	}, nil
}

// OperatorCheckConfig contains configuration relevant to an individual check's execution.
type OperatorCheckConfig struct {
	ScorecardImage, ScorecardWaitTime, ScorecardNamespace, ScorecardServiceAccount string
	IndexImage, DockerConfig, Channel                                              string
	Kubeconfig                                                                     []byte
}

// InitializeOperatorChecks returns opeartor checks for policy p give cfg.
func InitializeOperatorChecks(ctx context.Context, p policy.Policy, cfg OperatorCheckConfig) ([]certification.Check, error) {
	switch p {
	case policy.PolicyOperator:
		return []certification.Check{
			operatorpol.NewScorecardBasicSpecCheck(operatorsdk.New(cfg.ScorecardImage, exec.Command), cfg.ScorecardNamespace, cfg.ScorecardServiceAccount, cfg.Kubeconfig, cfg.ScorecardWaitTime),
			operatorpol.NewScorecardOlmSuiteCheck(operatorsdk.New(cfg.ScorecardImage, exec.Command), cfg.ScorecardNamespace, cfg.ScorecardServiceAccount, cfg.Kubeconfig, cfg.ScorecardWaitTime),
			operatorpol.NewDeployableByOlmCheck(cfg.IndexImage, cfg.DockerConfig, cfg.Channel),
			operatorpol.NewValidateOperatorBundleCheck(),
			operatorpol.NewCertifiedImagesCheck(pyxis.NewPyxisClient(
				certification.DefaultPyxisHost,
				"",
				"",
				&http.Client{Timeout: 60 * time.Second}),
			),
			operatorpol.NewSecurityContextConstraintsCheck(),
			&operatorpol.RelatedImagesCheck{},
		}, nil
	}

	return nil, fmt.Errorf("provided operator policy %s is unknown", p)
}

// ContainerCheckConfig contains configuration relevant to an individual check's execution.
type ContainerCheckConfig struct {
	DockerConfig, PyxisAPIToken, CertificationProjectID string
}

// InitializeContainerChecks returns the appropriate checks for policy p given cfg.
func InitializeContainerChecks(ctx context.Context, p policy.Policy, cfg ContainerCheckConfig) ([]certification.Check, error) {
	switch p {
	case policy.PolicyContainer:
		return []certification.Check{
			&containerpol.HasLicenseCheck{},
			containerpol.NewHasUniqueTagCheck(cfg.DockerConfig),
			&containerpol.MaxLayersCheck{},
			&containerpol.HasNoProhibitedPackagesCheck{},
			&containerpol.HasRequiredLabelsCheck{},
			&containerpol.RunAsNonRootCheck{},
			&containerpol.HasModifiedFilesCheck{},
			containerpol.NewBasedOnUbiCheck(pyxis.NewPyxisClient(
				certification.DefaultPyxisHost,
				cfg.PyxisAPIToken,
				cfg.CertificationProjectID,
				&http.Client{Timeout: 60 * time.Second})),
		}, nil
	case policy.PolicyRoot:
		return []certification.Check{
			&containerpol.HasLicenseCheck{},
			containerpol.NewHasUniqueTagCheck(cfg.DockerConfig),
			&containerpol.MaxLayersCheck{},
			&containerpol.HasNoProhibitedPackagesCheck{},
			&containerpol.HasRequiredLabelsCheck{},
			&containerpol.HasModifiedFilesCheck{},
			containerpol.NewBasedOnUbiCheck(pyxis.NewPyxisClient(
				certification.DefaultPyxisHost,
				cfg.PyxisAPIToken,
				cfg.CertificationProjectID,
				&http.Client{Timeout: 60 * time.Second})),
		}, nil
	case policy.PolicyScratch:
		return []certification.Check{
			&containerpol.HasLicenseCheck{},
			containerpol.NewHasUniqueTagCheck(cfg.DockerConfig),
			&containerpol.MaxLayersCheck{},
			&containerpol.HasRequiredLabelsCheck{},
			&containerpol.RunAsNonRootCheck{},
		}, nil
	}

	return nil, fmt.Errorf("provided container policy %s is unknown", p)
}

// makeCheckList returns a list of check names.
func makeCheckList(checks []certification.Check) []string {
	checkNames := make([]string, len(checks))

	for i, check := range checks {
		checkNames[i] = check.Name()
	}

	return checkNames
}

// checkNamesFor produces a slice of names for checks in the requested policy.
func checkNamesFor(ctx context.Context, p policy.Policy) []string {
	var c []certification.Check
	switch p {
	case policy.PolicyContainer, policy.PolicyRoot, policy.PolicyScratch:
		c, _ = InitializeContainerChecks(ctx, p, ContainerCheckConfig{})
	case policy.PolicyOperator:
		c, _ = InitializeOperatorChecks(ctx, p, OperatorCheckConfig{})
	default:
		return []string{}
	}

	return makeCheckList(c)
}

// OperatorPolicy returns the names of checks in the operator policy.
func OperatorPolicy(ctx context.Context) []string {
	return checkNamesFor(ctx, policy.PolicyOperator)
}

// ContainerPolicy returns the names of checks in the container policy.
func ContainerPolicy(ctx context.Context) []string {
	return checkNamesFor(ctx, policy.PolicyContainer)
}

// ScratchContainerPolicy returns the names of checks in the
// container policy with scratch exception.
func ScratchContainerPolicy(ctx context.Context) []string {
	return checkNamesFor(ctx, policy.PolicyScratch)
}

// RootExceptionContainerPolicy returns the names of checks in the
// container policy with root exception.
func RootExceptionContainerPolicy(ctx context.Context) []string {
	return checkNamesFor(ctx, policy.PolicyRoot)
}
