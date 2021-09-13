package operator

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ContainerUtil", func() {

	Describe("While ensuring that container util is working", func() {

		// tests: extractAnnotationsBytes
		Context("with an annotations yaml data read from disk", func() {
			Context("with the correct format", func() {
				data := []byte("annotations:\n foo: bar")

				It("should properly marshal to a map[string]string", func() {
					annotations, err := extractAnnotationsBytes(data)
					Expect(err).ToNot(HaveOccurred())
					Expect(annotations["foo"]).To(Equal("bar"))
				})
			})

			Context("containing no data read in from the yaml file", func() {
				data := []byte{}

				It("should return an error", func() {
					_, err := extractAnnotationsBytes(data)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("containing malformed or unexpected data", func() {
				data := []byte(`malformed`)

				It("should return an error", func() {
					_, err := extractAnnotationsBytes(data)
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
