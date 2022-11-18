package formatters

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Formatters", func() {
	Describe("When getting the formatter for the named default format", func() {
		It("should never fail", func() {
			_, err := NewByName(DefaultFormat)
			Expect(err).ToNot(HaveOccurred())
		})
	})
	Describe("When getting a new formatter for a configuration", func() {
		Context("with a valid configuration", func() {
			cfg := runtime.Config{
				ResponseFormat: "json",
			}

			It("should return a formatter and no error", func() {
				formatter, err := NewForConfig(cfg.ReadOnly())
				Expect(err).ToNot(HaveOccurred())
				Expect(formatter).ToNot(BeNil())
			})
		})

		Context("with an unknown format requested by the user", func() {
			cfg := runtime.Config{
				ResponseFormat: "unknownFormat",
			}

			It("should return an error", func() {
				formatter, err := NewForConfig(cfg.ReadOnly())

				Expect(err).To(HaveOccurred())
				Expect(formatter).To(BeNil())
			})
		})
	})

	Describe("When creating a new generic formatter", func() {
		Context("with improper arguments", func() {
			expectedResult := []byte(fmt.Errorf("failed to create a new generic formatter: formatter name is required").Error())
			var fn formatters.FormatterFunc //nolint:gosimple // We want to be explicit here for clarity
			fn = func(context.Context, certification.Results) ([]byte, error) {
				return expectedResult, nil
			}

			emptyNameFormatter, err := New("", "txt", fn)
			It("should return an error because of an empty name", func() {
				Expect(err).To(HaveOccurred())
				Expect(emptyNameFormatter).To(BeNil())
			})
		})

		Context("with proper arguments", func() {
			expectedResult := []byte("this is a test")
			name := "testFormatter"
			extension := "txt"
			var fn formatters.FormatterFunc //nolint:gosimple // We want to be explicit here for clarity
			fn = func(context.Context, certification.Results) ([]byte, error) {
				return expectedResult, nil
			}

			formatter, err := New(name, extension, fn)
			It("should not return an error", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(formatter).ToNot(BeNil())
			})

			formattingResult, err := formatter.Format(context.TODO(), certification.Results{})
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
