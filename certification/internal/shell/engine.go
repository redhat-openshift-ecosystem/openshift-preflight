package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/certification/errors"
	"github.com/komish/preflight/certification/runtime"
	"github.com/sirupsen/logrus"
)

type CheckEngine struct {
	Image  string
	Checks []certification.Check

	results        runtime.Results
	localImagePath string
	isDownloaded   bool
}

// ExecuteChecks runs all checks stored in the check engine.
func (e *CheckEngine) ExecuteChecks(logger *logrus.Logger) {
	logger.Info("target image: ", e.Image)
	for _, check := range e.Checks {
		e.results.TestedImage = e.Image
		targetImage := e.Image

		// check if the image needs downloading
		if !e.isDownloaded {
			isRemote, err := e.ContainerIsRemote(e.Image, logger)
			if err != nil {
				logger.Error("unable to determine if the image was remote: ", err)
				e.results.Errors = append(e.results.Errors, check)
				continue
			}

			var localImagePath string
			if isRemote {
				logger.Info("downloading image")
				imageTarballPath, err := e.GetContainerFromRegistry(e.Image, logger)
				if err != nil {
					logger.Error("unable to the container from the registry: ", err)
					e.results.Errors = append(e.results.Errors, check)
					continue
				}

				logger.Debugf("Tarball path: %s", imageTarballPath)
				defer os.RemoveAll(filepath.Dir(imageTarballPath))

				localImagePath, err = e.ExtractContainerTar(imageTarballPath, logger)
				if err != nil {
					logger.Error("unable to extract the container: ", err)
					e.results.Errors = append(e.results.Errors, check)
					continue
				}
			}

			e.localImagePath = localImagePath
		}

		// if we downloaded an image to disk, lets test against that.
		// COMMENTED: tests aren't currently written to support this
		// remove if we decide we do not care to have a tarball.
		if len(e.localImagePath) != 0 {
			targetImage = e.localImagePath
		}

		logger.Info("running check: ", check.Name())
		// run the validation
		passed, err := check.Validate(targetImage, logger)

		if err != nil {
			logger.WithFields(logrus.Fields{"result": "ERROR", "error": err.Error()}).Info("check completed: ", check.Name())
			e.results.Errors = append(e.results.Errors, check)
			continue
		}

		if !passed {
			logger.WithFields(logrus.Fields{"result": "FAILED"}).Info("check completed: ", check.Name())
			e.results.Failed = append(e.results.Failed, check)
			continue
		}

		logger.WithFields(logrus.Fields{"result": "PASSED"}).Info("check completed: ", check.Name())
		e.results.Passed = append(e.results.Passed, check)
	}
}

// StoreCheck stores a given check that needs to be executed in the check engine.
func (e *CheckEngine) StoreCheck(checks ...certification.Check) {
	e.Checks = append(e.Checks, checks...)
}

// Results will return the results of check execution.
func (e *CheckEngine) Results() runtime.Results {
	return e.results
}

func (e *CheckEngine) ExtractContainerTar(tarball string, logger *logrus.Logger) (string, error) {
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

func (e *CheckEngine) GetContainerFromRegistry(containerLoc string, logger *logrus.Logger) (string, error) {
	stdouterr, err := exec.Command("podman", "pull", containerLoc).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrGetRemoteContainerFailed, err)
	}
	lines := strings.Split(string(stdouterr), "\n")

	tempdir, err := os.MkdirTemp(os.TempDir(), "preflight-*")
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrCreateTempDir, err)
	}

	imgSig := lines[len(lines)-2]
	tarpath := filepath.Join(tempdir, imgSig+".tar")
	_, err = exec.Command("podman", "save", containerLoc, "--output", tarpath).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrSaveContainerFailed, err)
	}

	e.isDownloaded = true
	return tarpath, nil
}

func (e *CheckEngine) ContainerIsRemote(path string, logger *logrus.Logger) (bool, error) {
	// TODO: Implement, for not this is just returning
	// that the resource is remote and needs to be pulled.
	return true, nil
}
