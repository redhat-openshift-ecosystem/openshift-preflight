package shell

import (
	"fmt"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	containerutil "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/container"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	log "github.com/sirupsen/logrus"
)

const (
	passed  string = "PASSED"
	failed  string = "FAILED"
	errored string = "ERROR"
)

// CheckEngine implements a CheckEngine.
type CheckEngine struct {
	Image  string
	Checks []certification.Check

	results      runtime.Results
	isDownloaded bool
}

// ExecuteChecks runs all checks stored in the check engine.
func (e *CheckEngine) ExecuteChecks() error {
	log.Info("target image: ", e.Image)
	// check if the image needs downloading
	if !e.isDownloaded {
		isRemote, err := e.ContainerIsRemote(e.Image)
		if err != nil {
			return fmt.Errorf("%w: %s", errors.ErrGetRemoteContainerFailed, err)
		}

		if isRemote {
			log.Info("downloading image")

			stdouterr, err := containerutil.GetContainerFromRegistry(podmanEngine, e.Image)
			if err != nil {
				return fmt.Errorf("%w: %s", err, stdouterr)
			}
			e.isDownloaded = true
		}
	}

	for _, check := range e.Checks {
		e.results.TestedImage = e.Image
		targetImage := e.Image

		log.Info("running check: ", check.Name())
		// We want to know the time just for the check itself, so reset checkStartTime
		checkStartTime := time.Now()

		// run the validation
		passed, err := check.Validate(targetImage)

		checkElapsedTime := time.Since(checkStartTime)

		if err != nil {
			log.WithFields(log.Fields{"result": err, errored: err.Error()}).Info("check completed: ", check.Name())
			e.results.Errors = append(e.results.Errors, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		if !passed {
			log.WithFields(log.Fields{"result": failed}).Info("check completed: ", check.Name())
			e.results.Failed = append(e.results.Failed, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		log.WithFields(log.Fields{"result": passed}).Info("check completed: ", check.Name())
		e.results.Passed = append(e.results.Passed, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
	}

	// 2 possible status codes
	// 1. PASSED - all checks have passed successfully
	// 2. FAILED - At least one check failed or an error occured in one of the checks
	if len(e.results.Errors) > 0 || len(e.results.Failed) > 0 {
		e.results.Status = failed
	} else {
		e.results.Status = passed
	}

	return nil
}

// Results will return the results of check execution.
func (e *CheckEngine) Results() runtime.Results {
	return e.results
}

func (e *CheckEngine) ContainerIsRemote(path string) (bool, error) {
	// TODO: Implement, for not this is just returning
	// that the resource is remote and needs to be pulled.
	return true, nil
}
