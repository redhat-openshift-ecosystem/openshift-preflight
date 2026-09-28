package replay

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"
)

// writeTar creates a plain (non-gzipped) tar archive at path containing entries.
func writeTar(t *testing.T, path string, entries []tar.Header, contents map[string]string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	defer tw.Close()

	for i := range entries {
		h := entries[i]
		body := contents[h.Name]
		h.Size = int64(len(body))
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if body != "" {
			if _, err := tw.Write([]byte(body)); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestExtractArtifactsTar_ExtractsFilesAndDirs(t *testing.T) {
	root := t.TempDir()
	tarPath := filepath.Join(root, "artifacts.tar")
	dest := filepath.Join(root, "extracted")

	writeTar(t, tarPath, []tar.Header{
		{Name: "cert-image.json", Typeflag: tar.TypeReg, Mode: 0o644},
		{Name: "sub", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "sub/results.json", Typeflag: tar.TypeReg, Mode: 0o644},
	}, map[string]string{
		"cert-image.json":  `{"hello":"world"}`,
		"sub/results.json": `{"passed":true}`,
	})

	if err := ExtractArtifactsTar(tarPath, dest); err != nil {
		t.Fatalf("ExtractArtifactsTar() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "cert-image.json"))
	if err != nil {
		t.Fatalf("expected cert-image.json to be extracted: %v", err)
	}
	if string(got) != `{"hello":"world"}` {
		t.Errorf("cert-image.json contents = %q", got)
	}

	got, err = os.ReadFile(filepath.Join(dest, "sub", "results.json"))
	if err != nil {
		t.Fatalf("expected nested file to be extracted: %v", err)
	}
	if string(got) != `{"passed":true}` {
		t.Errorf("sub/results.json contents = %q", got)
	}
}

func TestExtractArtifactsTar_RejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	tarPath := filepath.Join(root, "archive.tar")
	dest := filepath.Join(root, "dest")

	writeTar(t, tarPath, []tar.Header{
		{Name: "../escape.txt", Typeflag: tar.TypeReg, Mode: 0o644},
	}, map[string]string{"../escape.txt": "pwned"})

	if err := ExtractArtifactsTar(tarPath, dest); err == nil {
		t.Fatal("expected an error for a path-traversal entry, got nil")
	}

	if _, err := os.Stat(filepath.Join(root, "escape.txt")); !os.IsNotExist(err) {
		t.Errorf("path traversal entry should not have been written outside dest: %v", err)
	}
}

func TestExtractArtifactsTar_RejectsAbsolutePath(t *testing.T) {
	root := t.TempDir()
	tarPath := filepath.Join(root, "archive.tar")
	dest := filepath.Join(root, "dest")

	writeTar(t, tarPath, []tar.Header{
		{Name: "/etc/passwd", Typeflag: tar.TypeReg, Mode: 0o644},
	}, map[string]string{"/etc/passwd": "pwned"})

	if err := ExtractArtifactsTar(tarPath, dest); err == nil {
		t.Fatal("expected an error for an absolute path entry, got nil")
	}
}

func TestExtractArtifactsTar_RejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	tarPath := filepath.Join(root, "archive.tar")
	dest := filepath.Join(root, "dest")

	writeTar(t, tarPath, []tar.Header{
		{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd", Mode: 0o777},
	}, nil)

	if err := ExtractArtifactsTar(tarPath, dest); err == nil {
		t.Fatal("expected an error for a symlink entry, got nil")
	}
}

func TestExtractArtifactsTar_MissingSourceFile(t *testing.T) {
	dest := t.TempDir()
	if err := ExtractArtifactsTar(filepath.Join(dest, "does-not-exist.tar"), dest); err == nil {
		t.Fatal("expected an error when the source tar does not exist, got nil")
	}
}
