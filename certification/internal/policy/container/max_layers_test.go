package container

import (
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

func generateLayers(layerCount int) []cranev1.Layer {
	layers := make([]cranev1.Layer, layerCount)
	for i := range layers {
		layers[i] = FakeLayer{}
	}
	return layers
}

func generateMinimalLayers() ([]cranev1.Layer, error) {
	return generateLayers(5), nil
}

func generateTooManyLayers() ([]cranev1.Layer, error) {
	return generateLayers(41), nil
}

var _ = Describe("LessThanMaxLayers", func() {
	var (
		maxLayersCheck MaxLayersCheck
		imgRef         certification.ImageReference
	)

	BeforeEach(func() {
		fakeImage := fakecranev1.FakeImage{
			LayersStub: generateMinimalLayers,
		}
		imgRef.ImageInfo = &fakeImage
	})

	Describe("Checking for less than max layers", func() {
		Context("When it has fewer layers than max", func() {
			It("should pass Validate", func() {
				ok, err := maxLayersCheck.Validate(imgRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When has more layers than max", func() {
			BeforeEach(func() {
				fakeImage := fakecranev1.FakeImage{
					LayersStub: generateTooManyLayers,
				}
				imgRef.ImageInfo = &fakeImage
			})
			It("should not succeed the check", func() {
				ok, err := maxLayersCheck.Validate(imgRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
