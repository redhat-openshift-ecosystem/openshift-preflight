package shell

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

var _ = Describe("BundlePackageNameUniqCheck", func() {
	var (
		bundlePackageNameUniqCheck ValidateOperatorPkNameUniqCheck
	)

	Describe("Operator Bundle Package Name Uniqueness Validate", func() {
		Context("When Operator Bundle Package Name Uniqueness Validate passes", func() {
			It("Should pass Validate", func() {
				ok, err := bundlePackageNameUniqCheck.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When Operator Bundle Package Name Uniqueness Validate does not Pass", func() {
			It("Should not pass Validate", func() {
				ok, err := bundlePackageNameUniqCheck.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
