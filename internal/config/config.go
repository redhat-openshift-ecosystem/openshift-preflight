package config

import "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"

// Config is a read-only preflight configuration.
type Config interface {
	commonConfig
	containerConfig
	operatorConfig
}

// commonConfig contains configurables common
// to all certification.
type commonConfig interface {
	Image() string
	Policy() policy.Policy
	ResponseFormat() string
	LogFile() string
	Artifacts() string
	WriteJUnit() bool
	DockerConfig() string
}

// containerConfig are configurables relevant to
// container certification.
type containerConfig interface {
	IsScratch() bool
	CertificationProjectID() string
	PyxisHost() string
	PyxisAPIToken() string
	Submit() bool
	Platform() string
	Insecure() bool
}

// operatorConfig are configurables relevant to
// operator certification.
type operatorConfig interface {
	IsBundle() bool
	Namespace() string
	ServiceAccount() string
	ScorecardImage() string
	ScorecardWaitTime() string
	Channel() string
	Kubeconfig() string
	IndexImage() string
}
