// Package runtime contains the structs and definitions consumed by Preflight at
// runtime.
package runtime

import (
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

type Config struct {
	Image          string
	EnabledChecks  []string
	ResponseFormat string
	Mounted        bool
	Bundle         bool
}

type Result struct {
	certification.Check
	ElapsedTime time.Duration
}

type Results struct {
	TestedImage   string
	PassedOverall bool
	BundleHash    string
	Passed        []Result
	Failed        []Result
	Errors        []Result
}
