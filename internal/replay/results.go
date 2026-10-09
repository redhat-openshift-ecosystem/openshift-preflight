package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy/container"
)

// basedOnUbiCheck is used purely as a source of truth for the BasedOnUbi
// check's Name/Description/Help/Suggestion/URLs, so the entry replay-submit
// writes into results.json matches what a normal online run would have
// produced, without duplicating that text here (and risking drift). It's
// never Validate()'d - LayerHashCheckEngine is intentionally left nil.
var basedOnUbiCheck = container.NewBasedOnUbiCheck(nil)

// basedOnUbiCheckName mirrors the Name() of the BasedOnUbi check
// (internal/policy/container.BasedOnUBICheck) so that we can find/replace its
// entry inside a previously written results.json.
var basedOnUbiCheckName = basedOnUbiCheck.Name()

// CheckInfo mirrors internal/formatters.checkExecutionInfo. It's re-declared
// here (rather than imported) because the upstream type is unexported and
// results.json is the only contract we actually need to depend on.
type CheckInfo struct {
	Name             string  `json:"name,omitempty"`
	ElapsedTime      float64 `json:"elapsed_time"`
	Description      string  `json:"description,omitempty"`
	Help             string  `json:"help,omitempty"`
	Suggestion       string  `json:"suggestion,omitempty"`
	KnowledgeBaseURL string  `json:"knowledgebase_url,omitempty"`
	CheckURL         string  `json:"check_url,omitempty"`
}

// ResultsText mirrors internal/formatters.resultsText.
type ResultsText struct {
	Passed   []CheckInfo `json:"passed"`
	Failed   []CheckInfo `json:"failed"`
	Errors   []CheckInfo `json:"errors"`
	Warnings []CheckInfo `json:"warning,omitempty"`
}

// UserResponse mirrors internal/formatters.UserResponse - the shape of
// results.json that preflight writes to the artifacts directory.
type UserResponse struct {
	Image             string          `json:"image"`
	Passed            bool            `json:"passed"`
	CertificationHash string          `json:"certification_hash,omitempty"`
	LibraryInfo       json.RawMessage `json:"test_library"`
	Results           ResultsText     `json:"results"`
}

// ReadResults reads and unmarshals results.json from dir.
func ReadResults(dir string) (*UserResponse, error) {
	b, err := os.ReadFile(filepath.Join(dir, check.DefaultTestResultsFilename))
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", check.DefaultTestResultsFilename, err)
	}

	var r UserResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("could not unmarshal %s: %w", check.DefaultTestResultsFilename, err)
	}

	return &r, nil
}

// WriteResults marshals r and writes it back to results.json in dir,
// overwriting whatever was there previously. Formatting mirrors what
// preflight itself writes (indented JSON) so the file stays human-readable.
func WriteResults(dir string, r *UserResponse) error {
	b, err := json.MarshalIndent(r, "", "    ")
	if err != nil {
		//coverage:ignore
		return fmt.Errorf("could not marshal %s: %w", check.DefaultTestResultsFilename, err)
	}

	if err := os.WriteFile(filepath.Join(dir, check.DefaultTestResultsFilename), b, 0o644); err != nil {
		return fmt.Errorf("could not write %s: %w", check.DefaultTestResultsFilename, err)
	}

	return nil
}

// removeBasedOnUbi drops any existing BasedOnUbi entry from every result
// bucket (errors/failed/passed/warnings). In the disconnected case we expect
// to find it under Errors (Pyxis was unreachable), but we scrub every bucket
// defensively in case a user re-runs replay-submit against results that were
// already replayed once.
func removeBasedOnUbi(r *UserResponse) {
	r.Results.Errors = removeByName(r.Results.Errors, basedOnUbiCheckName)
	r.Results.Failed = removeByName(r.Results.Failed, basedOnUbiCheckName)
	r.Results.Passed = removeByName(r.Results.Passed, basedOnUbiCheckName)
	r.Results.Warnings = removeByName(r.Results.Warnings, basedOnUbiCheckName)
}

func removeByName(entries []CheckInfo, name string) []CheckInfo {
	out := make([]CheckInfo, 0, len(entries))
	for _, e := range entries {
		if e.Name != name {
			out = append(out, e)
		}
	}
	return out
}

// ApplyBasedOnUbiResult removes any prior BasedOnUbi entry from r and inserts
// a fresh one reflecting the outcome of the online revalidation performed by
// replay-submit, then recalculates the top-level Passed field.
func ApplyBasedOnUbiResult(r *UserResponse, passed bool) {
	removeBasedOnUbi(r)

	metadata := basedOnUbiCheck.Metadata()

	entry := CheckInfo{
		Name:        basedOnUbiCheckName,
		Description: metadata.Description,
	}

	if passed {
		r.Results.Passed = append(r.Results.Passed, entry)
	} else {
		help := basedOnUbiCheck.Help()
		entry.Help = help.Message
		entry.Suggestion = help.Suggestion
		entry.KnowledgeBaseURL = metadata.KnowledgeBaseURL
		entry.CheckURL = metadata.CheckURL
		r.Results.Failed = append(r.Results.Failed, entry)
	}

	r.Passed = len(r.Results.Failed) == 0 && len(r.Results.Errors) == 0
}
