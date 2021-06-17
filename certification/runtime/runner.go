package runtime

import (
	"fmt"

	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/certification/errors"
	"github.com/komish/preflight/certification/internal/policy"
	"github.com/komish/preflight/version"
)

// TODO: Decide what of this file actually needs exporting

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
		policyFunc, exists := certification.PolicyMap[policyString]
		if !exists {
			err := fmt.Errorf("%w: %s",
				errors.ErrRequestedPolicyNotFound,
				policyString)
			return nil, err
		}

		policies[i] = policyFunc()
	}

	runner := &policyRunner{
		Image:    config.Image,
		Policies: policies,
	}

	return runner, nil
}

type policyRunner struct {
	Image    string
	Policies []certification.Policy
	Results  Results
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

type Results struct {
	TestedImage string
	Passed      []certification.Policy
	Failed      []certification.Policy
	Errors      []certification.Policy
}

type UserResponse struct {
	Image             string                 `json:"image" xml:"image"`
	ValidationVersion version.VersionContext `json:"validation_lib_version" xml:"validationLibVersion"`
	Results           UserResponseText       `json:"results" xml:"results"`
}

type UserResponseText struct {
	Passed []policy.Metadata
	Failed []policy.PolicyInfo
	Errors []policy.HelpText
	// TODO: Errors does not actually include any error information
	// and it needs to do so.
}
