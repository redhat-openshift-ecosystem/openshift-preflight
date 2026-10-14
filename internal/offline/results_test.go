package offline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/formatters"
)

// TestReadResults_MissingFile proves a helpful error is returned when
// results.json does not exist in dir, rather than a panic.
func TestReadResults_MissingFile(t *testing.T) {
	dir := t.TempDir()

	if _, err := ReadResults(dir); err == nil {
		t.Fatal("expected an error when results.json does not exist")
	}
}

// TestReadResults_InvalidJSON proves malformed JSON is reported clearly.
func TestReadResults_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, check.DefaultTestResultsFilename), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadResults(dir); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

// TestWriteResults_CreatesMissingDir proves WriteResults creates dir if it
// doesn't already exist, since it now writes through the same
// artifacts.ArtifactWriter the rest of preflight uses, which does the same.
func TestWriteResults_CreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist-yet")

	if err := WriteResults(dir, &formatters.UserResponse{}); err != nil {
		t.Fatalf("WriteResults() error = %v, want nil", err)
	}

	if _, err := os.Stat(filepath.Join(dir, check.DefaultTestResultsFilename)); err != nil {
		t.Errorf("expected %s to have been created: %v", check.DefaultTestResultsFilename, err)
	}
}

// TestWriteResults_PathComponentIsAFile proves a helpful error is returned
// when a component of dir already exists as a regular file, rather than a
// panic.
func TestWriteResults_PathComponentIsAFile(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(blocker, "results")

	if err := WriteResults(dir, &formatters.UserResponse{}); err == nil {
		t.Fatal("expected an error when a path component is a regular file")
	}
}

func TestReadWriteResultsRoundTrip(t *testing.T) {
	dir := t.TempDir()

	original := &formatters.UserResponse{
		Image:  "quay.io/example/image:latest",
		Passed: false,
		Results: formatters.ResultsText{
			Passed: []formatters.CheckExecutionInfo{{Name: "HasLicense", ElapsedTime: 12.5}},
			Errors: []formatters.CheckExecutionInfo{{Name: basedOnUbiCheckName, Help: "no catalog access"}},
		},
	}

	if err := WriteResults(dir, original); err != nil {
		t.Fatalf("WriteResults() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "results.json")); err != nil {
		t.Fatalf("expected results.json to exist: %v", err)
	}

	got, err := ReadResults(dir)
	if err != nil {
		t.Fatalf("ReadResults() error = %v", err)
	}

	if got.Image != original.Image {
		t.Errorf("Image = %q, want %q", got.Image, original.Image)
	}
	if len(got.Results.Passed) != 1 || got.Results.Passed[0].Name != "HasLicense" {
		t.Errorf("unexpected Passed entries: %+v", got.Results.Passed)
	}
	if len(got.Results.Errors) != 1 || got.Results.Errors[0].Name != basedOnUbiCheckName {
		t.Errorf("unexpected Errors entries: %+v", got.Results.Errors)
	}
}

func TestApplyCheckResult_MovesFromErrorsToPassed(t *testing.T) {
	r := &formatters.UserResponse{
		Passed: false,
		Results: formatters.ResultsText{
			Errors: []formatters.CheckExecutionInfo{{Name: basedOnUbiCheckName, Help: "no catalog access while offline"}},
			Passed: []formatters.CheckExecutionInfo{{Name: "HasLicense"}},
		},
	}

	ApplyCheckResult(r, basedOnUbiCheck, true)

	if got := len(r.Results.Errors); got != 0 {
		t.Errorf("Errors length = %d, want 0", got)
	}

	if got := len(r.Results.Passed); got != 2 {
		t.Fatalf("Passed length = %d, want 2", got)
	}

	var found bool
	for _, e := range r.Results.Passed {
		if e.Name == basedOnUbiCheckName {
			found = true
			if e.Help != "" {
				t.Errorf("expected no Help text on a passing entry, got %q", e.Help)
			}
		}
	}
	if !found {
		t.Errorf("expected %s to be present in Passed", basedOnUbiCheckName)
	}

	if !r.Passed {
		t.Errorf("Passed (overall) = false, want true")
	}
}

func TestApplyCheckResult_MovesFromErrorsToFailed(t *testing.T) {
	r := &formatters.UserResponse{
		Passed: false,
		Results: formatters.ResultsText{
			Errors: []formatters.CheckExecutionInfo{{Name: basedOnUbiCheckName}},
		},
	}

	ApplyCheckResult(r, basedOnUbiCheck, false)

	if got := len(r.Results.Errors); got != 0 {
		t.Errorf("Errors length = %d, want 0", got)
	}
	if got := len(r.Results.Failed); got != 1 {
		t.Fatalf("Failed length = %d, want 1", got)
	}
	if r.Results.Failed[0].Name != basedOnUbiCheckName {
		t.Errorf("Failed[0].Name = %q, want %q", r.Results.Failed[0].Name, basedOnUbiCheckName)
	}
	if r.Results.Failed[0].Suggestion == "" {
		t.Errorf("expected a Suggestion to be set on a failing entry")
	}
	if r.Passed {
		t.Errorf("Passed (overall) = true, want false")
	}
}

func TestApplyCheckResult_OtherFailuresKeepOverallFailed(t *testing.T) {
	r := &formatters.UserResponse{
		Results: formatters.ResultsText{
			Errors: []formatters.CheckExecutionInfo{{Name: basedOnUbiCheckName}},
			Failed: []formatters.CheckExecutionInfo{{Name: "HasUniqueTag"}},
		},
	}

	// BasedOnUbi now passes online, but there's still an unrelated failure.
	ApplyCheckResult(r, basedOnUbiCheck, true)

	if r.Passed {
		t.Errorf("Passed (overall) = true, want false because HasUniqueTag is still failing")
	}
	if got := len(r.Results.Failed); got != 1 {
		t.Fatalf("Failed length = %d, want 1 (only the pre-existing HasUniqueTag failure)", got)
	}
}

func TestApplyCheckResult_IsIdempotent(t *testing.T) {
	r := &formatters.UserResponse{
		Results: formatters.ResultsText{
			Errors: []formatters.CheckExecutionInfo{{Name: basedOnUbiCheckName}},
		},
	}

	ApplyCheckResult(r, basedOnUbiCheck, true)
	ApplyCheckResult(r, basedOnUbiCheck, true) // simulate re-running submit-offline-artifacts against already-replayed results

	if got := len(r.Results.Passed); got != 1 {
		t.Errorf("Passed length = %d, want 1 (no duplicate BasedOnUbi entries)", got)
	}
}
