package shell

import (
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	containerutil "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/container"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	log "github.com/sirupsen/logrus"
)

type MountedCheckEngine struct {
	Image string
	Check certification.Check

	results runtime.Results
}

// ExecuteChecks runs all checks stored in the check engine.
func (e *MountedCheckEngine) ExecuteChecks() error {
	log.Info("target image: ", e.Image)

	e.results.TestedImage = e.Image
	targetImage := e.Image

	log.Info("running check: ", e.Check.Name())
	// We want to know the time just for the check itself, so reset checkStartTime
	checkStartTime := time.Now()
	checkPassed, err := containerutil.RunInsideImageFS(podmanEngine, targetImage, e.Check.Validate)
	checkElapsedTime := time.Since(checkStartTime)

	switch {
	case err != nil:
		log.WithFields(log.Fields{"result": "ERROR", "error": err.Error()}).Info("check completed: ", e.Check.Name())
		e.results.Errors = append(e.results.Errors, runtime.Result{Check: e.Check, ElapsedTime: checkElapsedTime})
	case !checkPassed:
		log.WithFields(log.Fields{"result": "FAILED"}).Info("check completed: ", e.Check.Name())
		e.results.Failed = append(e.results.Failed, runtime.Result{Check: e.Check, ElapsedTime: checkElapsedTime})
	default:
		log.WithFields(log.Fields{"result": "PASSED"}).Info("check completed: ", e.Check.Name())
		e.results.Passed = append(e.results.Passed, runtime.Result{Check: e.Check, ElapsedTime: checkElapsedTime})
	}

	// 2 possible status codes
	// 1. PASSED - all checks have passed successfully
	// 2. FAILED - At least one check failed or an error occured in one of the checks
	if len(e.results.Errors) > 0 || len(e.results.Failed) > 0 {
		e.results.PassedOverall = false
	} else {
		e.results.PassedOverall = true
	}

	return nil
}

// Results will return the results of check execution.
func (e *MountedCheckEngine) Results() runtime.Results {
	return e.results
}
