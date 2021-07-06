package shell

import (
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/sirupsen/logrus"
)

type CheckEngine struct {
	Image  string
	Checks []certification.Check

	results      runtime.Results
	isDownloaded bool
}

// ExecuteChecks runs all checks stored in the check engine.
func (e *CheckEngine) ExecuteChecks(logger *logrus.Logger) {
	logger.Info("target image: ", e.Image)
	for _, check := range e.Checks {
		checkStartTime := time.Now()
		e.results.TestedImage = e.Image
		targetImage := e.Image

		// check if the image needs downloading
		if !e.isDownloaded {
			isRemote, err := e.ContainerIsRemote(e.Image, logger)
			if err != nil {
				logger.Error("unable to determine if the image was remote: ", err)
				e.results.Errors = append(e.results.Errors, runtime.Result{Check: check, ElapsedTime: time.Since(checkStartTime)})
				continue
			}

			if isRemote {
				logger.Info("downloading image")
				err := GetContainerFromRegistry(e.Image)
				if err != nil {
					logger.Error("unable to the container from the registry: ", err)
					e.results.Errors = append(e.results.Errors, runtime.Result{Check: check, ElapsedTime: time.Since(checkStartTime)})
					continue
				}
				e.isDownloaded = true
			}
		}

		// if we downloaded an image to disk, lets test against that.
		// COMMENTED: tests aren't currently written to support this
		// remove if we decide we do not care to have a tarball.
		// if len(e.localImagePath) != 0 {
		// 	targetImage = e.localImagePath
		// }

		logger.Info("running check: ", check.Name())
		// We want to know the time just for the check itself, so reset checkStartTime
		checkStartTime = time.Now()

		// run the validation
		passed, err := check.Validate(targetImage, logger)

		checkElapsedTime := time.Since(checkStartTime)

		if err != nil {
			logger.WithFields(logrus.Fields{"result": "ERROR", "error": err.Error()}).Info("check completed: ", check.Name())
			e.results.Errors = append(e.results.Errors, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		if !passed {
			logger.WithFields(logrus.Fields{"result": "FAILED"}).Info("check completed: ", check.Name())
			e.results.Failed = append(e.results.Failed, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
			continue
		}

		logger.WithFields(logrus.Fields{"result": "PASSED"}).Info("check completed: ", check.Name())
		e.results.Passed = append(e.results.Passed, runtime.Result{Check: check, ElapsedTime: checkElapsedTime})
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

func (e *CheckEngine) ContainerIsRemote(path string, logger *logrus.Logger) (bool, error) {
	// TODO: Implement, for not this is just returning
	// that the resource is remote and needs to be pulled.
	return true, nil
}
