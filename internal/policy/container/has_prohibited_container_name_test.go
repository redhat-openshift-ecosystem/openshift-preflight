package container

import (
	"context"

	"github.com/google/go-containerregistry/pkg/name"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
)

var _ = Describe("HasProhibitedContainerName", func() {
	var (
		hasProhibitedContainerName HasProhibitedContainerName
		imageRef                   image.ImageReference
	)

	Describe("Checking for trademark violations", func() {
		Context("When a container name does not violate trademark", func() {
			BeforeEach(func() {
				imageRef.ImageRepository = "opdev/simple-demo-operator"
			})
			It("should pass Validate", func() {
				ok, err := hasProhibitedContainerName.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When a local registry container name does not violate trademark", func() {
			BeforeEach(func() {
				imageRef.ImageRepository = "simple-demo-operator"
			})
			It("should pass Validate", func() {
				ok, err := hasProhibitedContainerName.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When a local registry containing rhcc remote repo path container name does not violate trademark", func() {
			BeforeEach(func() {
				reference, err := name.ParseReference("local.registry.test.example/quay.io/redhat-isv-containers/12345678900987654321123")
				Expect(err).ToNot(HaveOccurred())

				imageRef.ImageRepository = reference.Context().RepositoryStr()
			})
			It("should pass Validate", func() {
				ok, err := hasProhibitedContainerName.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When a container name violates trademark", func() {
			BeforeEach(func() {
				imageRef.ImageRepository = "opdev/red-hat-container"
			})
			It("should not pass Validate", func() {
				ok, err := hasProhibitedContainerName.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})

	AssertMetaData(&hasProhibitedContainerName)
})
