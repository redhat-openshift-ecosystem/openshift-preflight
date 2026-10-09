package replay

import (
	"archive/tar"
	"errors"
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

func TestExtractArtifactsTar_DestDirIsFile(t *testing.T) {
	root := t.TempDir()
	tarPath := filepath.Join(root, "archive.tar")
	dest := filepath.Join(root, "dest")

	// Create dest as a regular file so os.MkdirAll(dest) fails.
	if err := os.WriteFile(dest, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeTar(t, tarPath, []tar.Header{
		{Name: "foo.txt", Typeflag: tar.TypeReg, Mode: 0o644},
	}, map[string]string{"foo.txt": "hi"})

	if err := ExtractArtifactsTar(tarPath, dest); err == nil {
		t.Fatal("expected an error when destDir is occupied by a regular file")
	}
}

func TestExtractArtifactsTar_CorruptTar(t *testing.T) {
	root := t.TempDir()
	tarPath := filepath.Join(root, "corrupt.tar")
	dest := filepath.Join(root, "dest")

	// A tar header is 512 bytes; a short garbage file makes tar.Reader.Next()
	// return a non-io.EOF error.
	if err := os.WriteFile(tarPath, []byte("not a valid tar header, too short"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ExtractArtifactsTar(tarPath, dest); err == nil {
		t.Fatal("expected an error for a corrupt tar archive")
	}
}

func TestExtractArtifactsTar_DirEntryCollidesWithFile(t *testing.T) {
	root := t.TempDir()
	tarPath := filepath.Join(root, "archive.tar")
	dest := filepath.Join(root, "dest")

	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-create "sub" as a regular file so the tar's directory entry "sub"
	// can't be created.
	if err := os.WriteFile(filepath.Join(dest, "sub"), []byte("occupied"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeTar(t, tarPath, []tar.Header{
		{Name: "sub", Typeflag: tar.TypeDir, Mode: 0o755},
	}, nil)

	if err := ExtractArtifactsTar(tarPath, dest); err == nil {
		t.Fatal("expected an error when a directory entry collides with an existing file")
	}
}

func TestExtractArtifactsTar_RegEntryParentCollidesWithFile(t *testing.T) {
	root := t.TempDir()
	tarPath := filepath.Join(root, "archive.tar")
	dest := filepath.Join(root, "dest")

	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-create "sub" as a regular file so a nested entry's parent
	// directory can't be created.
	if err := os.WriteFile(filepath.Join(dest, "sub"), []byte("occupied"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeTar(t, tarPath, []tar.Header{
		{Name: "sub/file.txt", Typeflag: tar.TypeReg, Mode: 0o644},
	}, map[string]string{"sub/file.txt": "hi"})

	if err := ExtractArtifactsTar(tarPath, dest); err == nil {
		t.Fatal("expected an error when a regular entry's parent directory collides with an existing file")
	}
}

func TestExtractArtifactsTar_RegEntryTargetIsDirectory(t *testing.T) {
	root := t.TempDir()
	tarPath := filepath.Join(root, "archive.tar")
	dest := filepath.Join(root, "dest")

	// "file.txt" already exists as a directory at the target path.
	if err := os.MkdirAll(filepath.Join(dest, "file.txt"), 0o755); err != nil {
		t.Fatal(err)
	}

	writeTar(t, tarPath, []tar.Header{
		{Name: "file.txt", Typeflag: tar.TypeReg, Mode: 0o644},
	}, map[string]string{"file.txt": "hi"})

	if err := ExtractArtifactsTar(tarPath, dest); err == nil {
		t.Fatal("expected an error when a regular entry's target already exists as a directory")
	}
}

// errReader always fails, used to exercise writeRegularFile's io.Copy error path.
type errReader struct{}

func (errReader) Read(p []byte) (int, error) {
	return 0, errors.New("simulated read failure")
}

func TestWriteRegularFile_CopyFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")

	if err := writeRegularFile(errReader{}, target, 0o644); err == nil {
		t.Fatal("expected an error when the reader fails")
	}
}
