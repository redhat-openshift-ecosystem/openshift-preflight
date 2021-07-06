package shell

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
)

func ExtractContainerTar(tarball string) (string, error) {
	// we assume the input path is something like "abcdefg.tar", representing a container image,
	// so we need to remove the extension.
	containerIDSlice := strings.Split(filepath.Base(tarball), ".tar")
	if len(containerIDSlice) != 2 {
		// we expect a single entry in the slice, otherwise we split incorrectly
		return "", fmt.Errorf("%w: %s: %s", errors.ErrExtractingTarball, "received an improper container tarball name to extract", tarball)
	}

	outputDir := filepath.Join(filepath.Dir(tarball), containerIDSlice[0])
	err := os.Mkdir(outputDir, 0755)
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrExtractingTarball, err)
	}

	_, err = exec.Command("tar", "xvf", tarball, "--directory", outputDir).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrExtractingTarball, err)
	}

	return outputDir, nil
}

func GetContainerFromRegistry(imageLoc string) error {
	_, err := exec.Command("podman", "pull", imageLoc).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrGetRemoteContainerFailed, err)
	}

	return nil
}

// Calls podman to save the image to a temporary directory in /tmp/preflight-???, where
// ??? will be a a generated string. Should be follwed with:
// defer os.RemoveAll(tarballDir)
// Returns the locataion of the tarball.
func SaveContainerToFilesystem(imageLoc string) (string, error) {
	cmd := exec.Command("podman", "image", "inspect", "--format", "{{.Id}}", imageLoc)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrExtractingTarball, err)
	}

	tempdir, err := os.MkdirTemp(os.TempDir(), "preflight-*")
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrCreateTempDir, err)
	}

	imgSig := stdout.String()
	tarpath := filepath.Join(tempdir, imgSig+".tar")
	_, err = exec.Command("podman", "save", imageLoc, "--output", tarpath).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrSaveContainerFailed, err)
	}
	return tarpath, nil
}
