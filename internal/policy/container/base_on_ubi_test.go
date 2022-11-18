package container

import (
	"context"
	"net/http"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func ConfigFile() (*cranev1.ConfigFile, error) {
	return &cranev1.ConfigFile{
		Config: cranev1.Config{},
	}, nil
}

type fakeLayerHashChecker struct{}

func (flhc *fakeLayerHashChecker) CertifiedImagesContainingLayers(ctx context.Context, layers []cranev1.Hash) ([]pyxis.CertImage, error) {
	var matchingImages []pyxis.CertImage
	matchingImages = append(matchingImages, pyxis.CertImage{})
	return matchingImages, nil
}

type fakeLayerHashCheckerNoMatch struct{}

func (flhc *fakeLayerHashCheckerNoMatch) CertifiedImagesContainingLayers(ctx context.Context, layers []cranev1.Hash) ([]pyxis.CertImage, error) {
	var matchingImages []pyxis.CertImage
	return matchingImages, nil
}

type fakeLayerHashCheckerTimeout struct{}

func (flhc *fakeLayerHashCheckerTimeout) CertifiedImagesContainingLayers(ctx context.Context, layers []cranev1.Hash) ([]pyxis.CertImage, error) {
	return nil, http.ErrHandlerTimeout
}

var _ = Describe("BaseOnUBI", func() {
	var (
		basedOnUbiCheck BasedOnUBICheck
		imageRef        image.ImageReference
	)

	BeforeEach(func() {
		fakeImage := fakecranev1.FakeImage{
			ConfigFileStub: ConfigFile,
		}
		imageRef.ImageInfo = &fakeImage
	})
	AfterEach(func() {
		os.RemoveAll(imageRef.ImageFSPath)
	})
	Describe("Checking for UBI as a base", func() {
		Context("When the image contains a layer hash that is a ubi or ubi derived uncompressed top layer id", func() {
			JustBeforeEach(func() {
				basedOnUbiCheck.LayerHashCheckEngine = &fakeLayerHashChecker{}
			})
			Context("and pyxis returns a match", func() {
				It("should pass Validate", func() {
					ok, err := basedOnUbiCheck.Validate(context.TODO(), imageRef)
					Expect(err).ToNot(HaveOccurred())
					Expect(ok).To(BeTrue())
				})
			})
		})
		Context("When it is not based on UBI", func() {
			JustBeforeEach(func() {
				basedOnUbiCheck.LayerHashCheckEngine = &fakeLayerHashCheckerNoMatch{}
			})
			Context("When the image does not contain a layer hash that is a ubi or ubi derived uncompressed top layer id", func() {
				Context("and pyxis returns no matches", func() {
					It("should not pass Validate", func() {
						ok, err := basedOnUbiCheck.Validate(context.TODO(), imageRef)
						Expect(err).ToNot(HaveOccurred())
						Expect(ok).To(BeFalse())
					})
				})
			})
		})
		Context("When the pyxis call times out", func() {
			JustBeforeEach(func() {
				basedOnUbiCheck.LayerHashCheckEngine = &fakeLayerHashCheckerTimeout{}
			})
			It("should return an error", func() {
				ok, err := basedOnUbiCheck.Validate(context.TODO(), imageRef)
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		AssertMetaData(&basedOnUbiCheck)
	})
})
