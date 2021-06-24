package podmanexec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/certification/errors"
	"github.com/komish/preflight/certification/runtime"
)

type PolicyEngine struct {
	Image    string
	Policies []certification.Policy

	results        runtime.Results
	localImagePath string
	isDownloaded   bool
}

// ExecutePolicies runs all policies stored in the policy engine.
func (e *PolicyEngine) ExecutePolicies() {
	for _, policy := range e.Policies {
		e.results.TestedImage = e.Image
		targetImage := e.Image

		// check if the image needs downloading
		if !e.isDownloaded {
			// TODO: consider returning an error in the ExecutePolicies instead
			// of logging those policies as errors in the output.
			isRemote, err := e.ContainerIsRemote(e.Image)
			if err != nil {
				e.results.Errors = append(e.results.Errors, policy)
				continue
			}

			var localImagePath string
			if isRemote {
				imageTarballPath, err := e.GetContainerFromRegistry(e.Image)
				if err != nil {
					e.results.Errors = append(e.results.Errors, policy)
					continue
				}

				localImagePath, err = e.ExtractContainerTar(imageTarballPath)
				if err != nil {
					e.results.Errors = append(e.results.Errors, policy)
					continue
				}
			}

			e.localImagePath = localImagePath
		}

		// if we downloaded an image to disk, lets test against that.
		if len(e.localImagePath) == 0 {
			targetImage = e.localImagePath
		}

		// run the validation
		passed, err := policy.Validate(targetImage)

		if err != nil {
			e.results.Errors = append(e.results.Errors, policy)
			continue
		}

		if !passed {
			e.results.Failed = append(e.results.Failed, policy)
			continue
		}

		e.results.Passed = append(e.results.Passed, policy)
	}
}

// StorePolicy stores a given policy that needs to be executed in the policy runner.
func (e *PolicyEngine) StorePolicies(policies ...certification.Policy) {
	e.Policies = append(e.Policies, policies...)
}

// Results will return the results of policy execution.
func (e *PolicyEngine) Results() runtime.Results {
	return e.results
}

func (e *PolicyEngine) ExtractContainerTar(tarball string) (string, error) {
	// we assume the input path is something like "abcdefg.tar", representing a container image,
	// so we need to remove the extension.
	containerIDSlice := strings.Split(tarball, ".tar")
	if len(containerIDSlice) != 2 {
		// we expect a single entry in the slice, otherwise we split incorrectly
		return "", fmt.Errorf("%w: %s: %s", errors.ErrExtractingTarball, "received an improper container tarball name to extract", tarball)
	}

	outputDir := containerIDSlice[0]
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

func (e *PolicyEngine) GetContainerFromRegistry(containerLoc string) (string, error) {
	stdouterr, err := exec.Command("podman", "pull", containerLoc).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrGetRemoteContainerFailed, err)
	}
	lines := strings.Split(string(stdouterr), "\n")

	imgSig := lines[len(lines)-2]
	_, err = exec.Command("podman", "save", containerLoc, "--output", imgSig+".tar").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", errors.ErrSaveContainerFailed, err)
	}

	e.isDownloaded = true
	return imgSig + ".tar", nil
}

func (e *PolicyEngine) ContainerIsRemote(path string) (bool, error) {
	// TODO: Implement, for not this is just returning
	// that the resource is remote and needs to be pulled.
	return true, nil
}
