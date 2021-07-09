package engine

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/shell"
)

var _ = Describe("TestPolicyEngine", func() {
	var (
		hasNoProhibitedCheck   certification.Check = &shell.HasNoProhibitedPackagesCheck{}
		validateOperatorBundle certification.Check = &shell.ValidateOperatorBundlePolicy{}
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
