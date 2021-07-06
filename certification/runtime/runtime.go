package runtime

import (
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"
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

type UserResponse struct {
	Image             string                 `json:"image" xml:"image"`
	ValidationVersion version.VersionContext `json:"validation_lib_version" xml:"validationLibVersion"`
	Results           UserResponseText       `json:"results" xml:"results"`
}

type UserResponseText struct {
	Passed []certification.Metadata
	Failed []certification.CheckInfo
	Errors []certification.HelpText
}
