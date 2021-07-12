package shell

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ScorecardOlmSuiteCheck", func() {

	var (
		scorecardOlmSuiteCheck ScorecardOlmSuiteCheck
	)

	Describe("Operator Bundle Scorecard use a good operator bundle image", func() {
		Context("When Operator Bundle Scorecard Check is run", func() {
			It("Expect to Pass", func() {
				ok, err := scorecardOlmSuiteCheck.Validate("quay.io/rocrisp/preflight-operator-bundle:v1")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
	})

	Describe("Operator Bundle Scorecard use a container image", func() {
		Context("When Operator Bundle Scorecard Check is run", func() {
			It("Expect to Fail", func() {
				ok, err := scorecardOlmSuiteCheck.Validate("quay.io/rocrisp/preflight:v1")
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})