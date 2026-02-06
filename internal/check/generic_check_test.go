package check

import (
	"context"
	"errors"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Generic check tests", func() {
	validatorFn := func(ctx context.Context, imageRef image.ImageReference) (bool, error) { //nolint:unparam // ctx param is unused
		switch imageRef.ImageFSPath {
		case "error":
			return false, errors.New("invalid validator")
		case "failed":
			return false, nil
		default:
			return true, nil
		}
	}
	metadata := Metadata{
		Description: "test metadata",
	}
	helpText := HelpText{
		Message: "test message",
	}
	When("A generic check is created", func() {
		var testCheck Check
		var imgRef image.ImageReference
		BeforeEach(func() {
			testCheck = NewGenericCheck(
				"testname",
				validatorFn,
				metadata,
				helpText,
				nil,
			)
			imgRef = image.ImageReference{}
		})
		It("should return the correct name", func() {
			Expect(testCheck.Name()).To(Equal("testname"))
		})
		It("should return the correct metadata", func() {
			Expect(testCheck.Metadata().Description).To(Equal("test metadata"))
		})
		It("should return the correct helpText", func() {
			Expect(testCheck.Help().Message).To(Equal("test message"))
		})
		It("should execute the validator successfully", func() {
			result, err := testCheck.Validate(context.TODO(), imgRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})
		Context("but an error occurs", func() {
			It("should return an error and false for result", func() {
				imgRef.ImageFSPath = "error"
				result, err := testCheck.Validate(context.TODO(), imgRef)
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeFalse())
			})
		})
		Context("but an failure occurs", func() {
			It("should not return an error and false for result", func() {
				imgRef.ImageFSPath = "failed"
				result, err := testCheck.Validate(context.TODO(), imgRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeFalse())
			})
		})
	})
})
