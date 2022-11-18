package operator

import (
	"context"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BundleValidateCheck", func() {
	var bundleValidateCheck ValidateOperatorBundleCheck

	BeforeEach(func() {
		bundleValidateCheck = *NewValidateOperatorBundleCheck()
	})

	AssertMetaData(&bundleValidateCheck)

	// TODO: Add more tests and bundles to testdata/ that excecise each of the
	// validations that we use.
	Describe("Operator Bundle Validate", func() {
		Context("When Operator Bundle Validate passes", func() {
			It("Should pass Validate", func() {
				imageRef := certification.ImageReference{
					ImageFSPath: "./testdata/valid_bundle",
				}
				ok, err := bundleValidateCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When Operator Bundle Validate does not Pass", func() {
			It("Should not pass Validate", func() {
				imageRef := certification.ImageReference{
					ImageFSPath: "./testdata/invalid_bundle",
				}
				ok, err := bundleValidateCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
