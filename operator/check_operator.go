package operator

import (
	"context"
	"fmt"
	goruntime "runtime"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	preflighterr "github.com/redhat-openshift-ecosystem/openshift-preflight/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"
)

type Option = func(*operatorCheck)

// NewCheck is a check runner that executes the Operator Policy.
func NewCheck(image, indeximage string, kubeconfig []byte, opts ...Option) *operatorCheck {
	c := &operatorCheck{
		image:               image,
		kubeconfig:          kubeconfig,
		indeximage:          indeximage,
		csvTimeout:          runtime.DefaultCSVTimeout,
		subscriptionTimeout: runtime.DefaultSubscriptionTimeout,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Run executes the check and returns the results.
func (c operatorCheck) Run(ctx context.Context) (certification.Results, error) {
	err := c.resolve(ctx)
	if err != nil {
		return certification.Results{}, err
	}

	cfg := runtime.Config{
		//coverage:ignore
		Image:        c.image,
		DockerConfig: c.dockerConfigFilePath,
		Scratch:      true,
		Bundle:       true,
		Insecure:     c.insecure,
		Platform:     goruntime.GOARCH,
	}
	eng, err := engine.New(ctx, c.checks, c.kubeconfig, cfg)
	if err != nil {
		//coverage:ignore
		return certification.Results{}, err
	}

	// NOTE(): The engine reads the cluster's version, but requires the KUBECONFIG
	// environment variable to do it. Ultimately, the call should be refactored to remove the
	// requirement, and be made here (unrelated to the engine). With that said, for now
	// this is being left as is because the values aren't currently added to results.
	//
	// See: https://github.com/redhat-openshift-ecosystem/openshift-preflight/pull/322

	//coverage:ignore
	if err := eng.ExecuteChecks(ctx); err != nil {
		//coverage:ignore
		return certification.Results{}, err
	}

	//coverage:ignore
	return eng.Results(ctx), nil
}

func (c *operatorCheck) resolve(ctx context.Context) error {
	if c.resolved {
		return nil
	}

	switch {
	case c.image == "":
		return preflighterr.ErrImageEmpty
	case c.kubeconfig == nil:
		return preflighterr.ErrKubeconfigEmpty
	case c.indeximage == "":
		return preflighterr.ErrIndexImageEmpty
	}

	c.policy = policy.PolicyOperator
	newChecks, err := engine.InitializeOperatorChecks(ctx, c.policy, engine.OperatorCheckConfig{
		IndexImage:          c.indeximage,
		DockerConfig:        c.dockerConfigFilePath,
		Channel:             c.operatorChannel,
		Kubeconfig:          c.kubeconfig,
		CSVTimeout:          c.csvTimeout,
		SubscriptionTimeout: c.subscriptionTimeout,
	})
	if err != nil {
		//coverage:ignore
		return fmt.Errorf("%w: %s", preflighterr.ErrCannotInitializeChecks, err)
	}
	c.checks = newChecks
	c.resolved = true

	return nil
}

// List the available operator checks.
func (c operatorCheck) List(ctx context.Context) (policy.Policy, []check.Check, error) {
	return c.policy, c.checks, c.resolve(ctx)
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

// WithInsecureConnection allows for preflight to connect to an insecure registry
// to pull images.
func WithInsecureConnection() Option {
	return func(oc *operatorCheck) {
		oc.insecure = true
	}
}

// WithCSVTimeout customizes how long to wait for a ClusterServiceVersion to become healthy.
func WithCSVTimeout(csvTimeout time.Duration) Option {
	//coverage:ignore
	return func(oc *operatorCheck) {
		//coverage:ignore
		oc.csvTimeout = csvTimeout
	}
}

// WithSubscriptionTimeout customizes how long to wait for a subscription to become healthy.
func WithSubscriptionTimeout(subscriptionTimeout time.Duration) Option {
	//coverage:ignore
	return func(oc *operatorCheck) {
		//coverage:ignore
		oc.subscriptionTimeout = subscriptionTimeout
	}
}

type operatorCheck struct {
	// required
	image      string
	kubeconfig []byte
	indeximage string
	// optional
	operatorChannel      string
	dockerConfigFilePath string
	insecure             bool
	checks               []check.Check
	resolved             bool
	policy               policy.Policy
	csvTimeout           time.Duration
	subscriptionTimeout  time.Duration
}
