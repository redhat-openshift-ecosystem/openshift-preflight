package lib

import (
	"context"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Lib Operator Functions", func() {
	BeforeEach(createAndCleanupDirForArtifactsAndLogs)

	Context("When instantiating a CheckOperatorRunner", func() {
		var cfg *runtime.Config
		BeforeEach(func() {
			cfg = &runtime.Config{
				Image:          "quay.io/example/foo:latest",
				ResponseFormat: formatters.DefaultFormat,
			}
		})

		Context("with a valid policy formatter", func() {
			It("should return with no error, and the appropriate formatter", func() {
				cfg.ResponseFormat = "xml"
				runner, err := NewCheckOperatorRunner(context.TODO(), cfg)
				Expect(err).ToNot(HaveOccurred())
				expectedFormatter, err := formatters.NewByName(cfg.ResponseFormat)
				Expect(err).ToNot(HaveOccurred())
				Expect(runner.Formatter.PrettyName()).To(Equal(expectedFormatter.PrettyName()))
			})
		})

		Context("with an invalid policy formatter", func() {
			It("should return an error", func() {
				cfg.ResponseFormat = "foo"
				_, err := NewCheckOperatorRunner(context.TODO(), cfg)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with an invalid policy definition", func() {
			It("should return the container policy engine anyway", func() {
				cfg.Policy = "badpolicy"
				beforeCfg := *cfg
				runner, err := NewCheckOperatorRunner(context.TODO(), cfg)
				Expect(err).ToNot(HaveOccurred())

				_, err = engine.NewForConfig(context.TODO(), cfg.ReadOnly())
				Expect(runner.Cfg.Policy).ToNot(Equal(beforeCfg.Policy))
				Expect(runner.Cfg.Policy).To(Equal(policy.PolicyOperator))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with an invalid formatter definition", func() {
			It("should return an error", func() {
				cfg.ResponseFormat = "foo"
				_, err := NewCheckOperatorRunner(context.TODO(), cfg)
				Expect(err).To(HaveOccurred())
			})
		})

		It("should contain a ResultWriterFile ResultWriter", func() {
			runner, err := NewCheckOperatorRunner(context.TODO(), cfg)
			Expect(err).ToNot(HaveOccurred())
			_, rwIsExpectedType := runner.Rw.(*runtime.ResultWriterFile)
			Expect(rwIsExpectedType).To(BeTrue())
		})
	})
})
