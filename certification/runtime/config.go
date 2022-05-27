package runtime

import (
	"os"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/policy"
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
	// Operator-Specific Fields
	Namespace         string
	ServiceAccount    string
	ScorecardImage    string
	ScorecardWaitTime string
	Channel           string
	IndexImage        string
	Kubeconfig        string
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
	cfg.WriteJUnit = viper.GetBool("junit")
	cfg.storeContainerPolicyConfiguration(vcfg)
	cfg.storeOperatorPolicyConfiguration(vcfg)
	return &cfg, nil
}

// storeContainerPolicyConfiguration reads container-policy-specific config
// items in viper, normalizes them, and stores them in Config.
func (c *Config) storeContainerPolicyConfiguration(vcfg viper.Viper) {
	c.PyxisAPIToken = vcfg.GetString("pyxis_api_token")
	c.Submit = vcfg.GetBool("submit")
	c.PyxisHost = pyxisHostLookup(vcfg.GetString("pyxis_env"), vcfg.GetString("pyxis_host"))

	// Strip the ospid- prefix from the project ID if provided.
	certificationProjectID := vcfg.GetString("certification_project_id")
	if strings.HasPrefix(certificationProjectID, "ospid-") {
		certificationProjectID = strings.Split(certificationProjectID, "-")[1]
	}
	c.CertificationProjectID = certificationProjectID
}

// storeOperatorPolicyConfiguration reads operator-policy-specific config
// items in viper, normalizes them, and stores them in Config.
func (c *Config) storeOperatorPolicyConfiguration(vcfg viper.Viper) {
	c.Kubeconfig = os.Getenv("KUBECONFIG")
	c.Namespace = viper.GetString("namespace")
	c.ServiceAccount = viper.GetString("serviceaccount")
	c.ScorecardImage = viper.GetString("scorecard_image")
	c.ScorecardWaitTime = viper.GetString("scorecard_wait_time")
	c.Channel = viper.GetString("channel")
	c.IndexImage = viper.GetString("indeximage")
}
