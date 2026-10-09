package replay

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractArtifactsTar extracts the plain (non-gzipped) tarball written by
// `preflight check container --offline` (see check.DefaultArtifactsTarFileName)
// into destDir. destDir is created if it does not already exist.
//
// Only regular files and directories are supported - this tarball is produced
// by preflight itself from a flat artifacts directory, so anything else
// (symlinks, devices, etc.) is treated as unexpected/unsafe and rejected.
// Entries that would escape destDir (via "../" or an absolute path) are
// rejected as well.
func ExtractArtifactsTar(tarPath, destDir string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("could not open artifacts tar %s: %w", tarPath, err)
	}
	defer f.Close()

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("could not create extraction directory %s: %w", destDir, err)
	}

	tr := tar.NewReader(f)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("could not read next tar entry: %w", err)
		}

		target, err := safeJoin(destDir, header.Name)
		if err != nil {
			return fmt.Errorf("refusing to extract %q: %w", header.Name, err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("could not create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("could not create parent directory for %s: %w", target, err)
			}

			if err := writeRegularFile(tr, target, header.FileInfo().Mode()); err != nil {
				return err
			}
		default:
			return fmt.Errorf("refusing to extract %q: unsupported tar entry type %v", header.Name, header.Typeflag)
		}
	}

	return nil
}

// writeRegularFile copies the current tar entry (r) into a new file at
// target with the given permissions.
func writeRegularFile(r io.Reader, target string, mode os.FileMode) error {
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", target, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, r); err != nil { //nolint:gosec // artifacts.tar is a trusted, preflight-generated archive of bounded size.
		return fmt.Errorf("could not write file %s: %w", target, err)
	}

	return nil
}

// safeJoin joins base and name, and verifies the result is still contained
// within base - rejecting absolute paths and "../" traversal (i.e. zip-slip).
func safeJoin(base, name string) (string, error) {
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	cleanName := filepath.Clean(name)
	if cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal is not allowed")
	}

	target := filepath.Join(base, cleanName)
	cleanBase := filepath.Clean(base)
	if target != cleanBase && !strings.HasPrefix(target, cleanBase+string(filepath.Separator)) {
		// Defense in depth: filepath.Clean guarantees cleanName can't reach
		// here without already being caught by the ".." check above, so this
		// branch is unreachable through the public API, but kept in case
		// that invariant ever changes.
		//coverage:ignore
		return "", fmt.Errorf("resulting path escapes destination directory")
	}

	return target, nil
}
