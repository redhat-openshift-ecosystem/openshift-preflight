package offline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
)

// fakePyxisClient is a minimal test double for lib.PyxisClient, used to
// exercise submit() without making real Pyxis network calls.
type fakePyxisClient struct {
	project       *pyxis.CertProject
	getProjectErr error

	submitResults *pyxis.CertificationResults
	submitErr     error
}

func (f *fakePyxisClient) FindImagesByDigest(ctx context.Context, digests []string) ([]pyxis.CertImage, error) {
	return nil, nil
}

func (f *fakePyxisClient) GetProject(ctx context.Context) (*pyxis.CertProject, error) {
	return f.project, f.getProjectErr
}

func (f *fakePyxisClient) SubmitResults(ctx context.Context, input *pyxis.CertificationInput) (*pyxis.CertificationResults, error) {
	return f.submitResults, f.submitErr
}

// newSuccessfulFakePyxisClient returns a fakePyxisClient that succeeds at
// both GetProject and SubmitResults, for tests where submission isn't the
// focus but Run() always submits so it must still succeed.
func newSuccessfulFakePyxisClient() *fakePyxisClient {
	return &fakePyxisClient{
		project: &pyxis.CertProject{ID: "abc123", Container: pyxis.Container{Type: "scratch"}},
		submitResults: &pyxis.CertificationResults{
			CertImage:   &pyxis.CertImage{ID: "img123"},
			TestResults: &pyxis.TestResults{ID: "tr123"},
		},
	}
}

// TestSubmit_BuildsPyxisClientWhenNotInjected proves submit() builds a real
// lib.PyxisClient when Options.PyxisClient isn't injected, without making a
// real network call (the target host refuses the connection immediately).
func TestSubmit_BuildsPyxisClientWhenNotInjected(t *testing.T) {
	dir := t.TempDir()

	err := submit(context.Background(), dir, Options{
		CertificationComponentID: "abc123",
		PyxisAPIToken:            "some-token",
		PyxisHost:                "127.0.0.1:1", // nothing listens here; connection refused immediately
	})
	if err == nil {
		t.Fatal("expected an error querying a closed local port")
	}
}

// TestSubmit_WithInjectedClient_GetProjectFails proves an injected
// Options.PyxisClient is used instead of building one, and that submitter
// errors are propagated.
func TestSubmit_WithInjectedClient_GetProjectFails(t *testing.T) {
	dir := t.TempDir()

	fakePC := &fakePyxisClient{getProjectErr: errors.New("network unreachable")}

	err := submit(context.Background(), dir, Options{
		CertificationComponentID: "abc123",
		PyxisClient:              fakePC,
	})
	if err == nil {
		t.Fatal("expected an error when GetProject fails, got nil")
	}
}

// TestSubmit_WithInjectedClient_Success drives submit() through a
// successful ContainerCertificationSubmitter.Submit call using a fake Pyxis
// client, proving the artifact-reading/submission wiring works end to end.
func TestSubmit_WithInjectedClient_Success(t *testing.T) {
	dir := t.TempDir()

	writeJSON(t, filepath.Join(dir, check.DefaultCertImageFilename), map[string]any{"docker_image_digest": "sha256:deadbeef"})
	writeJSON(t, filepath.Join(dir, check.DefaultTestResultsFilename), map[string]any{"image": "quay.io/example/myapp:latest", "passed": true})

	// This invocation's own log, distinct from any preflight.log extracted into dir.
	logFile := filepath.Join(dir, "this-run.log")
	if err := os.WriteFile(logFile, []byte("fake preflight log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fakePC := &fakePyxisClient{
		// Container.Type == "scratch" makes ScratchProject() true, so submit
		// doesn't also need a rpm-manifest.json on disk (scratch images
		// don't have one).
		project: &pyxis.CertProject{ID: "abc123", Container: pyxis.Container{Type: "scratch"}},
		submitResults: &pyxis.CertificationResults{
			CertImage:   &pyxis.CertImage{ID: "img123"},
			TestResults: &pyxis.TestResults{ID: "tr123"},
		},
	}

	err := submit(context.Background(), dir, Options{
		CertificationComponentID: "abc123",
		PyxisClient:              fakePC,
		LogFile:                  logFile,
	})
	if err != nil {
		t.Fatalf("submit() error = %v", err)
	}
}
