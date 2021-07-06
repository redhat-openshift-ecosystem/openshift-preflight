package formatters

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
)

// getResponse will extract the runtime's results and format it to fit the
// UserResponse definition in a way that can then be formatted.
func getResponse(r runtime.Results) runtime.UserResponse {
	passedChecks := make([]certification.Metadata, len(r.Passed))
	failedChecks := make([]certification.CheckInfo, len(r.Failed))
	erroredChecks := make([]certification.HelpText, len(r.Errors))

	if len(r.Passed) > 0 {
		for i, check := range r.Passed {
			passedChecks[i] = check.Metadata()
		}
	}

	if len(r.Failed) > 0 {
		for i, check := range r.Failed {
			failedChecks[i] = certification.CheckInfo{
				Metadata: check.Metadata(),
				HelpText: check.Help(),
			}
		}
	}

	if len(r.Errors) > 0 {
		for i, check := range r.Errors {
			erroredChecks[i] = check.Help()
		}
	}

	response := runtime.UserResponse{
		Image:             r.TestedImage,
		ValidationVersion: version.Version,
		Results: runtime.UserResponseText{
			Passed: passedChecks,
			Failed: failedChecks,
			Errors: erroredChecks,
		},
	}

	return response
}
