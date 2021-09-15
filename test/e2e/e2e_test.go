package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
)

// These tests confirm that all container and operator policies properly pass
// with a known-good image, and properly fail with a known-bad image.
// Any check that is found in the error section of the Result will cause this
// to fail.
var _ = Describe("policy validation", func() {
	Describe("When enforcing operator policy", func() {
		var (
			// TODO: replace the failure case with crafted examples.
			goodImage = "quay.io/opdev/simple-demo-operator-bundle:v0.0.2"
			badImage  = "quay.io/komish/preflight-test-bundle-fails:latest"
		)

		Context("with a known-good image", func() {

			cfg := runtime.Config{
				Image:         goodImage,
				EnabledChecks: engine.OperatorPolicy(),
			}

			engine, err := engine.NewForConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			engine.ExecuteChecks()
			results := engine.Results()

			It("should pass all checks", func() {
				Expect(len(results.Passed)).To(Equal(len(cfg.EnabledChecks)))
			})
		})

		Context("with a known-bad image", func() {
			cfg := runtime.Config{
				Image:         badImage,
				EnabledChecks: engine.OperatorPolicy(),
			}

			engine, err := engine.NewForConfig(cfg)
			Expect(err).To(BeNil())

			engine.ExecuteChecks()
			results := engine.Results()

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
				Image:         goodImage,
				EnabledChecks: engine.ContainerPolicy(),
			}

			engine, err := engine.NewForConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			engine.ExecuteChecks()
			results := engine.Results()

			It("should pass all checks", func() {
				Expect(len(results.Passed)).To(Equal(len(cfg.EnabledChecks)))
				Expect(len(results.Errors)).To(BeZero())
				Expect(len(results.Failed)).To(BeZero())
			})
		})

		// check temporarily disabled: currently unable to find a container image to use
		// for the container-fails.Dockerfile that would fail both the HasMinimalVulnerabilitiesCheck
		// check in addition to all other container checks.
		XContext("with a known-bad image", func() {
			cfg := runtime.Config{
				Image:         badImage,
				EnabledChecks: engine.ContainerPolicy(),
			}

			engine, err := engine.NewForConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			engine.ExecuteChecks()
			results := engine.Results()

			// TODO: Replace this check so that you test for individual check failures
			It("should fail all checks", func() {
				Expect(len(results.Passed)).To(BeZero())
			})
		})
	})
})
