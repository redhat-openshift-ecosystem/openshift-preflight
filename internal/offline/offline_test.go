package offline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
)

// TestReadCertImage_MissingFile proves a helpful error is returned when
// cert-image.json does not exist in dir, rather than a panic.
func TestReadCertImage_MissingFile(t *testing.T) {
	dir := t.TempDir()

	if _, err := readCertImage(dir); err == nil {
		t.Fatal("expected an error when cert-image.json does not exist")
	}
}

// TestReadCertImage_InvalidJSON proves malformed JSON is reported clearly.
func TestReadCertImage_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, check.DefaultCertImageFilename), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := readCertImage(dir); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

func TestDiffIDsFromCertImage(t *testing.T) {
	validRawConfig := `{
		"architecture": "amd64",
		"os": "linux",
		"rootfs": {
			"type": "layers",
			"diff_ids": [
				"sha256:02f151e07d44d20fac36bab4b8e0dab4b7d81e6da75a4a3f30ab476e9c0ac2e5",
				"sha256:aa8823b5c71ec9b7dc06c1c2f60c1b2d573ac6b4b74d43ff77b8ad0c00a6f1a1"
			]
		}
	}`

	tests := []struct {
		name      string
		certImage *pyxis.CertImage
		wantCount int
		wantErr   bool
	}{
		{
			name:      "valid raw config",
			certImage: &pyxis.CertImage{RawConfig: validRawConfig},
			wantCount: 2,
		},
		{
			name:      "missing raw config",
			certImage: &pyxis.CertImage{},
			wantErr:   true,
		},
		{
			name:      "no diff ids",
			certImage: &pyxis.CertImage{RawConfig: `{"rootfs":{"type":"layers","diff_ids":[]}}`},
			wantErr:   true,
		},
		{
			name:      "invalid json",
			certImage: &pyxis.CertImage{RawConfig: "not json"},
			wantErr:   true,
		},
		{
			name:      "invalid diff id hash",
			certImage: &pyxis.CertImage{RawConfig: `{"rootfs":{"type":"layers","diff_ids":["not-a-hash"]}}`},
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hashes, err := diffIDsFromCertImage(test.certImage)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(hashes) != test.wantCount {
				t.Errorf("len(hashes) = %d, want %d", len(hashes), test.wantCount)
			}
		})
	}
}

// fakeLayerHashChecker is a test double for layerHashChecker.
type fakeLayerHashChecker struct {
	images []pyxis.CertImage
	err    error
}

func (f *fakeLayerHashChecker) CertifiedImagesContainingLayers(ctx context.Context, uncompressedLayerHashes []cranev1.Hash) ([]pyxis.CertImage, error) {
	return f.images, f.err
}

func TestRevalidateBasedOnUbi(t *testing.T) {
	hash, err := cranev1.NewHash("sha256:02f151e07d44d20fac36bab4b8e0dab4b7d81e6da75a4a3f30ab476e9c0ac2e5")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		checker layerHashChecker
		want    bool
		wantErr bool
	}{
		{
			name:    "at least one certified image found",
			checker: &fakeLayerHashChecker{images: []pyxis.CertImage{{ID: "abc123"}}},
			want:    true,
		},
		{
			name:    "no certified images found",
			checker: &fakeLayerHashChecker{images: nil},
			want:    false,
		},
		{
			name:    "pyxis query fails",
			checker: &fakeLayerHashChecker{err: errors.New("network unreachable")},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := revalidateBasedOnUbi(context.Background(), test.checker, []cranev1.Hash{hash})
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Errorf("revalidateBasedOnUbi() = %v, want %v", got, test.want)
			}
		})
	}
}
