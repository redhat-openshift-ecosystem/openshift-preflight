package replay

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// setupMultiArchRoot creates a <root>/<platform>/artifacts.tar layout for
// each of the given platforms, mimicking what
// `check container --offline` writes for a manifest-list image.
func setupMultiArchRoot(t *testing.T, platforms []string) string {
	t.Helper()

	root := t.TempDir()
	for _, platform := range platforms {
		platformDir := filepath.Join(root, platform)
		if err := os.MkdirAll(platformDir, 0o755); err != nil {
			t.Fatal(err)
		}
		buildFakeArtifactsTar(t, platformDir) // writes <platformDir>/src and <platformDir>/artifacts.tar
	}

	return root
}

func TestDiscoverPlatformTars(t *testing.T) {
	root := setupMultiArchRoot(t, []string{"amd64", "arm64", "s390x"})

	got, err := DiscoverPlatformTars(root)
	if err != nil {
		t.Fatalf("DiscoverPlatformTars() error = %v", err)
	}

	want := map[string]bool{"amd64": true, "arm64": true, "s390x": true}
	if len(got) != len(want) {
		t.Fatalf("got %d platforms, want %d: %+v", len(got), len(want), got)
	}
	for platform := range want {
		path, ok := got[platform]
		if !ok {
			t.Errorf("expected platform %q to be discovered", platform)
			continue
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("discovered tar for %q does not exist: %v", platform, err)
		}
	}
}

func TestDiscoverPlatformTars_IgnoresNonPlatformDirs(t *testing.T) {
	root := setupMultiArchRoot(t, []string{"amd64"})
	// An unrelated empty directory (e.g. a stray dir) should be ignored, not error.
	if err := os.MkdirAll(filepath.Join(root, "not-a-platform"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := DiscoverPlatformTars(root)
	if err != nil {
		t.Fatalf("DiscoverPlatformTars() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d platforms, want 1: %+v", len(got), got)
	}
	if _, ok := got["amd64"]; !ok {
		t.Errorf("expected amd64 to be discovered, got %+v", got)
	}
}

func TestDiscoverPlatformTars_NoneFound(t *testing.T) {
	root := t.TempDir()
	if _, err := DiscoverPlatformTars(root); err == nil {
		t.Fatal("expected an error when no platform subdirectory contains an artifacts.tar")
	}
}

func TestDiscoverPlatformTars_MissingRoot(t *testing.T) {
	if _, err := DiscoverPlatformTars(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected an error for a missing root directory")
	}
}

func TestRunBatch_AllPlatformsSucceed(t *testing.T) {
	root := setupMultiArchRoot(t, []string{"amd64", "arm64"})
	extractRoot := filepath.Join(t.TempDir(), "extracted")

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}

	batch, err := RunBatch(context.Background(), root, Options{
		ExtractDir:      extractRoot,
		PyxisHost:       "fake-pyxis.example.com",
		PyxisHTTPClient: fakeClient,
		Submit:          false,
	})
	if err != nil {
		t.Fatalf("RunBatch() error = %v", err)
	}

	if !batch.AllSucceeded() {
		t.Fatalf("expected all platforms to succeed, got errors: %+v", batch.Errors)
	}
	if len(batch.Platforms) != 2 {
		t.Fatalf("got %d platform results, want 2: %+v", len(batch.Platforms), batch.Platforms)
	}

	for _, platform := range []string{"amd64", "arm64"} {
		summary, ok := batch.Platforms[platform]
		if !ok {
			t.Errorf("expected a result for platform %q", platform)
			continue
		}
		if !summary.BasedOnUbiPased {
			t.Errorf("platform %q: BasedOnUbiPased = false, want true", platform)
		}

		// Verify each platform extracted into its own subdirectory and has
		// a correctly updated results.json - proving platforms don't clobber
		// each other.
		wantExtractDir := filepath.Join(extractRoot, platform)
		if summary.ExtractDir != wantExtractDir {
			t.Errorf("platform %q: ExtractDir = %q, want %q", platform, summary.ExtractDir, wantExtractDir)
		}

		results, err := ReadResults(summary.ExtractDir)
		if err != nil {
			t.Fatalf("platform %q: ReadResults() error = %v", platform, err)
		}
		if !results.Passed {
			t.Errorf("platform %q: results.json passed = false, want true", platform)
		}
	}
}

func TestRunBatch_OnePlatformFailsDoesNotStopOthers(t *testing.T) {
	root := setupMultiArchRoot(t, []string{"amd64", "arm64"})

	// Corrupt the arm64 tarball so it fails to extract, while amd64 stays valid.
	if err := os.WriteFile(filepath.Join(root, "arm64", "artifacts.tar"), []byte("not a tar file"), 0o644); err != nil {
		t.Fatal(err)
	}

	fakeClient := &http.Client{Transport: localRoundTripper{handler: fakePyxisGraphQLHandler(true)}}

	batch, err := RunBatch(context.Background(), root, Options{
		ExtractDir:      filepath.Join(t.TempDir(), "extracted"),
		PyxisHost:       "fake-pyxis.example.com",
		PyxisHTTPClient: fakeClient,
	})
	if err != nil {
		t.Fatalf("RunBatch() error = %v", err)
	}

	if batch.AllSucceeded() {
		t.Fatal("expected arm64 to fail")
	}
	if _, ok := batch.Errors["arm64"]; !ok {
		t.Errorf("expected an error recorded for arm64, got %+v", batch.Errors)
	}
	if _, ok := batch.Platforms["amd64"]; !ok {
		t.Errorf("expected amd64 to still succeed despite arm64 failing, got %+v", batch.Platforms)
	}
}

func TestRunBatch_MissingRoot(t *testing.T) {
	if _, err := RunBatch(context.Background(), filepath.Join(t.TempDir(), "nope"), Options{}); err == nil {
		t.Fatal("expected an error for a missing root directory")
	}
}
