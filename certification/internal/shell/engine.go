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

// CheckEngine implements a CheckRunner.
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
			log.WithFields(log.Fields{"result": "ERROR", "error": err.Error()}).Info("check completed: ", check.Name())
			e.results.Errors = append(e.results.Errors, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		if !passed {
			log.WithFields(log.Fields{"result": "FAILED"}).Info("check completed: ", check.Name())
			e.results.Failed = append(e.results.Failed, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		log.WithFields(log.Fields{"result": "PASSED"}).Info("check completed: ", check.Name())
		e.results.Passed = append(e.results.Passed, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
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
