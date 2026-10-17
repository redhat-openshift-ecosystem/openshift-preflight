package offline

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// extractArtifactsTar extracts the plain (non-gzipped) tarball written by
// `preflight check container --offline` into destDir (created if needed).
// Only regular, non-escaping entries are allowed; os.Root blocks symlink escapes.
func extractArtifactsTar(tarPath, destDir string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("could not open artifacts tar %s: %w", tarPath, err)
	}
	defer f.Close()

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("could not create extraction directory %s: %w", destDir, err)
	}

	root, err := os.OpenRoot(destDir)
	if err != nil {
		//coverage:ignore
		return fmt.Errorf("could not open extraction directory %s: %w", destDir, err)
	}
	defer root.Close()

	tr := tar.NewReader(f)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("could not read next tar entry: %w", err)
		}

		entryPath, err := safeEntryPath(header.Name)
		if err != nil {
			return fmt.Errorf("refusing to extract %q: %w", header.Name, err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := root.MkdirAll(entryPath, 0o755); err != nil {
				return fmt.Errorf("could not create directory %s: %w", entryPath, err)
			}
		case tar.TypeReg:
			if dir := filepath.Dir(entryPath); dir != "." {
				if err := root.MkdirAll(dir, 0o755); err != nil {
					return fmt.Errorf("could not create parent directory for %s: %w", entryPath, err)
				}
			}

			if err := writeRegularFile(root, tr, entryPath, header.FileInfo().Mode()); err != nil {
				return err
			}
		default:
			return fmt.Errorf("refusing to extract %q: unsupported tar entry type %v", header.Name, header.Typeflag)
		}
	}

	return nil
}

// writeRegularFile copies the current tar entry (r) into a new file at
// entryPath (resolved relative to root, and confined beneath it even across
// symlinks) with the given permissions.
func writeRegularFile(root *os.Root, r io.Reader, entryPath string, mode os.FileMode) error {
	out, err := root.OpenFile(entryPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", entryPath, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, r); err != nil { //nolint:gosec // artifacts.tar is a trusted, preflight-generated archive of bounded size.
		return fmt.Errorf("could not write file %s: %w", entryPath, err)
	}

	return nil
}

// safeEntryPath rejects absolute paths and "../" traversal (zip-slip) via
// filepath.IsLocal - a lexical check on the name, complementary to (not a
// replacement for) extractArtifactsTar's os.Root symlink confinement.
func safeEntryPath(name string) (string, error) {
	entryPath := filepath.FromSlash(name)
	if !filepath.IsLocal(entryPath) {
		return "", fmt.Errorf("escapes destination directory")
	}

	return filepath.Clean(entryPath), nil
}
