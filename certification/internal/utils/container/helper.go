package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
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

func GetContainerFromRegistry(podmanEngine cli.PodmanEngine, imageLoc string) (string, error) {
	pullReport, err := podmanEngine.Pull(imageLoc, cli.ImagePullOptions{})
	if err != nil {
		return pullReport.StdoutErr, fmt.Errorf("%w: %s", errors.ErrGetRemoteContainerFailed, err)
	}

	return pullReport.StdoutErr, nil
}

// Calls podman to save the image to a temporary directory in /tmp/preflight-{randomString}.
// Should be followed with:
// defer os.RemoveAll(tarballDir)
// Returns the locataion of the tarball.
func SaveContainerToFilesystem(podmanEngine cli.PodmanEngine, imageLog string) (string, error) {
	inspectReport, err := podmanEngine.InspectImage(imageLog, cli.ImageInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrImageInspectFailed, err)
	}

	tempdir, err := os.MkdirTemp(os.TempDir(), "preflight-*")
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrCreateTempDir, err)
	}

	imgSig := inspectReport.Images[0].Id
	tarpath := filepath.Join(tempdir, imgSig+".tar")
	err = podmanEngine.Save(imgSig, []string{}, cli.ImageSaveOptions{Destination: tarpath})
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrSaveContainerFailed, err)
	}
	return tarpath, nil
}

type ContainerFn func(string) (bool, error)

func unmountAndRemove(podmanEngine cli.PodmanEngine, containerId string) {
	podmanEngine.Unmount(containerId)
	podmanEngine.Remove(containerId)
}

// RunInsideContainerFS executes a provided function by creating a container,
// based on the passed image, and mounting the filsystem. This allows the passed
// function to operate on the filesystem natively, without having to do further
// "shell outs".
func RunInsideContainerFS(podmanEngine cli.PodmanEngine, image string, containerFn ContainerFn) (bool, error) {
	createResult, err := podmanEngine.Create(image, &cli.PodmanCreateOptions{})
	if err != nil {
		log.Error("could not retrieve containerId", err)
		return false, err
	}
	containerId := createResult.ContainerID

	report, err := podmanEngine.Mount(containerId)
	if err != nil {
		log.Error("could not mount filesystem", err)
		return false, err
	}

	defer unmountAndRemove(podmanEngine, containerId)

	return containerFn(report.MountDir)
}
