package e2e

import (
	"context"
	"os"

	"github.com/redhat-openshift-ecosystem/preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/preflight/certification/policy"
	"github.com/redhat-openshift-ecosystem/preflight/certification/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// These tests confirm that all container and operator policies properly pass
// with a known-good image, and properly fail with a known-bad image.
// Any check that is found in the error section of the Result will cause this
// to fail.
var _ = Describe("policy validation", func() {
	BeforeEach(func() {
		tmpDir, err := os.MkdirTemp("", "artifacts-*")
		Expect(err).ToNot(HaveOccurred())

		artifacts.SetDir(tmpDir)
		DeferCleanup(os.RemoveAll, tmpDir)
		DeferCleanup(artifacts.Reset)
	})

	Describe("When enforcing operator policy", func() {
		var (
			// TODO: replace the failure case with crafted examples.
			goodImage = "quay.io/opdev/simple-demo-operator-bundle:v0.0.2"
			badImage  = "quay.io/komish/preflight-test-bundle-fails:latest"
		)

		Context("with a known-good image", func() {
			cfg := runtime.Config{
				Image:  goodImage,
				Policy: policy.PolicyOperator,
			}

			e, err := engine.NewForConfig(context.TODO(), cfg.ReadOnly())
			Expect(err).ToNot(HaveOccurred())

			ctx := context.TODO()
			Expect(e.ExecuteChecks(ctx)).To(Succeed())
			results := e.Results(ctx)

			It("should pass all checks", func() {
				Expect(len(results.Passed)).To(Equal(len(engine.OperatorPolicy(context.TODO()))))
			})
		})

		Context("with a known-bad image", func() {
			cfg := runtime.Config{
				Image:  badImage,
				Policy: policy.PolicyOperator,
			}

			e, err := engine.NewForConfig(context.TODO(), cfg.ReadOnly())
			Expect(err).To(BeNil())

			ctx := context.TODO()
			Expect(e.ExecuteChecks(ctx)).To(Succeed())
			results := e.Results(ctx)

			// TODO: Replace this check so that you test for individual check failures
			It("should not pass any checks", func() {
				Expect(len(results.Passed)).To(BeZero())
			})
		})
	})

	Describe("When enforcing container policy", func() {
		var (
			// TODO: replace the passing case with the container used by
			// https://github.com/opdev/simple-demo-operator
			goodImage = "quay.io/komish/preflight-test-container-passes:latest"
			badImage  = "quay.io/komish/preflight-test-container-fails:latest"
		)

		Context("with a known-good image", func() {
			cfg := runtime.Config{
				Image:  goodImage,
				Policy: policy.PolicyContainer,
			}

			e, err := engine.NewForConfig(context.TODO(), cfg.ReadOnly())
			Expect(err).ToNot(HaveOccurred())

			ctx := context.TODO()
			Expect(e.ExecuteChecks(ctx)).To(Succeed())
			results := e.Results(ctx)

			It("should pass all checks", func() {
				Expect(len(results.Passed)).To(Equal(len(engine.ContainerPolicy(context.TODO()))))
				Expect(len(results.Errors)).To(BeZero())
				Expect(len(results.Failed)).To(BeZero())
			})
		})

		// check temporarily disabled: currently unable to find a container image to use
		// for the container-fails.Dockerfile that would fail both the HasMinimalVulnerabilitiesCheck
		// check in addition to all other container checks.
		XContext("with a known-bad image", func() {
			cfg := runtime.Config{
				Image:  badImage,
				Policy: policy.PolicyContainer,
			}

			e, err := engine.NewForConfig(context.TODO(), cfg.ReadOnly())
			Expect(err).ToNot(HaveOccurred())

			ctx := context.TODO()
			Expect(e.ExecuteChecks(ctx)).To(Succeed())
			results := e.Results(ctx)

			// TODO: Replace this check so that you test for individual check failures
			It("should fail all checks", func() {
				Expect(len(results.Passed)).To(BeZero())
			})
		})
	})
})
