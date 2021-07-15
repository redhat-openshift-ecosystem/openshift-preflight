package formatters

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
)

// getResponse will extract the runtime's results and format it to fit the
// UserResponse definition in a way that can then be formatted.
func getResponse(r runtime.Results) UserResponse {
	passedChecks := make([]checkExecutionInfo, len(r.Passed))
	failedChecks := make([]checkExecutionInfo, len(r.Failed))
	erroredChecks := make([]checkExecutionInfo, len(r.Errors))

	if len(r.Passed) > 0 {
		for i, check := range r.Passed {
			passedChecks[i] = checkExecutionInfo{
				Name:        check.Name(),
				ElapsedTime: check.ElapsedTime.String(),
				Description: check.Metadata().Description,
			}
		}
	}

	if len(r.Failed) > 0 {
		for i, check := range r.Failed {
			failedChecks[i] = checkExecutionInfo{
				Name:             check.Name(),
				ElapsedTime:      check.ElapsedTime.String(),
				Description:      check.Metadata().Description,
				Help:             check.Help().Message,
				Suggestion:       check.Help().Suggestion,
				KnowledgeBaseURL: check.Metadata().KnowledgeBaseURL,
				CheckURL:         check.Metadata().CheckURL,
			}
		}
	}

	if len(r.Errors) > 0 {
		for i, check := range r.Errors {
			erroredChecks[i] = checkExecutionInfo{
				Name:         check.Name(),
				ElapsedTime:  check.ElapsedTime.String(),
				Description:  check.Metadata().Description,
				ErrorMessage: "Check " + check.Name() + " encountered an error. Please review the logs for more information",
			}
		}
	}

	response := UserResponse{
		Image:             r.TestedImage,
		ValidationVersion: version.Version,
		Results: resultsText{
			Passed: passedChecks,
			Failed: failedChecks,
			Errors: erroredChecks,
		},
	}

	return response
}

type UserResponse struct {
	Image             string                 `json:"image" xml:"image"`
	ValidationVersion version.VersionContext `json:"validation_lib_version" xml:"validationLibVersion"`
	Results           resultsText            `json:"results" xml:"results"`
}

type resultsText struct {
	Passed []checkExecutionInfo `json:"passed" xml:"passed"`
	Failed []checkExecutionInfo `json:"failed" xml:"failed"`
	Errors []checkExecutionInfo `json:"errors" xml:"errors"`
}

type checkExecutionInfo struct {
	Name             string `json:"name,omitempty" xml:"name,omitempty"`
	ElapsedTime      string `json:"elapsed_time,omitempty" xml:"elapsed_time,omitempty"`
	Description      string `json:"description,omitempty" xml:"description,omitempty"`
	Help             string `json:"help,omitempty" xml:"help,omitempty"`
	Suggestion       string `json:"suggestion,omitempty" xml:"suggestion,omitempty"`
	KnowledgeBaseURL string `json:"knowledgebase_url,omitempty" xml:"knowledgebase_url,omitempty"`
	CheckURL         string `json:"check_url,omitempty" xml:"check_url,omitempty"`
	ErrorMessage     string `json:"error_message,omitempty" xml:"error_message,omitempty"`
}
