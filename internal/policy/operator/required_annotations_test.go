package operator

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RequiredAnnotations", func() {
	var h RequiredAnnotations
	BeforeEach(func() {
		h = RequiredAnnotations{}
	})

	AssertMetaData(h)

	When("Getting the CSV from a bundle", func() {
		It("Should fail if a CSV is not found", func() {
			_, err := h.getBundleCSV(context.TODO(), "./testdata/doesnotexist")
			Expect(err).To(HaveOccurred())
		})

		It("Should successfully find a CSV in a valid bundle", func() {
			csv, err := h.getBundleCSV(context.TODO(), "./testdata/disconnected_bundle")
			Expect(err).ToNot(HaveOccurred())
			Expect(csv.ObjectMeta.Name).ToNot(BeEmpty())
		})
	})

	When("Validating that a CSV has all of the required annotations", func() {
		var bundlepath string
		It("Should succeed with a bundle that has been prepared as expected", func() {
			bundlepath = "./testdata/required_annotations_bundle"
			passed, err := h.validate(context.TODO(), bundlepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(passed).To(BeTrue())
		})

		It("Should fail with a bundle that has not been prepared as expected", func() {
			bundlepath = "./testdata/invalid_bundle"
			passed, err := h.validate(context.TODO(), bundlepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(passed).To(BeFalse())
		})

		It("Should fail with a bundle that has not been prepared as expected, and is missing an entry", func() {
			bundlepath = "./testdata/incorrect_required_annotations_bundle"
			passed, err := h.validate(context.TODO(), bundlepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(passed).To(BeFalse())
		})
		It("Should fail with a bundle that has an incorrect optional value", func() {
			bundlepath = "./testdata/incorrect_optional_annotations_bundle"
			passed, err := h.validate(context.TODO(), bundlepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(passed).To(BeFalse())
		})
	})
})
