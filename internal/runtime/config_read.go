package runtime

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/config"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"
)

// ensure ReadOnlyConfig always implements certification.Configurable
var _ config.Config = &ReadOnlyConfig{}

// ReadOnlyConfig is a Config that cannot be modified. It implements
// certification.Configurable.
type ReadOnlyConfig struct {
	cfg Config
}

func (ro *ReadOnlyConfig) Image() string {
	return ro.cfg.Image
}

func (ro *ReadOnlyConfig) Policy() policy.Policy {
	return ro.cfg.Policy
}

func (ro *ReadOnlyConfig) ResponseFormat() string {
	return ro.cfg.ResponseFormat
}

func (ro *ReadOnlyConfig) IsBundle() bool {
	return ro.cfg.Bundle
}

func (ro *ReadOnlyConfig) IsScratch() bool {
	return ro.cfg.Scratch
}

func (ro *ReadOnlyConfig) LogFile() string {
	return ro.cfg.LogFile
}

func (ro *ReadOnlyConfig) CertificationProjectID() string {
	return ro.cfg.CertificationProjectID
}

func (ro *ReadOnlyConfig) PyxisHost() string {
	return ro.cfg.PyxisHost
}

func (ro *ReadOnlyConfig) PyxisAPIToken() string {
	return ro.cfg.PyxisAPIToken
}

func (ro *ReadOnlyConfig) DockerConfig() string {
	return ro.cfg.DockerConfig
}

func (ro *ReadOnlyConfig) Submit() bool {
	return ro.cfg.Submit
}

func (ro *ReadOnlyConfig) Namespace() string {
	return ro.cfg.Namespace
}

func (ro *ReadOnlyConfig) ServiceAccount() string {
	return ro.cfg.ServiceAccount
}

func (ro *ReadOnlyConfig) ScorecardImage() string {
	return ro.cfg.ScorecardImage
}

func (ro *ReadOnlyConfig) ScorecardWaitTime() string {
	return ro.cfg.ScorecardWaitTime
}

func (ro *ReadOnlyConfig) Channel() string {
	return ro.cfg.Channel
}

func (ro *ReadOnlyConfig) Artifacts() string {
	return ro.cfg.Artifacts
}

func (ro *ReadOnlyConfig) WriteJUnit() bool {
	return ro.cfg.WriteJUnit
}

func (ro *ReadOnlyConfig) Kubeconfig() string {
	return ro.cfg.Kubeconfig
}

func (ro *ReadOnlyConfig) IndexImage() string {
	return ro.cfg.IndexImage
}

func (ro *ReadOnlyConfig) Platform() string {
	return ro.cfg.Platform
}

func (ro *ReadOnlyConfig) Insecure() bool {
	return ro.cfg.Insecure
}
