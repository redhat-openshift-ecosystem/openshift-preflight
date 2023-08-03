package operator

import (
	"context"
	"fmt"
	goruntime "runtime"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	preflighterr "github.com/redhat-openshift-ecosystem/openshift-preflight/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"
)

type Option = func(*operatorCheck)

// TODO(): replace this value when the default in package cmd is moved to a central location
const defaultScorecardWaitTime = "240"

// NewCheck is a check runner that executes the Operator Policy.
func NewCheck(image, indeximage string, kubeconfig []byte, opts ...Option) *operatorCheck {
	c := &operatorCheck{
		image:             image,
		kubeconfig:        kubeconfig,
		indeximage:        indeximage,
		scorecardWaitTime: defaultScorecardWaitTime,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Run executes the check and returns the results.
func (c operatorCheck) Run(ctx context.Context) (certification.Results, error) {
	switch {
	case c.image == "":
		return certification.Results{}, preflighterr.ErrImageEmpty
	case c.kubeconfig == nil:
		return certification.Results{}, preflighterr.ErrKubeconfigEmpty
	case c.indeximage == "":
		return certification.Results{}, preflighterr.ErrIndexImageEmpty
	}

	pol := policy.PolicyOperator

	checks, err := engine.InitializeOperatorChecks(ctx, pol, engine.OperatorCheckConfig{
		ScorecardImage:          c.scorecardImage,
		ScorecardWaitTime:       c.scorecardWaitTime,
		ScorecardNamespace:      c.scorecardNamespace,
		ScorecardServiceAccount: c.scorecardServiceAccount,
		IndexImage:              c.indeximage,
		DockerConfig:            c.dockerConfigFilePath,
		Channel:                 c.operatorChannel,
		Kubeconfig:              c.kubeconfig,
	})
	if err != nil {
		return certification.Results{}, fmt.Errorf("%w: %s", preflighterr.ErrCannotInitializeChecks, err)
	}

	cfg := runtime.Config{
		Image:        c.image,
		DockerConfig: c.dockerConfigFilePath,
		Scratch:      true,
		Bundle:       true,
		Insecure:     c.insecure,
		Platform:     goruntime.GOARCH,
	}
	eng, err := engine.New(ctx, checks, c.kubeconfig, cfg)
	if err != nil {
		return certification.Results{}, err
	}

	// NOTE(): The engine reads the cluster's version, but requires the KUBECONFIG
	// environment variable to do it. Ultimately, the call should be refactored to remove the
	// requirement, and be made here (unrelated to the engine). With that said, for now
	// this is being left as is because the values aren't currently added to results.
	//
	// See: https://github.com/redhat-openshift-ecosystem/openshift-preflight/pull/322

	if err := eng.ExecuteChecks(ctx); err != nil {
		return certification.Results{}, err
	}

	if err != nil {
		return certification.Results{}, err
	}

	return eng.Results(ctx), nil
}

// WithScorecardNamespace configures the namespace value to use for OperatorSDK Scorecard checks.
func WithScorecardNamespace(ns string) Option {
	return func(oc *operatorCheck) {
		oc.scorecardNamespace = ns
	}
}

// WithOperatorChannel configures the operator value to use when attempting to deploy the
// operator under test.
func WithOperatorChannel(ch string) Option {
	return func(oc *operatorCheck) {
		oc.operatorChannel = ch
	}
}

// WithDockerConfigJSONFromFile is a path to credentials necessary to pull the image under tests.
func WithDockerConfigJSONFromFile(path string) Option {
	return func(oc *operatorCheck) {
		oc.dockerConfigFilePath = path
	}
}

// WithScorecardWaitTime overrides the wait time passed to OperatorSDK Scorecard-based checks
// The seconds value should be a string representation of a number of seconds without a suffix.
func WithScorecardWaitTime(seconds string) Option {
	return func(oc *operatorCheck) {
		oc.scorecardWaitTime = seconds
	}
}

// WithScorecardServiceAccount adjusts the service account used for OperatorSDK Scorecard-based
// checks.
func WithScorecardServiceAccount(sa string) Option {
	return func(oc *operatorCheck) {
		oc.scorecardServiceAccount = sa
	}
}

// WithScorecardImage overrides the Operator-SDK Scorecard image value. This
// option should ONLY be used in disconnected environments to overcome image
// accessibility restrictions.
//
// Most users should omit this option.
func WithScorecardImage(image string) Option {
	return func(oc *operatorCheck) {
		oc.scorecardImage = image
	}
}

// WithInsecureConnection allows for preflight to connect to an insecure registry
// to pull images.
func WithInsecureConnection() Option {
	return func(oc *operatorCheck) {
		oc.insecure = true
	}
}

type operatorCheck struct {
	// required
	image      string
	kubeconfig []byte
	indeximage string
	// optional
	scorecardImage          string
	scorecardNamespace      string
	scorecardServiceAccount string
	scorecardWaitTime       string
	operatorChannel         string
	dockerConfigFilePath    string
	insecure                bool
}
