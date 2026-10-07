package replay

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
)

// localRoundTripper serves requests directly from an in-process http.Handler,
// bypassing the network entirely. This mirrors the same technique used by
// internal/pyxis's own test suite (see pyxis_suite_test.go) to fake Pyxis
// responses without a real listener or TLS certificate.
type localRoundTripper struct {
	handler http.Handler
}

func (l localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	l.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

// buildFakeArtifactsTar writes a realistic offline artifacts directory
// (cert-image.json, results.json, preflight.log) to disk, exactly as
// `preflight check container --offline` would, and tars it up.
func buildFakeArtifactsTar(t *testing.T, dir string) string {
	t.Helper()

	certImage := map[string]any{
		"docker_image_digest": "sha256:deadbeef",
		"architecture":        "amd64",
		"parsed_data": map[string]any{
			"architecture": "amd64",
			"uncompressed_layer_sizes": []map[string]any{
				{"layer_id": "sha256:02f151e07d44d20fac36bab4b8e0dab4b7d81e6da75a4a3f30ab476e9c0ac2e5", "size_bytes": 1000},
				{"layer_id": "sha256:aa8823b5c71ec9b7dc06c1c2f60c1b2d573ac6b4b74d43ff77b8ad0c00a6f1a1", "size_bytes": 2000},
			},
		},
		"uncompressed_top_layer_id": "sha256:02f151e07d44d20fac36bab4b8e0dab4b7d81e6da75a4a3f30ab476e9c0ac2e5",
	}

	results := map[string]any{
		"image":        "quay.io/example/myapp:latest",
		"passed":       false,
		"test_library": map[string]any{"name": "preflight", "version": "1.0.0"},
		"results": map[string]any{
			"passed": []map[string]any{
				{"name": "HasLicense", "elapsed_time": 12.1, "description": "has a license"},
			},
			"failed": []map[string]any{},
			"errors": []map[string]any{
				{
					"name":         basedOnUbiCheckName,
					"elapsed_time": 5.0,
					"description":  basedOnUbiCheck.Metadata().Description,
					"help":         "could not reach pyxis: no network",
				},
			},
			"warning": []map[string]any{},
		},
	}

	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeJSON(t, filepath.Join(srcDir, check.DefaultCertImageFilename), certImage)
	writeJSON(t, filepath.Join(srcDir, check.DefaultTestResultsFilename), results)
	if err := os.WriteFile(filepath.Join(srcDir, "preflight.log"), []byte("fake preflight log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tarPath := filepath.Join(dir, check.DefaultArtifactsTarFileName)
	tarDir(t, srcDir, tarPath)

	return tarPath
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// tarDir mirrors checkContainerRunE's artifactsTar helper: a flat tar of the
// regular files directly under src (no leading "./").
func tarDir(t *testing.T, src, tarPath string) {
	t.Helper()

	out, err := os.Create(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	tw := tar.NewWriter(out)
	defer tw.Close()

	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			t.Fatal(err)
		}
		header, err := tar.FileInfoHeader(info, entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(filepath.Join(src, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()
	}
}

// fakePyxisGraphQLHandler responds to the CertifiedImagesContainingLayers
// GraphQL query, simulating Pyxis finding (or not finding) a certified UBI
// image containing one of the submitted layers.
func fakePyxisGraphQLHandler(found bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !found {
			_, _ = w.Write([]byte(`{"data":{"find_images":{"error":null,"total":0,"page":0,"data":[]}}}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"data": {
				"find_images": {
					"error": null,
					"total": 1,
					"page": 0,
					"data": [
						{
							"uncompressed_top_layer_id": "sha256:02f151e07d44d20fac36bab4b8e0dab4b7d81e6da75a4a3f30ab476e9c0ac2e5",
							"_id": "deadb33f",
							"freshness_grades": []
						}
					]
				}
			}
		}`))
	}
}

// TestRun_DefaultsExtractDirToTarballDirectory verifies that Run() extracts
// into the same directory the tarball itself lives in when Options.ExtractDir
// is left empty, rather than hiding the extracted/updated files in a
// temporary directory the caller would otherwise have to go looking for.
func TestRun_DefaultsExtractDirToTarballDirectory(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}

	summary, err := Run(context.Background(), Options{
		ArtifactsTarPath: tarPath,
		// ExtractDir intentionally left empty.
		PyxisHost:       "fake-pyxis.example.com",
		PyxisHTTPClient: fakeClient,
		Submit:          false,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if summary.ExtractDir != workDir {
		t.Fatalf("summary.ExtractDir = %q, want %q (the tarball's own directory)", summary.ExtractDir, workDir)
	}
	if _, err := os.Stat(filepath.Join(summary.ExtractDir, check.DefaultTestResultsFilename)); err != nil {
		t.Errorf("expected %s to have been extracted alongside the tarball: %v", check.DefaultTestResultsFilename, err)
	}
}

// TestRun_DefaultExtractDirUpdatesPreExistingFilesInPlace proves that when
// the raw offline artifacts (cert-image.json/results.json/preflight.log) are
// still sitting next to the tarball - exactly as `check container --offline`
// leaves them, since it tars the same directory it wrote them into - Run()
// overwrites/updates those exact files in place rather than writing a second,
// separate copy elsewhere.
func TestRun_DefaultExtractDirUpdatesPreExistingFilesInPlace(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)

	// Simulate the raw offline files already being present alongside the
	// tarball (e.g. the whole artifacts/<platform>/ directory was copied
	// over, not just artifacts.tar), with deliberately stale/sentinel
	// content so we can tell whether Run() actually overwrote them.
	stalePreflightLog := filepath.Join(workDir, "preflight.log")
	if err := os.WriteFile(stalePreflightLog, []byte("STALE: should be overwritten\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	staleResults := filepath.Join(workDir, check.DefaultTestResultsFilename)
	if err := os.WriteFile(staleResults, []byte(`{"image":"stale","passed":false}`), 0o644); err != nil {
		t.Fatal(err)
	}

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}

	summary, err := Run(context.Background(), Options{
		ArtifactsTarPath: tarPath,
		// ExtractDir intentionally left empty.
		PyxisHost:       "fake-pyxis.example.com",
		PyxisHTTPClient: fakeClient,
		Submit:          false,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if summary.ExtractDir != workDir {
		t.Fatalf("summary.ExtractDir = %q, want %q", summary.ExtractDir, workDir)
	}

	// results.json at the exact same path must now reflect the revalidated
	// outcome - not the stale sentinel content pre-placed above.
	got, err := ReadResults(workDir)
	if err != nil {
		t.Fatalf("ReadResults() error = %v", err)
	}
	if got.Image == "stale" {
		t.Errorf("results.json still has the stale sentinel content; expected it to be overwritten in place")
	}
	if !got.Passed {
		t.Errorf("results.json passed = false, want true")
	}

	// preflight.log at the exact same path must now contain the extracted
	// offline log plus the revalidation note - not the stale sentinel.
	logContents, err := os.ReadFile(stalePreflightLog)
	if err != nil {
		t.Fatalf("could not read preflight.log: %v", err)
	}
	if strings.Contains(string(logContents), "STALE") {
		t.Errorf("preflight.log still has the stale sentinel content; expected it to be overwritten in place, got %q", logContents)
	}
	if !strings.Contains(string(logContents), "BasedOnUbi was revalidated") {
		t.Errorf("expected a revalidation note appended to preflight.log, got %q", logContents)
	}
}

// TestRun_EndToEnd_BasedOnUbiPasses drives the full replay-submit pipeline -
// extracting a fake offline artifacts.tar, querying a faked Pyxis that finds
// a matching certified UBI layer, and verifying results.json on disk ends up
// updated with BasedOnUbi passing and the overall result passing.
func TestRun_EndToEnd_BasedOnUbiPasses(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)
	extractDir := filepath.Join(workDir, "extracted")

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}

	summary, err := Run(context.Background(), Options{
		ArtifactsTarPath: tarPath,
		ExtractDir:       extractDir,
		PyxisHost:        "fake-pyxis.example.com",
		PyxisHTTPClient:  fakeClient,
		Submit:           false, // dry-run: just prove the revalidation + results.json update works
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !summary.BasedOnUbiPased {
		t.Errorf("summary.BasedOnUbiPased = false, want true")
	}
	if summary.Submitted {
		t.Errorf("summary.Submitted = true, want false (Submit was not requested)")
	}

	got, err := ReadResults(extractDir)
	if err != nil {
		t.Fatalf("ReadResults() error = %v", err)
	}

	if !got.Passed {
		t.Errorf("results.json passed = false, want true")
	}
	if len(got.Results.Errors) != 0 {
		t.Errorf("expected no remaining errors, got %+v", got.Results.Errors)
	}

	var foundBasedOnUbi bool
	for _, entry := range got.Results.Passed {
		if entry.Name == basedOnUbiCheckName {
			foundBasedOnUbi = true
		}
	}
	if !foundBasedOnUbi {
		t.Errorf("expected BasedOnUbi in Passed, got %+v", got.Results.Passed)
	}

	// The pre-existing HasLicense pass, produced entirely offline, must survive untouched.
	var foundHasLicense bool
	for _, entry := range got.Results.Passed {
		if entry.Name == "HasLicense" {
			foundHasLicense = true
		}
	}
	if !foundHasLicense {
		t.Errorf("expected pre-existing HasLicense pass to be preserved, got %+v", got.Results.Passed)
	}

	// preflight.log, extracted as-is from the (fake) disconnected run, still
	// says BasedOnUbi could not reach Pyxis. Run() must append a note
	// documenting the real, revalidated outcome so that stale entry isn't
	// the last/only word on BasedOnUbi in a log that might get submitted.
	logContents, err := os.ReadFile(filepath.Join(extractDir, "preflight.log"))
	if err != nil {
		t.Fatalf("could not read preflight.log: %v", err)
	}
	if !strings.Contains(string(logContents), "fake preflight log") {
		t.Errorf("expected the original offline log content to be preserved, got %q", logContents)
	}
	if !strings.Contains(string(logContents), "BasedOnUbi was revalidated") {
		t.Errorf("expected a revalidation note appended to preflight.log, got %q", logContents)
	}
	if !strings.Contains(string(logContents), "passed=true") {
		t.Errorf("expected the revalidation note to record passed=true, got %q", logContents)
	}
}

// TestRun_EndToEnd_BasedOnUbiFails proves the failure path: when Pyxis finds
// no certified image containing the layers, BasedOnUbi should end up Failed
// and the overall result should stay failed.
func TestRun_EndToEnd_BasedOnUbiFails(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)
	extractDir := filepath.Join(workDir, "extracted")

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(false)}}

	summary, err := Run(context.Background(), Options{
		ArtifactsTarPath: tarPath,
		ExtractDir:       extractDir,
		PyxisHost:        "fake-pyxis.example.com",
		PyxisHTTPClient:  fakeClient,
		Submit:           false,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if summary.BasedOnUbiPased {
		t.Errorf("summary.BasedOnUbiPased = true, want false")
	}

	got, err := ReadResults(extractDir)
	if err != nil {
		t.Fatalf("ReadResults() error = %v", err)
	}

	if got.Passed {
		t.Errorf("results.json passed = true, want false")
	}

	var foundFailed bool
	for _, entry := range got.Results.Failed {
		if entry.Name == basedOnUbiCheckName {
			foundFailed = true
			if entry.Suggestion == "" {
				t.Errorf("expected a Suggestion on the failed BasedOnUbi entry")
			}
		}
	}
	if !foundFailed {
		t.Errorf("expected BasedOnUbi in Failed, got %+v", got.Results.Failed)
	}

	logContents, err := os.ReadFile(filepath.Join(extractDir, "preflight.log"))
	if err != nil {
		t.Fatalf("could not read preflight.log: %v", err)
	}
	if !strings.Contains(string(logContents), "passed=false") {
		t.Errorf("expected the revalidation note to record passed=false, got %q", logContents)
	}
}

// TestAppendRevalidationLogEntry_MissingDir proves a helpful error is
// returned when the extraction directory doesn't exist (e.g. was removed
// out from under Run()), rather than a panic.
func TestAppendRevalidationLogEntry_MissingDir(t *testing.T) {
	err := appendRevalidationLogEntry(filepath.Join(t.TempDir(), "does-not-exist"), true)
	if err == nil {
		t.Fatal("expected an error when the extraction directory does not exist")
	}
}

// TestRun_EndToEnd_WithSubmit drives the full replay-submit pipeline with
// Submit: true, using an injected fake lib.PyxisClient (via Options.PyxisClient)
// so the submission path is exercised without any real Pyxis network calls.
func TestRun_EndToEnd_WithSubmit(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)
	extractDir := filepath.Join(workDir, "extracted")

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}
	fakePC := &fakePyxisClient{
		project: &pyxis.CertProject{ID: "abc123", Container: pyxis.Container{Type: "scratch"}},
		submitResults: &pyxis.CertificationResults{
			CertImage:   &pyxis.CertImage{ID: "img123"},
			TestResults: &pyxis.TestResults{ID: "tr123"},
		},
	}

	summary, err := Run(context.Background(), Options{
		ArtifactsTarPath:         tarPath,
		ExtractDir:               extractDir,
		PyxisHost:                "fake-pyxis.example.com",
		PyxisHTTPClient:          fakeClient,
		CertificationComponentID: "abc123",
		Submit:                   true,
		PyxisClient:              fakePC,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !summary.Submitted {
		t.Errorf("summary.Submitted = false, want true")
	}
}

// TestRun_EndToEnd_SubmitFails proves that a submission failure is
// propagated from Run(), even though results.json was already successfully
// updated on disk (i.e. the revalidation itself isn't lost).
func TestRun_EndToEnd_SubmitFails(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)
	extractDir := filepath.Join(workDir, "extracted")

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}
	fakePC := &fakePyxisClient{getProjectErr: errors.New("simulated submission failure")}

	_, err := Run(context.Background(), Options{
		ArtifactsTarPath:         tarPath,
		ExtractDir:               extractDir,
		PyxisHost:                "fake-pyxis.example.com",
		PyxisHTTPClient:          fakeClient,
		CertificationComponentID: "abc123",
		Submit:                   true,
		PyxisClient:              fakePC,
	})
	if err == nil {
		t.Fatal("expected an error when submission fails, got nil")
	}

	// results.json should still reflect the revalidated BasedOnUbi outcome.
	got, err := ReadResults(extractDir)
	if err != nil {
		t.Fatalf("ReadResults() error = %v", err)
	}
	if !got.Passed {
		t.Errorf("results.json passed = false, want true even though submission failed")
	}
}

// TestRun_UsesDefaultHTTPClientWhenNilProvided proves Run() falls back to
// its own *http.Client when Options.PyxisHTTPClient is left nil, rather than
// panicking on a nil client. PyxisHost points at a closed local port so the
// resulting Pyxis query fails immediately (connection refused) instead of
// reaching the real network or hanging for the client's timeout.
func TestRun_UsesDefaultHTTPClientWhenNilProvided(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)
	extractDir := filepath.Join(workDir, "extracted")

	_, err := Run(context.Background(), Options{
		ArtifactsTarPath: tarPath,
		ExtractDir:       extractDir,
		PyxisHost:        "127.0.0.1:1", // nothing listens on port 1; connection refused immediately
	})
	if err == nil {
		t.Fatal("expected an error querying a closed local port")
	}
}

// TestRun_EndToEnd_InvalidCertImage proves a helpful error is returned when
// cert-image.json inside the tarball is malformed, rather than a panic.
func TestRun_EndToEnd_InvalidCertImage(t *testing.T) {
	workDir := t.TempDir()
	srcDir := filepath.Join(workDir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, check.DefaultCertImageFilename), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	tarPath := filepath.Join(workDir, check.DefaultArtifactsTarFileName)
	tarDir(t, srcDir, tarPath)

	_, err := Run(context.Background(), Options{
		ArtifactsTarPath: tarPath,
		ExtractDir:       filepath.Join(workDir, "extracted"),
		PyxisHost:        "fake-pyxis.example.com",
	})
	if err == nil {
		t.Fatal("expected an error for a malformed cert-image.json")
	}
}

// TestRun_EndToEnd_CertImageMissingParsedData proves a helpful error is
// returned when cert-image.json is valid JSON but lacks the parsed_data
// this workflow depends on (e.g. it was written by an older preflight).
func TestRun_EndToEnd_CertImageMissingParsedData(t *testing.T) {
	workDir := t.TempDir()
	srcDir := filepath.Join(workDir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(srcDir, check.DefaultCertImageFilename), map[string]any{"docker_image_digest": "sha256:deadbeef"})

	tarPath := filepath.Join(workDir, check.DefaultArtifactsTarFileName)
	tarDir(t, srcDir, tarPath)

	_, err := Run(context.Background(), Options{
		ArtifactsTarPath: tarPath,
		ExtractDir:       filepath.Join(workDir, "extracted"),
		PyxisHost:        "fake-pyxis.example.com",
	})
	if err == nil {
		t.Fatal("expected an error when cert-image.json has no parsed_data")
	}
}

// TestRun_EndToEnd_MissingResultsJSON proves a helpful error is returned
// when the tarball has a usable cert-image.json (so BasedOnUbi could be
// revalidated) but is missing results.json to write that outcome into.
func TestRun_EndToEnd_MissingResultsJSON(t *testing.T) {
	workDir := t.TempDir()
	srcDir := filepath.Join(workDir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(srcDir, check.DefaultCertImageFilename), map[string]any{
		"parsed_data": map[string]any{
			"uncompressed_layer_sizes": []map[string]any{
				{"layer_id": "sha256:02f151e07d44d20fac36bab4b8e0dab4b7d81e6da75a4a3f30ab476e9c0ac2e5"},
			},
		},
	})
	// results.json intentionally omitted.

	tarPath := filepath.Join(workDir, check.DefaultArtifactsTarFileName)
	tarDir(t, srcDir, tarPath)

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}

	_, err := Run(context.Background(), Options{
		ArtifactsTarPath: tarPath,
		ExtractDir:       filepath.Join(workDir, "extracted"),
		PyxisHost:        "fake-pyxis.example.com",
		PyxisHTTPClient:  fakeClient,
	})
	if err == nil {
		t.Fatal("expected an error when results.json is missing")
	}
}

// TestRun_MissingArtifactsTar proves a helpful error is returned when the
// tarball path does not exist, rather than a panic or opaque failure.
func TestRun_MissingArtifactsTar(t *testing.T) {
	workDir := t.TempDir()

	_, err := Run(context.Background(), Options{
		ArtifactsTarPath: filepath.Join(workDir, "does-not-exist.tar"),
		ExtractDir:       filepath.Join(workDir, "extracted"),
		PyxisHost:        "fake-pyxis.example.com",
	})
	if err == nil {
		t.Fatal("expected an error for a missing artifacts tar, got nil")
	}
}
