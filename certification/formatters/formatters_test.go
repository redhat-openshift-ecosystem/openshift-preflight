package formatters

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
)

var _ = Describe("Formatters", func() {
	Describe("When getting a new formatter for a configuration", func() {
		Context("with a valid configuration", func() {
			cfg := runtime.Config{
				ResponseFormat: "json",
			}

			It("should return a formatter and no error", func() {
				formatter, err := NewForConfig(cfg)
				Expect(err).ToNot(HaveOccurred())
				Expect(formatter).ToNot(BeNil())
			})
		})

		Context("with an unknown format requested by the user", func() {
			cfg := runtime.Config{
				ResponseFormat: "unknownFormat",
			}

			It("should return an error", func() {
				formatter, err := NewForConfig(cfg)

				Expect(err).To(HaveOccurred())
				Expect(formatter).To(BeNil())
			})
		})
	})

	Describe("When creating a new generic formatter", func() {

		Context("with improper arguments", func() {
			expectedResult := []byte(fmt.Errorf("failed to create a new generic formatter: %w",
				errors.ErrFormatterNameNotProvided).Error())
			var fn FormatterFunc = func(runtime.Results) ([]byte, error) {
				return expectedResult, nil
			}

			emptyNameFormatter, err := New("", "txt", fn)
			It("should return an error because of an empty name", func() {
				Expect(err).To(HaveOccurred())
				Expect(emptyNameFormatter).To(BeNil())
			})
		})

		Context("with proper arguments", func() {
			var expectedResult []byte = []byte("this is a test")
			var name string = "testFormatter"
			var extension string = "txt"
			var fn FormatterFunc = func(runtime.Results) ([]byte, error) {
				return expectedResult, nil
			}

			formatter, err := New(name, extension, fn)
			It("should not return an error", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(formatter).ToNot(BeNil())
			})

			formattingResult, err := formatter.Format(runtime.Results{})
			It("should format results as expected", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(formattingResult).To(Equal(expectedResult))
			})

			It("should be identifiable as the provided name", func() {
				Expect(formatter.PrettyName()).To(Equal(name))
			})
		})
	})
})
