package offline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy/container"
)

// basedOnUbiCheck supplies Name/Description/Help for results.json entries.
// Never Validate()'d; LayerHashCheckEngine is intentionally nil.
var basedOnUbiCheck = container.NewBasedOnUbiCheck(nil)

// basedOnUbiCheckName is basedOnUbiCheck.Name() for test assertions.
var basedOnUbiCheckName = basedOnUbiCheck.Name()

// ReadResults reads and unmarshals results.json from dir.
func ReadResults(dir string) (*formatters.UserResponse, error) {
	b, err := os.ReadFile(filepath.Join(dir, check.DefaultTestResultsFilename))
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", check.DefaultTestResultsFilename, err)
	}

	var r formatters.UserResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("could not unmarshal %s: %w", check.DefaultTestResultsFilename, err)
	}

	return &r, nil
}

// WriteResults writes r to results.json in dir via artifacts.ArtifactWriter.
func WriteResults(dir string, r *formatters.UserResponse) error {
	b, err := json.MarshalIndent(r, "", "    ")
	if err != nil {
		//coverage:ignore
		return fmt.Errorf("could not marshal %s: %w", check.DefaultTestResultsFilename, err)
	}

	aw, err := artifacts.NewFilesystemWriter(artifacts.WithDirectory(dir))
	if err != nil {
		//coverage:ignore
		return fmt.Errorf("could not create artifact writer for %s: %w", dir, err)
	}

	if _, err := aw.WriteFile(check.DefaultTestResultsFilename, bytes.NewReader(b)); err != nil {
		return fmt.Errorf("could not write %s: %w", check.DefaultTestResultsFilename, err)
	}

	return nil
}

// removeCheckByName drops name from every result bucket (errors/failed/
// passed/warnings), covering offline errors and re-run already-replayed cases.
func removeCheckByName(r *formatters.UserResponse, name string) {
	r.Results.Errors = removeByName(r.Results.Errors, name)
	r.Results.Failed = removeByName(r.Results.Failed, name)
	r.Results.Passed = removeByName(r.Results.Passed, name)
	r.Results.Warnings = removeByName(r.Results.Warnings, name)
}

func removeByName(entries []formatters.CheckExecutionInfo, name string) []formatters.CheckExecutionInfo {
	out := make([]formatters.CheckExecutionInfo, 0, len(entries))
	for _, e := range entries {
		if e.Name != name {
			out = append(out, e)
		}
	}
	return out
}

// findCheckByName returns the first entry named name across every result
// bucket, used to preserve fields (e.g. ElapsedTime) when rewriting an entry.
func findCheckByName(r *formatters.UserResponse, name string) (formatters.CheckExecutionInfo, bool) {
	for _, bucket := range [][]formatters.CheckExecutionInfo{
		r.Results.Errors,
		r.Results.Failed,
		r.Results.Passed,
		r.Results.Warnings,
	} {
		for _, e := range bucket {
			if e.Name == name {
				return e, true
			}
		}
	}
	return formatters.CheckExecutionInfo{}, false
}

// RemoveCheckResult drops c from r and recalculates Passed.
func RemoveCheckResult(r *formatters.UserResponse, c check.Check) {
	removeCheckByName(r, c.Name())
	r.Passed = len(r.Results.Failed) == 0 && len(r.Results.Errors) == 0
}

// ApplyCheckResult replaces c's entry with a revalidated outcome, preserves
// ElapsedTime, and recalculates Passed. Generic for future checks.
func ApplyCheckResult(r *formatters.UserResponse, c check.Check, passed bool) {
	prior, _ := findCheckByName(r, c.Name())
	removeCheckByName(r, c.Name())

	metadata := c.Metadata()

	entry := formatters.CheckExecutionInfo{
		Name:        c.Name(),
		Description: metadata.Description,
		ElapsedTime: prior.ElapsedTime,
	}

	if passed {
		r.Results.Passed = append(r.Results.Passed, entry)
	} else {
		help := c.Help()
		entry.Help = help.Message
		entry.Suggestion = help.Suggestion
		entry.KnowledgeBaseURL = metadata.KnowledgeBaseURL
		entry.CheckURL = metadata.CheckURL
		r.Results.Failed = append(r.Results.Failed, entry)
	}

	r.Passed = len(r.Results.Failed) == 0 && len(r.Results.Errors) == 0
}
