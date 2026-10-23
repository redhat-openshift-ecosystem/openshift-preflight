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

// basedOnUbiCheck is used purely as a source of truth for the BasedOnUbi
// check's Name/Description/Help/Suggestion/URLs, so the entry written into
// results.json matches what a normal online run would produce. It's never
// Validate()'d - LayerHashCheckEngine is intentionally left nil.
var basedOnUbiCheck = container.NewBasedOnUbiCheck(nil)

// basedOnUbiCheckName mirrors the Name() of the BasedOnUbi check
// (internal/policy/container.BasedOnUBICheck) so that we can find/replace its
// entry inside a previously written results.json.
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

// WriteResults marshals r and writes it back to results.json in dir,
// overwriting whatever was there previously, using the same
// artifacts.ArtifactWriter the rest of preflight writes artifacts with.
// Formatting mirrors preflight's own indented JSON output.
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

// removeCheckByName drops any existing entry named name from every result
// bucket (errors/failed/passed/warnings). We expect it under Errors (Pyxis
// was unreachable), but scrub every bucket in case a user re-runs
// submit-offline-artifacts against already-revalidated results.
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

// ApplyCheckResult removes any prior entry for c from r, inserts a fresh one
// reflecting the outcome of revalidating c, and recalculates the top-level
// Passed field. Generic over any check.Check for reuse by future checks.
func ApplyCheckResult(r *formatters.UserResponse, c check.Check, passed bool) {
	removeCheckByName(r, c.Name())

	metadata := c.Metadata()

	entry := formatters.CheckExecutionInfo{
		Name:        c.Name(),
		Description: metadata.Description,
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
