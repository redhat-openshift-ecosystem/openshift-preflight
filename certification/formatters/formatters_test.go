package formatters_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cmd"
)

var _ = Describe("Formatters", func() {
	Describe("When getting a new formatter for a configuration", func() {
		Context("with a valid configuration", func() {
			cfg := runtime.Config{
				ResponseFormat: "json",
			}

			It("should return a formatter and no error", func() {
				formatter, err := formatters.NewForConfig(cfg)
				Expect(err).ToNot(HaveOccurred())
				Expect(formatter).ToNot(BeNil())
			})
		})

		Context("with an unknown format requested by the user", func() {
			cfg := runtime.Config{
				ResponseFormat: "unknownFormat",
			}

			It("should return an error", func() {
				formatter, err := formatters.NewForConfig(cfg)

				Expect(err).To(HaveOccurred())
				Expect(formatter).To(BeNil())
			})
		})
	})

	Describe("When creating a new generic formatter", func() {
		Context("with proper arguments", func() {
			var expectedResult []byte = []byte("this is a test")
			var name string = "testFormatter"
			var fn formatters.FormatterFunc = func(runtime.Results) ([]byte, error) {
				return expectedResult, nil
			}

			formatter, err := formatters.New(name, fn)
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

	Describe("When querying all supported formats", func() {
		all := formatters.AllFormats()
		It("should support at least one format", func() {
			Expect(len(all)).ToNot(BeZero())
		})

		It("should support the default format", func() {
			var exists = false
			for _, format := range all {
				if format == cmd.DefaultOutputFormat {
					exists = true
					break
				}
			}

			Expect(exists).To(BeTrue())
		})
	})
})
