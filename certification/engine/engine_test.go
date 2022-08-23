package engine

import (
	"context"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Engine Creation", func() {
	When("getting a new engine for a configuration", func() {
		Context("with a valid configurations", func() {
			Context("for the container policy", func() {
				cfg := runtime.Config{
					Image:          "dummy/image",
					Policy:         policy.PolicyContainer,
					ResponseFormat: "json",
				}

				It("should return an engine and no error", func() {
					engine, err := NewForConfig(context.TODO(), cfg.ReadOnly())
					Expect(err).ToNot(HaveOccurred())
					Expect(engine).ToNot(BeNil())
				})
				It("should return the correct names", func() {
					names := ContainerPolicy(context.TODO())
					Expect(names).To(ContainElements([]string{
						"HasLicense",
						"HasUniqueTag",
						"LayerCountAcceptable",
						"HasNoProhibitedPackages",
						"HasRequiredLabel",
						"RunAsNonRoot",
						"HasModifiedFiles",
						"BasedOnUbi",
					}))
				})
			})
			Context("for the operator policy", func() {
				cfg := runtime.Config{
					Image:          "dummy/image",
					Policy:         policy.PolicyOperator,
					ResponseFormat: "json",
				}
				It("should return an engine and no error", func() {
					engine, err := NewForConfig(context.TODO(), cfg.ReadOnly())
					Expect(err).ToNot(HaveOccurred())
					Expect(engine).ToNot(BeNil())
				})
				It("should return the correct names", func() {
					names := OperatorPolicy(context.TODO())
					Expect(names).To(ContainElements([]string{
						"ScorecardBasicSpecCheck",
						"ScorecardOlmSuiteCheck",
						"DeployableByOLM",
						"ValidateOperatorBundle",
					}))
				})
			})

			Context("for the scratch policy", func() {
				cfg := runtime.Config{
					Image:          "dummy/image",
					Policy:         policy.PolicyScratch,
					ResponseFormat: "json",
				}

				It("should return an engine and no error", func() {
					engine, err := NewForConfig(context.TODO(), cfg.ReadOnly())
					Expect(err).ToNot(HaveOccurred())
					Expect(engine).ToNot(BeNil())
				})
				It("should return the correct names", func() {
					names := ScratchContainerPolicy(context.TODO())
					Expect(names).To(ContainElements([]string{
						"HasLicense",
						"HasUniqueTag",
						"LayerCountAcceptable",
						"HasRequiredLabel",
						"RunAsNonRoot",
					}))
				})
			})

			Context("for the Root policy", func() {
				cfg := runtime.Config{
					Image:          "dummy/image",
					Policy:         policy.PolicyRoot,
					ResponseFormat: "json",
				}

				It("should return an engine and no error", func() {
					engine, err := NewForConfig(context.TODO(), cfg.ReadOnly())
					Expect(err).ToNot(HaveOccurred())
					Expect(engine).ToNot(BeNil())
				})
				It("should return the correct names", func() {
					names := RootExceptionContainerPolicy(context.TODO())
					Expect(names).To(ContainElements([]string{
						"HasLicense",
						"HasUniqueTag",
						"LayerCountAcceptable",
						"HasNoProhibitedPackages",
						"HasRequiredLabel",
						"HasModifiedFiles",
					}))
				})
			})
		})

		Context("with an invalid policy", func() {
			cfg := runtime.Config{
				Image:          "dummy/image",
				Policy:         "invalid",
				ResponseFormat: "json",
			}

			It("should return an error and no engine", func() {
				engine, err := NewForConfig(context.TODO(), cfg.ReadOnly())
				Expect(err).To(HaveOccurred())
				Expect(engine).To(BeNil())
			})
		})
	})
})
