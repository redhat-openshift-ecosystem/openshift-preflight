package operator

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RestrictedNetworkGuidelines", func() {
	var ch FollowsRestrictedNetworkEnablementGuidelines
	BeforeEach(func() {
		//nolint:staticcheck // I'm not sure why it flags this as unused.
		ch = FollowsRestrictedNetworkEnablementGuidelines{}
	})

	AssertMetaData(ch)

	When("Getting the CSV from a bundle", func() {
		It("Should fail if a CSV is not found", func() {
			_, err := ch.getBundleCSV(context.TODO(), "./testdata/doesnotexist")
			Expect(err).To(HaveOccurred())
		})

		It("Should successfully find a CSV in a valid bundle", func() {
			csv, err := ch.getBundleCSV(context.TODO(), "./testdata/disconnected_bundle")
			Expect(err).ToNot(HaveOccurred())
			Expect(csv.ObjectMeta.Name).ToNot(BeEmpty())
		})
	})

	When("Validating that a CSV has attempted to follow the restricted network readiness guidelines.", func() {
		var bundlepath string
		It("Should succeed with a bundle that has been prepared as expected", func() {
			bundlepath = "./testdata/disconnected_bundle"
			passed, err := ch.validate(context.TODO(), bundlepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(passed).To(BeTrue())
		})

		It("Should fail with a bundle that has not been prepared for disconnected environments", func() {
			bundlepath = "./testdata/invalid_bundle"
			passed, err := ch.validate(context.TODO(), bundlepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(passed).To(BeFalse())
		})
	})
})
