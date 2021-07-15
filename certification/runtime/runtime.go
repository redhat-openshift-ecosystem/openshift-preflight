package runtime

import (
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

type Config struct {
	Image          string
	EnabledChecks  []string
	ResponseFormat string
}

type Result struct {
	certification.Check
	ElapsedTime time.Duration
}

type Results struct {
	TestedImage string
	Passed      []Result
	Failed      []Result
	Errors      []Result
}
