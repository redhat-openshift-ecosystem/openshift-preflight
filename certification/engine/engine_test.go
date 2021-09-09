package engine

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	containerpol "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/policy/container"
	operatorpol "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/policy/operator"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
)

var _ = Describe("TestPolicyEngine", func() {
	var (
		hasNoProhibitedCheck   certification.Check = &containerpol.HasNoProhibitedPackagesCheck{}
		validateOperatorBundle certification.Check = &operatorpol.ValidateOperatorBundleCheck{}
	)

	Describe("Querying all policies", func() {
		Context("When it is a container policy", func() {
			It("should return the check", func() {
				check := queryChecks(hasNoProhibitedCheck.Name())
				Expect(check.Name()).To(Equal(hasNoProhibitedCheck.Name()))
			})
		})
		Context("When it is an operator policy", func() {
			It("should return the check", func() {
				check := queryChecks(validateOperatorBundle.Name())
				Expect(check.Name()).To(Equal(validateOperatorBundle.Name()))
			})
		})
		Context("When it is an invalid check name", func() {
			It("should return nil", func() {
				check := queryChecks("abc")
				Expect(check).To(BeNil())
			})
		})
	})
})

var _ = Describe("Engine Creation", func() {
	Describe("When getting a new engine for a configuration", func() {
		Context("with a valid configuration", func() {
			cfg := runtime.Config{
				Image:          "dummy/image",
				EnabledChecks:  ContainerPolicy(),
				ResponseFormat: "json",
			}

			It("should return an engine and no error", func() {
				engine, err := NewForConfig(cfg)
				Expect(err).ToNot(HaveOccurred())
				Expect(engine).ToNot(BeNil())
			})
		})

		Context("with a configuration that has no checks", func() {
			cfg := runtime.Config{
				Image:          "dummy/image",
				EnabledChecks:  []string{},
				ResponseFormat: "json",
			}

			It("should return an error indicating no checks were provided", func() {
				engine, err := NewForConfig(cfg)
				Expect(err).To(HaveOccurred())
				Expect(engine).To(BeNil())
			})
		})

		Context("with a configuration that has an unknown check", func() {
			cfg := runtime.Config{
				Image:          "dummy/image",
				EnabledChecks:  []string{"UnknownCheck"},
				ResponseFormat: "json",
			}

			It("should return an error indicating no checks were provided", func() {
				engine, err := NewForConfig(cfg)
				Expect(err).To(HaveOccurred())
				Expect(engine).To(BeNil())
			})
		})
	})
})
