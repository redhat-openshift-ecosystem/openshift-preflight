package runtime

import (
	"fmt"

	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/certification/errors"
	"github.com/komish/preflight/certification/internal/policy/podmanexec"
)

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

type PolicyRunner interface {
	ExecutePolicies()
	// StorePolicies(...[]certification.Policy)
	GetResults() Results
}

func NewForConfig(config Config) (*policyRunner, error) {
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

	runner := &policyRunner{
		Image:    config.Image,
		Policies: policies,
	}

	return runner, nil
}

// ExecutePolicies runs all policies stored in the policy runner.
func (pr *policyRunner) ExecutePolicies() {
	pr.Results.TestedImage = pr.Image
	for _, policy := range pr.Policies {
		passed, err := policy.Validate(pr.Image)

		if err != nil {
			pr.Results.Errors = append(pr.Results.Errors, policy)
			continue
		}

		if !passed {
			pr.Results.Failed = append(pr.Results.Failed, policy)
			continue
		}

		pr.Results.Passed = append(pr.Results.Passed, policy)
	}
}

// StorePolicy stores a given policy that needs to be executed in the policy runner.
func (pr *policyRunner) StorePolicies(policies ...certification.Policy) {
	// pr.Policies = append(pr.Policies, policies...)
}

// GetResults will return the results of policy execution
func (pr *policyRunner) GetResults() Results {
	return pr.Results
}
