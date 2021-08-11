package shell

import (
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	fileutils "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/file"
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

// RunInsideImageFS executes a provided function by mounting the image filesystem
// to the host. Note that any ContainerFn that is expected to run using this function
// should know that the input is a filesystem path.
func RunInsideImageFS(podmanEngine cli.PodmanEngine, image string, containerFn ContainerFn) (bool, error) {
	report, err := podmanEngine.MountImage(image)
	if err != nil {
		log.Error("stdout: ", report.Stdout)
		log.Error("stderr: ", report.Stderr)
		log.Error("could not mount filesystem", err)
		return false, err
	}

	defer func() {
		report, err := podmanEngine.UnmountImage(image)
		if err != nil {
			log.Warn("stdout: ", report.Stdout)
			log.Warn("stderr: ", report.Stderr)
		}
	}()

	return containerFn(strings.TrimSpace(report.MountDir))
}

func GenerateBundleHash(podmanEngine cli.PodmanEngine, image string) (string, error) {
	hashCmd := `find . -not -name "Dockerfile" -type f -printf '%f\t%p\n' | sort -V -k1 | cut -d$'\t' -f2 | tr '\n' '\0' | xargs -r0 -I {} md5sum "{}"` // >> $HOME/hashes.txt`
	bundleCmd := fmt.Sprintf("pushd $(podman image mount %[1]s) &> /dev/null && %[2]s && popd &> /dev/null && podman image unmount %[1]s &> /dev/null", image, hashCmd)
	report, err := podmanEngine.Unshare(map[string]string{}, "/bin/bash", "-c", bundleCmd)
	if err != nil {
		log.Errorf("could not generate bundle hash")
		log.Debugf(fmt.Sprintf("Stdout: %s", report.Stdout))
		log.Debugf(fmt.Sprintf("Stderr: %s", report.Stderr))
		return "", err
	}
	log.Tracef(fmt.Sprintf("Hash is: %s", report.Stdout))
	err = os.WriteFile(fileutils.ArtifactPath("hashes.txt"), []byte(report.Stdout), 0644)
	if err != nil {
		log.Errorf("could not write bundle hash file")
		return "", err
	}
	sum := md5.Sum([]byte(report.Stdout))

	return fmt.Sprintf("%x", sum), nil
}
