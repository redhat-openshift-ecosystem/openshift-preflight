package runtime

import (
	"os"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

var _ = Describe("Viper to Runtime Config", func() {
	var baseViperCfg *viper.Viper
	var expectedRuntimeCfg *Config
	BeforeEach(func() {
		baseViperCfg = viper.New()
		expectedRuntimeCfg = &Config{}

		if val, isSet := os.LookupEnv("KUBECONFIG"); isSet {
			DeferCleanup(os.Setenv, "KUBECONFIG", val)
			Expect(os.Unsetenv("KUBECONFIG")).To(Succeed())
		}
		baseViperCfg.Set("logfile", "logfile")
		expectedRuntimeCfg.LogFile = "logfile"
		baseViperCfg.Set("dockerConfig", "dockerConfig")
		expectedRuntimeCfg.DockerConfig = "dockerConfig"
		baseViperCfg.Set("artifacts", "artifacts")
		expectedRuntimeCfg.Artifacts = "artifacts"
		baseViperCfg.Set("junit", true)
		expectedRuntimeCfg.WriteJUnit = true

		baseViperCfg.Set("pyxis_api_token", "apitoken")
		expectedRuntimeCfg.PyxisAPIToken = "apitoken"
		baseViperCfg.Set("submit", true)
		expectedRuntimeCfg.Submit = true
		baseViperCfg.Set("pyxis_env", "prod")
		expectedRuntimeCfg.PyxisHost = "catalog.redhat.com/api/containers"
		baseViperCfg.Set("certification_project_id", "000000000000")
		expectedRuntimeCfg.CertificationProjectID = "000000000000"
		baseViperCfg.Set("platform", "s390x")
		expectedRuntimeCfg.Platform = "s390x"
		baseViperCfg.Set("insecure", true)
		expectedRuntimeCfg.Insecure = true

		baseViperCfg.Set("namespace", "myns")
		expectedRuntimeCfg.Namespace = "myns"
		baseViperCfg.Set("serviceaccount", "mysa")
		expectedRuntimeCfg.ServiceAccount = "mysa"
		baseViperCfg.Set("scorecard_image", "myscorecardimage")
		expectedRuntimeCfg.ScorecardImage = "myscorecardimage"
		baseViperCfg.Set("scorecard_wait_time", "100")
		expectedRuntimeCfg.ScorecardWaitTime = "100"
		baseViperCfg.Set("channel", "mychannel")
		expectedRuntimeCfg.Channel = "mychannel"
		baseViperCfg.Set("indeximage", "myindeximage")
		expectedRuntimeCfg.IndexImage = "myindeximage"
	})

	Context("With values in a viper config", func() {
		It("should populate a runtime.Config with container and operator policy values", func() {
			cfg, err := NewConfigFrom(*baseViperCfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(*cfg).To(BeEquivalentTo(*expectedRuntimeCfg))
		})
	})

	It("should only have 24 struct keys for tests to be valid", func() {
		// If this test fails, it means a developer has added or removed
		// keys from runtime.Config, and so these tests may no longer be
		// accurate in confirming that the derived configuration from viper
		// matches.
		keys := reflect.TypeOf(Config{}).NumField()
		Expect(keys).To(Equal(24))
	})
})
