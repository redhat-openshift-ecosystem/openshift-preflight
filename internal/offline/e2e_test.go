package offline

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
)

// localRoundTripper serves requests directly from an in-process http.Handler,
// bypassing the network entirely, mirroring the technique used by
// internal/pyxis's own test suite (see pyxis_suite_test.go).
type localRoundTripper struct {
	handler http.Handler
}

func (l localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	l.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

// fakeRawConfig is a minimal OCI image config blob - the shape stored in
// cert-image.json's raw_config field - with just enough (rootfs.diff_ids)
// for diffIDsFromCertImage to parse it back.
const fakeRawConfig = `{
	"architecture": "amd64",
	"os": "linux",
	"rootfs": {
		"type": "layers",
		"diff_ids": [
			"sha256:02f151e07d44d20fac36bab4b8e0dab4b7d81e6da75a4a3f30ab476e9c0ac2e5",
			"sha256:aa8823b5c71ec9b7dc06c1c2f60c1b2d573ac6b4b74d43ff77b8ad0c00a6f1a1"
		]
	},
	"config": {}
}`

// buildFakeArtifactsTar writes a realistic offline artifacts directory
// (cert-image.json, results.json, preflight.log) to disk, exactly as
// `preflight check container --offline` would, and tars it up.
func buildFakeArtifactsTar(t *testing.T, dir string) string {
	t.Helper()

	certImage := map[string]any{
		"docker_image_digest": "sha256:deadbeef",
		"architecture":        "amd64",
		"raw_config":          fakeRawConfig,
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

// TestRun_AlwaysExtractsToAPrivateTempDir proves Run() never extracts into
// the tarball's own directory, so a rerun can't clobber a pre-existing
// results.json from a previously-successful revalidation.
func TestRun_AlwaysExtractsToAPrivateTempDir(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)

	// Simulate a previously-validated results.json already sitting next to
	// the tarball, with sentinel content that must survive this run
	// untouched.
	preExistingResults := filepath.Join(workDir, check.DefaultTestResultsFilename)
	if err := os.WriteFile(preExistingResults, []byte(`{"image":"already-validated","passed":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	logFile := filepath.Join(workDir, "this-run.log")
	if err := os.WriteFile(logFile, []byte("this invocation's own log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}

	// Calls the unexported run() directly (rather than Run()) so the
	// extraction directory can be inspected below before it's cleaned up -
	// Run() always removes it before returning.
	extractDir, _, err := run(context.Background(), Options{
		ArtifactsTarPath:         tarPath,
		LogFile:                  logFile,
		PyxisHost:                "fake-pyxis.example.com",
		PyxisHTTPClient:          fakeClient,
		CertificationComponentID: "abc123",
		PyxisClient:              newSuccessfulFakePyxisClient(),
	})
	defer os.RemoveAll(extractDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if extractDir == workDir || extractDir == filepath.Dir(tarPath) {
		t.Fatalf("extractDir = %q, must not be the tarball's own directory %q", extractDir, workDir)
	}
	if _, err := os.Stat(filepath.Join(extractDir, check.DefaultTestResultsFilename)); err != nil {
		t.Errorf("expected %s to have been extracted into the private temp dir: %v", check.DefaultTestResultsFilename, err)
	}

	// The pre-existing results.json next to the tarball must be untouched.
	got, err := os.ReadFile(preExistingResults)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "already-validated") {
		t.Errorf("pre-existing results.json next to the tarball was modified, got %q", got)
	}
}

// TestRun_EndToEnd_BasedOnUbiPasses drives the full pipeline: extracting a
// fake offline artifacts.tar, querying a faked Pyxis that finds a matching
// certified UBI layer, and verifying results.json ends up passing.
func TestRun_EndToEnd_BasedOnUbiPasses(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)

	logFile := filepath.Join(workDir, "this-run.log")
	if err := os.WriteFile(logFile, []byte("this invocation's own log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}

	// Calls the unexported run() directly (rather than Run()) so the
	// extracted/updated artifacts can be inspected below before they're
	// cleaned up - Run() always removes extractDir before returning.
	extractDir, summary, err := run(context.Background(), Options{
		ArtifactsTarPath:         tarPath,
		LogFile:                  logFile,
		PyxisHost:                "fake-pyxis.example.com",
		PyxisHTTPClient:          fakeClient,
		CertificationComponentID: "abc123",
		PyxisClient:              newSuccessfulFakePyxisClient(),
	})
	defer os.RemoveAll(extractDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !summary.BasedOnUbiPassed {
		t.Errorf("summary.BasedOnUbiPassed = false, want true")
	}
	if !summary.Submitted {
		t.Errorf("summary.Submitted = false, want true")
	}

	got, err := ReadResults(summary.ExtractDir)
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

	// preflight.log, extracted as-is, is left completely untouched by run().
	logContents, err := os.ReadFile(filepath.Join(summary.ExtractDir, "preflight.log"))
	if err != nil {
		t.Fatalf("could not read preflight.log: %v", err)
	}
	if string(logContents) != "fake preflight log\n" {
		t.Errorf("expected the extracted offline log to be left untouched, got %q", logContents)
	}
}

// TestRun_EndToEnd_BasedOnUbiFails proves the failure path: when Pyxis finds
// no certified image containing the layers, BasedOnUbi should end up Failed
// and the overall result should stay failed.
func TestRun_EndToEnd_BasedOnUbiFails(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)

	logFile := filepath.Join(workDir, "this-run.log")
	if err := os.WriteFile(logFile, []byte("this invocation's own log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(false)}}

	// Calls the unexported run() directly (rather than Run()) so
	// results.json can be inspected below before it's cleaned up - Run()
	// always removes extractDir before returning.
	extractDir, summary, err := run(context.Background(), Options{
		ArtifactsTarPath:         tarPath,
		LogFile:                  logFile,
		PyxisHost:                "fake-pyxis.example.com",
		PyxisHTTPClient:          fakeClient,
		CertificationComponentID: "abc123",
		PyxisClient:              newSuccessfulFakePyxisClient(),
	})
	defer os.RemoveAll(extractDir)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if summary.BasedOnUbiPassed {
		t.Errorf("summary.BasedOnUbiPassed = true, want false")
	}

	got, err := ReadResults(summary.ExtractDir)
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
}

// TestRun_EndToEnd_WithSubmit drives the full submit-offline-artifacts
// pipeline using an injected fake lib.PyxisClient (via Options.PyxisClient)
// so the submission path is exercised without any real Pyxis network calls.
func TestRun_EndToEnd_WithSubmit(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)

	// LogFile is this invocation's own --logfile, separate from the
	// preflight.log extracted from the tarball above (Run() never touches it).
	logFile := filepath.Join(workDir, "preflight.log")
	if err := os.WriteFile(logFile, []byte("this invocation's own log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}

	summary, err := Run(context.Background(), Options{
		ArtifactsTarPath:         tarPath,
		LogFile:                  logFile,
		PyxisHost:                "fake-pyxis.example.com",
		PyxisHTTPClient:          fakeClient,
		CertificationComponentID: "abc123",
		PyxisClient:              newSuccessfulFakePyxisClient(),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !summary.Submitted {
		t.Errorf("summary.Submitted = false, want true")
	}

	// Run() always removes extractDir before returning, on every outcome.
	if _, statErr := os.Stat(summary.ExtractDir); !os.IsNotExist(statErr) {
		t.Errorf("expected extractDir %q to have been removed, stat err = %v", summary.ExtractDir, statErr)
	}
}

// TestRun_CleansUpExtractDirEvenWhenSubmitFails proves Run() still removes
// extractDir when submit() fails - there's no outcome where Run() returns
// while leaving its temporary extraction directory behind.
func TestRun_CleansUpExtractDirEvenWhenSubmitFails(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}
	fakePC := &fakePyxisClient{getProjectErr: errors.New("simulated submission failure")}

	summary, err := Run(context.Background(), Options{
		ArtifactsTarPath:         tarPath,
		PyxisHost:                "fake-pyxis.example.com",
		PyxisHTTPClient:          fakeClient,
		CertificationComponentID: "abc123",
		PyxisClient:              fakePC,
	})
	if err == nil {
		t.Fatal("expected an error when submission fails, got nil")
	}
	if summary == nil {
		t.Fatal("expected a non-nil summary even though submission failed")
	}

	if _, statErr := os.Stat(summary.ExtractDir); !os.IsNotExist(statErr) {
		t.Errorf("expected extractDir %q to have been removed, stat err = %v", summary.ExtractDir, statErr)
	}
}

// TestRun_EndToEnd_SubmitFails proves that a submission failure is
// propagated from Run(), even though results.json was already successfully
// updated on disk (i.e. the revalidation itself isn't lost).
func TestRun_EndToEnd_SubmitFails(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}
	fakePC := &fakePyxisClient{getProjectErr: errors.New("simulated submission failure")}

	// Calls the unexported run() directly (rather than Run()) so
	// results.json can be inspected below before it's cleaned up - Run()
	// always removes extractDir before returning.
	extractDir, summary, err := run(context.Background(), Options{
		ArtifactsTarPath:         tarPath,
		PyxisHost:                "fake-pyxis.example.com",
		PyxisHTTPClient:          fakeClient,
		CertificationComponentID: "abc123",
		PyxisClient:              fakePC,
	})
	defer os.RemoveAll(extractDir)
	if err == nil {
		t.Fatal("expected an error when submission fails, got nil")
	}

	// run() still returns a Summary (with ExtractDir populated) even on a
	// submission failure, since results.json was already successfully
	// updated on disk before submit() was attempted.
	if summary == nil {
		t.Fatal("expected a non-nil summary even though submission failed")
	}

	// results.json should still reflect the revalidated BasedOnUbi outcome.
	got, err := ReadResults(summary.ExtractDir)
	if err != nil {
		t.Fatalf("ReadResults() error = %v", err)
	}
	if !got.Passed {
		t.Errorf("results.json passed = false, want true even though submission failed")
	}
}

// TestRun_UsesDefaultHTTPClientWhenNilProvided proves Run() falls back to
// its own *http.Client when Options.PyxisHTTPClient is nil. PyxisHost points
// at a closed local port so the query fails immediately instead of hanging.
func TestRun_UsesDefaultHTTPClientWhenNilProvided(t *testing.T) {
	workDir := t.TempDir()
	tarPath := buildFakeArtifactsTar(t, workDir)

	_, err := Run(context.Background(), Options{
		ArtifactsTarPath: tarPath,
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
		PyxisHost:        "fake-pyxis.example.com",
	})
	if err == nil {
		t.Fatal("expected an error for a malformed cert-image.json")
	}
}

// TestRun_EndToEnd_CertImageMissingRawConfig proves a helpful error is
// returned when cert-image.json lacks the raw_config field
// diffIDsFromCertImage depends on.
func TestRun_EndToEnd_CertImageMissingRawConfig(t *testing.T) {
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
		PyxisHost:        "fake-pyxis.example.com",
	})
	if err == nil {
		t.Fatal("expected an error when cert-image.json has no raw_config")
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
		"raw_config": fakeRawConfig,
	})
	// results.json intentionally omitted.

	tarPath := filepath.Join(workDir, check.DefaultArtifactsTarFileName)
	tarDir(t, srcDir, tarPath)

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}

	_, err := Run(context.Background(), Options{
		ArtifactsTarPath: tarPath,
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
		PyxisHost:        "fake-pyxis.example.com",
	})
	if err == nil {
		t.Fatal("expected an error for a missing artifacts tar, got nil")
	}
}
