package runtime

import (
	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/version"
)

type Config struct {
	Image          string
	EnabledChecks  []string
	ResponseFormat string
}
type Results struct {
	TestedImage string
	Passed      []certification.Check
	Failed      []certification.Check
	Errors      []certification.Check
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
