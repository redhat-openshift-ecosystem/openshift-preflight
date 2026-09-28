package replay

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadWriteResultsRoundTrip(t *testing.T) {
	dir := t.TempDir()

	original := &UserResponse{
		Image:  "quay.io/example/image:latest",
		Passed: false,
		Results: ResultsText{
			Passed: []CheckInfo{{Name: "HasLicense", ElapsedTime: 12.5}},
			Errors: []CheckInfo{{Name: basedOnUbiCheckName, Help: "no catalog access"}},
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

func TestApplyBasedOnUbiResult_MovesFromErrorsToPassed(t *testing.T) {
	r := &UserResponse{
		Passed: false,
		Results: ResultsText{
			Errors: []CheckInfo{{Name: basedOnUbiCheckName, Help: "no catalog access while offline"}},
			Passed: []CheckInfo{{Name: "HasLicense"}},
		},
	}

	ApplyBasedOnUbiResult(r, true)

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

func TestApplyBasedOnUbiResult_MovesFromErrorsToFailed(t *testing.T) {
	r := &UserResponse{
		Passed: false,
		Results: ResultsText{
			Errors: []CheckInfo{{Name: basedOnUbiCheckName}},
		},
	}

	ApplyBasedOnUbiResult(r, false)

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

func TestApplyBasedOnUbiResult_OtherFailuresKeepOverallFailed(t *testing.T) {
	r := &UserResponse{
		Results: ResultsText{
			Errors: []CheckInfo{{Name: basedOnUbiCheckName}},
			Failed: []CheckInfo{{Name: "HasUniqueTag"}},
		},
	}

	// BasedOnUbi now passes online, but there's still an unrelated failure.
	ApplyBasedOnUbiResult(r, true)

	if r.Passed {
		t.Errorf("Passed (overall) = true, want false because HasUniqueTag is still failing")
	}
	if got := len(r.Results.Failed); got != 1 {
		t.Fatalf("Failed length = %d, want 1 (only the pre-existing HasUniqueTag failure)", got)
	}
}

func TestApplyBasedOnUbiResult_IsIdempotent(t *testing.T) {
	r := &UserResponse{
		Results: ResultsText{
			Errors: []CheckInfo{{Name: basedOnUbiCheckName}},
		},
	}

	ApplyBasedOnUbiResult(r, true)
	ApplyBasedOnUbiResult(r, true) // simulate re-running replay-submit against already-replayed results

	if got := len(r.Results.Passed); got != 1 {
		t.Errorf("Passed length = %d, want 1 (no duplicate BasedOnUbi entries)", got)
	}
}
