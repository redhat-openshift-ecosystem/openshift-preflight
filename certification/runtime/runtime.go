package runtime

import (
	"github.com/komish/preflight/certification"
	"github.com/komish/preflight/version"
)

type Config struct {
	Image           string
	EnabledPolicies []string
	ResponseFormat  string
}

type policyRunner struct {
	Image    string
	Policies []certification.Policy
	Results  Results
}

type Results struct {
	TestedImage string
	Passed      []certification.Policy
	Failed      []certification.Policy
	Errors      []certification.Policy
}

type UserResponse struct {
	Image             string                 `json:"image" xml:"image"`
	ValidationVersion version.VersionContext `json:"validation_lib_version" xml:"validationLibVersion"`
	Results           UserResponseText       `json:"results" xml:"results"`
}

type UserResponseText struct {
	Passed []certification.Metadata
	Failed []certification.PolicyInfo
	Errors []certification.HelpText
	// TODO: Errors does not actually include any error information
	// and it needs to do so.
}
