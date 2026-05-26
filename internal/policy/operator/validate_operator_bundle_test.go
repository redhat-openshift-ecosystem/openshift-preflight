package operator

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	test "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/test"
)

var _ = Describe("BundleValidateCheck", func() {
	var bundleValidateCheck ValidateOperatorBundleCheck
	var ctx context.Context

	BeforeEach(func() {
		bundleValidateCheck = *NewValidateOperatorBundleCheck()
		ctx = test.NewTestLoggerContext(context.TODO())
	})

	AssertMetaData(&bundleValidateCheck)

	// TODO: Add more tests and bundles to testdata/ that exercise each of the
	// validations that we use.
	Describe("Operator Bundle Validate", func() {
		Context("When Operator Bundle Validate passes", func() {
			It("Should pass Validate", func() {
				imageRef := image.ImageReference{
					ImageFSPath: "./testdata/all_namespaces",
				}
				ok, err := bundleValidateCheck.Validate(ctx, imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When Operator Bundle Validate does not Pass", func() {
			It("Should not pass Validate", func() {
				imageRef := image.ImageReference{
					ImageFSPath: "./testdata/invalid_bundle",
				}
				ok, err := bundleValidateCheck.Validate(ctx, imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
