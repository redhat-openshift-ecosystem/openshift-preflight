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

// newSuccessfulFakePyxisClient returns a fake that succeeds at GetProject and
// SubmitResults. scratch controls CertProject.ScratchProject().
func newSuccessfulFakePyxisClient(scratch bool) *fakePyxisClient {
	containerType := ""
	if scratch {
		containerType = "scratch"
	}
	return &fakePyxisClient{
		project: &pyxis.CertProject{ID: "abc123", Container: pyxis.Container{Type: containerType}},
		submitResults: &pyxis.CertificationResults{
			CertImage:   &pyxis.CertImage{ID: "img123"},
			TestResults: &pyxis.TestResults{ID: "tr123"},
		},
	}
}

// TestSubmit_WithInjectedClient_Success drives submit() through a successful
// ContainerCertificationSubmitter.Submit using a pre-fetched certProject.
func TestSubmit_WithInjectedClient_Success(t *testing.T) {
	dir := t.TempDir()

	writeJSON(t, filepath.Join(dir, check.DefaultCertImageFilename), map[string]any{"docker_image_digest": "sha256:deadbeef"})
	writeJSON(t, filepath.Join(dir, check.DefaultTestResultsFilename), map[string]any{"image": "quay.io/example/myapp:latest", "passed": true})

	logFile := filepath.Join(dir, "this-run.log")
	if err := os.WriteFile(logFile, []byte("fake preflight log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fakePC := newSuccessfulFakePyxisClient(true)
	err := submit(context.Background(), dir, Options{
		CertificationComponentID: "abc123",
		PyxisClient:              fakePC,
		LogFile:                  logFile,
	}, fakePC, fakePC.project)
	if err != nil {
		t.Fatalf("submit() error = %v", err)
	}
}

// TestSubmit_WithInjectedClient_SubmitResultsFails proves submitter errors
// are propagated when CertProject is already provided.
func TestSubmit_WithInjectedClient_SubmitResultsFails(t *testing.T) {
	dir := t.TempDir()

	writeJSON(t, filepath.Join(dir, check.DefaultCertImageFilename), map[string]any{"docker_image_digest": "sha256:deadbeef"})
	writeJSON(t, filepath.Join(dir, check.DefaultTestResultsFilename), map[string]any{"image": "quay.io/example/myapp:latest", "passed": true})
	logFile := filepath.Join(dir, "this-run.log")
	if err := os.WriteFile(logFile, []byte("fake preflight log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	project := &pyxis.CertProject{ID: "abc123", Container: pyxis.Container{Type: "scratch"}}
	fakePC := &fakePyxisClient{
		project:   project,
		submitErr: errors.New("submit failed"),
	}

	err := submit(context.Background(), dir, Options{
		CertificationComponentID: "abc123",
		PyxisClient:              fakePC,
		LogFile:                  logFile,
	}, fakePC, project)
	if err == nil {
		t.Fatal("expected an error when SubmitResults fails, got nil")
	}
}
