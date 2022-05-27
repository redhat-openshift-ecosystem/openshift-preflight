package engine

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
)

var _ = Describe("Engine Creation", func() {
	Describe("When getting a new engine for a configuration", func() {
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
			})
		})
	})
})
