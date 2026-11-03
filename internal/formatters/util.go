package formatters

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
)

// getResponse will extract the runtime's results and format it to fit the
// UserResponse definition in a way that can then be formatted.
func getResponse(r certification.Results) UserResponse {
	passedChecks := make([]CheckExecutionInfo, 0, len(r.Passed))
	failedChecks := make([]CheckExecutionInfo, 0, len(r.Failed))
	erroredChecks := make([]CheckExecutionInfo, 0, len(r.Errors))
	warnedChecks := make([]CheckExecutionInfo, 0, len(r.Warned))

	if len(r.Passed) > 0 {
		for _, check := range r.Passed {
			passedChecks = append(passedChecks, CheckExecutionInfo{
				Name:        check.Name(),
				ElapsedTime: float64(check.ElapsedTime.Milliseconds()),
				Description: check.Metadata().Description,
			})
		}
	}

	if len(r.Failed) > 0 {
		for _, check := range r.Failed {
			failedChecks = append(failedChecks, CheckExecutionInfo{
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
			erroredChecks = append(erroredChecks, CheckExecutionInfo{
				Name:        check.Name(),
				ElapsedTime: float64(check.ElapsedTime.Milliseconds()),
				Description: check.Metadata().Description,
				Help:        check.Help().Message,
			})
		}
	}

	if len(r.Warned) > 0 {
		for _, check := range r.Warned {
			warnedChecks = append(warnedChecks, CheckExecutionInfo{
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

	response := UserResponse{
		Image:             r.TestedImage,
		Passed:            r.PassedOverall,
		LibraryInfo:       version.Version,
		CertificationHash: r.CertificationHash,
		Results: ResultsText{
			Passed:   passedChecks,
			Failed:   failedChecks,
			Errors:   erroredChecks,
			Warnings: warnedChecks,
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
	Results           ResultsText            `json:"results" xml:"results"`
}

// ResultsText represents the results of check execution against the asset.
type ResultsText struct {
	Passed   []CheckExecutionInfo `json:"passed" xml:"passed"`
	Failed   []CheckExecutionInfo `json:"failed" xml:"failed"`
	Errors   []CheckExecutionInfo `json:"errors" xml:"errors"`
	Warnings []CheckExecutionInfo `json:"warning,omitempty" xml:"warning,omitempty"`
}

// CheckExecutionInfo contains all possible output fields that a user might see in their result.
// Empty fields will be omitted.
type CheckExecutionInfo struct {
	Name             string  `json:"name,omitempty" xml:"name,omitempty"`
	ElapsedTime      float64 `json:"elapsed_time" xml:"elapsed_time"`
	Description      string  `json:"description,omitempty" xml:"description,omitempty"`
	Help             string  `json:"help,omitempty" xml:"help,omitempty"`
	Suggestion       string  `json:"suggestion,omitempty" xml:"suggestion,omitempty"`
	KnowledgeBaseURL string  `json:"knowledgebase_url,omitempty" xml:"knowledgebase_url,omitempty"`
	CheckURL         string  `json:"check_url,omitempty" xml:"check_url,omitempty"`
}
