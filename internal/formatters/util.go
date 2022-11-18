package formatters

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
)

// getResponse will extract the runtime's results and format it to fit the
// UserResponse definition in a way that can then be formatted.
func getResponse(r certification.Results) UserResponse {
	passedChecks := make([]checkExecutionInfo, 0, len(r.Passed))
	failedChecks := make([]checkExecutionInfo, 0, len(r.Failed))
	erroredChecks := make([]checkExecutionInfo, 0, len(r.Errors))

	if len(r.Passed) > 0 {
		for _, check := range r.Passed {
			passedChecks = append(passedChecks, checkExecutionInfo{
				Name:        check.Name(),
				ElapsedTime: float64(check.ElapsedTime.Milliseconds()),
				Description: check.Metadata().Description,
			})
		}
	}

	if len(r.Failed) > 0 {
		for _, check := range r.Failed {
			failedChecks = append(failedChecks, checkExecutionInfo{
				Name:             check.Name(),
				ElapsedTime:      float64(check.ElapsedTime.Milliseconds()),
				Description:      check.Metadata().Description,
				Help:             check.Help().Message,
				Suggestion:       check.Help().Suggestion,
				KnowledgeBaseURL: check.Metadata().KnowledgeBaseURL,
				CheckURL:         check.Metadata().CheckURL,
			})
		}
	}

	if len(r.Errors) > 0 {
		for _, check := range r.Errors {
			erroredChecks = append(erroredChecks, checkExecutionInfo{
				Name:        check.Name(),
				ElapsedTime: float64(check.ElapsedTime.Milliseconds()),
				Description: check.Metadata().Description,
				Help:        check.Help().Message,
			})
		}
	}

	response := UserResponse{
		Image:             r.TestedImage,
		Passed:            r.PassedOverall,
		LibraryInfo:       version.Version,
		CertificationHash: r.CertificationHash,
		Results: resultsText{
			Passed: passedChecks,
			Failed: failedChecks,
			Errors: erroredChecks,
		},
	}

	return response
}

// UserResponse is the standard user-facing response.
type UserResponse struct {
	Image             string                 `json:"image" xml:"image"`
	Passed            bool                   `json:"passed" xml:"passed"`
	CertificationHash string                 `json:"certification_hash,omitempty" xml:"certification_hash,omitempty"`
	LibraryInfo       version.VersionContext `json:"test_library" xml:"test_library"`
	Results           resultsText            `json:"results" xml:"results"`
}

// resultsText represents the results of check execution against the asset.
type resultsText struct {
	Passed []checkExecutionInfo `json:"passed" xml:"passed"`
	Failed []checkExecutionInfo `json:"failed" xml:"failed"`
	Errors []checkExecutionInfo `json:"errors" xml:"errors"`
}

// checkExecutionInfo contains all possible output fields that a user might see in their result.
// Empty fields will be omitted.
type checkExecutionInfo struct {
	Name             string  `json:"name,omitempty" xml:"name,omitempty"`
	ElapsedTime      float64 `json:"elapsed_time" xml:"elapsed_time"`
	Description      string  `json:"description,omitempty" xml:"description,omitempty"`
	Help             string  `json:"help,omitempty" xml:"help,omitempty"`
	Suggestion       string  `json:"suggestion,omitempty" xml:"suggestion,omitempty"`
	KnowledgeBaseURL string  `json:"knowledgebase_url,omitempty" xml:"knowledgebase_url,omitempty"`
	CheckURL         string  `json:"check_url,omitempty" xml:"check_url,omitempty"`
}
