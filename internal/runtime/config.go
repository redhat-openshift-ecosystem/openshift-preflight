package runtime

import (
	"os"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/option"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"

	"github.com/spf13/viper"
)

// Config contains configuration details for running preflight.
type Config struct {
	Image          string
	Policy         policy.Policy
	ResponseFormat string
	Bundle         bool
	Scratch        bool
	LogFile        string
	Artifacts      string
	WriteJUnit     bool
	// Container-Specific Fields
	CertificationProjectID string
	PyxisHost              string
	PyxisAPIToken          string
	DockerConfig           string
	Submit                 bool
	Platform               string
	Insecure               bool
	Offline                bool
	ManifestListDigest     string
	Konflux                bool
	// Operator-Specific Fields
	Namespace           string
	ServiceAccount      string
	ScorecardImage      string
	ScorecardWaitTime   string
	Channel             string
	IndexImage          string
	Kubeconfig          string
	CSVTimeout          time.Duration
	SubscriptionTimeout time.Duration
}

// ReadOnly returns an uneditably configuration.
func (c *Config) ReadOnly() *ReadOnlyConfig {
	return &ReadOnlyConfig{
		cfg: *c,
	}
}

// NewConfigFrom will return a runtime.Config based on the stored inputs in
// the provided viper.Viper. Note that shared configuration should be set
// in this function, and not in policy-specific functions. Defaults, should
// also be set after this function has been called.
func NewConfigFrom(vcfg viper.Viper) (*Config, error) {
	cfg := Config{}
	cfg.LogFile = vcfg.GetString("logfile")
	cfg.DockerConfig = vcfg.GetString("dockerConfig")
	cfg.Artifacts = vcfg.GetString("artifacts")
	cfg.WriteJUnit = vcfg.GetBool("junit")
	cfg.storeContainerPolicyConfiguration(vcfg)
	cfg.storeOperatorPolicyConfiguration(vcfg)
	return &cfg, nil
}

// storeContainerPolicyConfiguration reads container-policy-specific config
// items in viper, normalizes them, and stores them in Config.
func (c *Config) storeContainerPolicyConfiguration(vcfg viper.Viper) {
	c.PyxisAPIToken = vcfg.GetString("pyxis_api_token")
	c.Submit = vcfg.GetBool("submit")
	c.PyxisHost = PyxisHostLookup(vcfg.GetString("pyxis_env"), vcfg.GetString("pyxis_host"))
	c.CertificationProjectID = vcfg.GetString("certification_project_id")
	c.Platform = vcfg.GetString("platform")
	c.Insecure = vcfg.GetBool("insecure")
	c.Offline = vcfg.GetBool("offline")
	// todo should we have a flag for this? trying to hide this option as much as possible
	c.Konflux = vcfg.GetBool("konflux")
}

// storeOperatorPolicyConfiguration reads operator-policy-specific config
// items in viper, normalizes them, and stores them in Config.
func (c *Config) storeOperatorPolicyConfiguration(vcfg viper.Viper) {
	c.Kubeconfig = os.Getenv("KUBECONFIG")
	c.Namespace = vcfg.GetString("namespace")
	c.ServiceAccount = vcfg.GetString("serviceaccount")
	c.ScorecardImage = vcfg.GetString("scorecard_image")
	c.ScorecardWaitTime = vcfg.GetString("scorecard_wait_time")
	c.Channel = vcfg.GetString("channel")
	c.IndexImage = vcfg.GetString("indeximage")
	c.CSVTimeout = vcfg.GetDuration("csv_timeout")
	c.SubscriptionTimeout = vcfg.GetDuration("subscription_timeout")
}

// This is to satisfy the CraneConfig interface
func (c *Config) CraneDockerConfig() string {
	return c.DockerConfig
}

func (c *Config) CranePlatform() string {
	return c.Platform
}

func (c *Config) CraneInsecure() bool {
	return c.Insecure
}

var _ option.CraneConfig = &Config{}
