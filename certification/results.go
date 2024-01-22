package certification

import (
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"
)

type openshiftClusterVersion = runtime.OpenshiftClusterVersion

type Result struct {
	check.Check
	ElapsedTime time.Duration
	// Err contains the error a check itself throws if it failed to run.
	// If populated, the expectation is that this Result is in the
	// Results{}.Errors slice.
	err error
}

type Results struct {
	TestedImage       string
	PassedOverall     bool
	TestedOn          openshiftClusterVersion
	CertificationHash string
	Passed            []Result
	Failed            []Result
	Errors            []Result
	Warned            []Result
}

func (r Result) Error() error {
	return r.err
}

func (r *Result) WithError(err error) *Result {
	r.err = err
	return r
}
