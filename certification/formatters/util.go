package formatters

import (
	"github.com/komish/preflight/certification/internal/policy"
	"github.com/komish/preflight/certification/runtime"
	"github.com/komish/preflight/version"
)

// getResponse will extract the runtime's results and format it to fit the
// UserResponse definition in a way that can then be formatted.
func getResponse(r runtime.Results) runtime.UserResponse {
	passedPolicies := make([]policy.Metadata, len(r.Passed))
	failedPolicies := make([]policy.PolicyInfo, len(r.Failed))
	erroredPolicies := make([]policy.HelpText, len(r.Errors))

	if len(r.Passed) > 0 {
		for i, policyData := range r.Passed {
			passedPolicies[i] = policyData.Meta()
		}
	}

	if len(r.Failed) > 0 {
		for i, policyData := range r.Failed {
			failedPolicies[i] = policy.PolicyInfo{
				Metadata: policyData.Meta(),
				HelpText: policyData.Help(),
			}
		}
	}

	if len(r.Errors) > 0 {
		for i, policyData := range r.Errors {
			erroredPolicies[i] = policyData.Help()
		}
	}

	response := runtime.UserResponse{
		Image:             r.TestedImage,
		ValidationVersion: version.Version,
		Results: runtime.UserResponseText{
			Passed: passedPolicies,
			Failed: failedPolicies,
			Errors: erroredPolicies,
		},
	}

	return response
}
