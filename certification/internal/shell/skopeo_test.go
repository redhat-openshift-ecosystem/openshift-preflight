package shell

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Skopeo Engine", func() {
	var (
		skopeoEngine SkopeoCLIEngine
	)
	Describe("Fetching the image name", func() {
		Context("When it has a tag", func() {
			It("should return image name", func() {
				image, err := skopeoEngine.imageName("quay.io/opdev/pachyderm-operator:0.0.5")
				Expect(err).ToNot(HaveOccurred())
				Expect(image).To(Equal("quay.io/opdev/pachyderm-operator"))
			})
		})
		Context("When it has a digest", func() {
			It("should return image name", func() {
				image, err := skopeoEngine.imageName("quay.io/opdev/ubi-micro@sha256:d389f")
				Expect(err).ToNot(HaveOccurred())
				Expect(image).To(Equal("quay.io/opdev/ubi-micro"))
			})
		})
		Context("When it does not have neither a digest, nor a tag", func() {
			It("should return error", func() {
				image, err := skopeoEngine.imageName("quay.io/opdev/ubi-micro")
				Expect(err).To(HaveOccurred())
				Expect(image).To(Equal(""))
			})
		})
		Context("When it has a tag, but does not have a registry", func() {
			It("should return image name", func() {
				image, err := skopeoEngine.imageName("opdev/ubi-micro:0.0.5")
				Expect(err).ToNot(HaveOccurred())
				Expect(image).To(Equal("opdev/ubi-micro"))
			})
		})
		Context("When it has a digest, but does not have a registry", func() {
			It("should return image name", func() {
				image, err := skopeoEngine.imageName("opdev/ubi-micro@sha256:d389f")
				Expect(err).ToNot(HaveOccurred())
				Expect(image).To(Equal("opdev/ubi-micro"))
			})
		})
		Context("When it does not have a digest/tag/registry", func() {
			It("should return error", func() {
				image, err := skopeoEngine.imageName("opdev/ubi-micro")
				Expect(err).To(HaveOccurred())
				Expect(image).To(Equal(""))
			})
		})
	})

})
