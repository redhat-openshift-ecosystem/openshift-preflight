package container

import (
	"context"
	"errors"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
)

func getTrademarkLabels(bad bool) map[string]string {
	labels := map[string]string{
		"name":       "name",
		"vendor":     "vendor",
		"maintainer": "maintainer",
	}

	if bad {
		labels["maintainer"] = "Red Hat"
	}

	return labels
}

func getProhibitedConfigFile() (*cranev1.ConfigFile, error) {
	return &cranev1.ConfigFile{
		Config: cranev1.Config{
			Labels: getTrademarkLabels(false),
		},
	}, nil
}

func getBadProhibitedConfigFile() (*cranev1.ConfigFile, error) {
	return &cranev1.ConfigFile{
		Config: cranev1.Config{
			Labels: getTrademarkLabels(true),
		},
	}, nil
}

var _ = Describe("HasNoProhibitedLabelsCheck", func() {
	var (
		hasProhibitedLabelsCheck HasNoProhibitedLabelsCheck
		imageRef                 image.ImageReference
	)

	BeforeEach(func() {
		fakeImage := fakecranev1.FakeImage{
			ConfigFileStub: getProhibitedConfigFile,
		}
		imageRef.ImageInfo = &fakeImage
	})

	Describe("Checking for prohibited labels", func() {
		Context("When it has no prohibited labels", func() {
			It("should pass Validate", func() {
				ok, err := hasProhibitedLabelsCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When it has prohibited labels", func() {
			BeforeEach(func() {
				fakeImage := fakecranev1.FakeImage{
					ConfigFileStub: getBadProhibitedConfigFile,
				}
				imageRef.ImageInfo = &fakeImage
			})
			It("should not pass Validate", func() {
				ok, err := hasProhibitedLabelsCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		Context("When ConfigFile returns an error", func() {
			BeforeEach(func() {
				fakeImage := fakecranev1.FakeImage{
					ConfigFileStub: func() (*cranev1.ConfigFile, error) {
						return &cranev1.ConfigFile{}, errors.New("config error")
					},
				}
				imageRef.ImageInfo = &fakeImage
			})
			It("should return an error", func() {
				ok, err := hasProhibitedLabelsCheck.Validate(context.TODO(), imageRef)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("could not retrieve image labels"))
				Expect(ok).To(BeFalse())
			})
		})
		Context("When violatesRedHatTrademark returns an error", func() {
			var origValidator func(string) (bool, error)
			BeforeEach(func() {
				origValidator = violatesRedHatTrademark
				violatesRedHatTrademark = func(_ string) (bool, error) {
					return false, errors.New("trademark error")
				}
			})
			AfterEach(func() {
				violatesRedHatTrademark = origValidator
			})
			It("should return an error", func() {
				ok, err := hasProhibitedLabelsCheck.Validate(context.TODO(), imageRef)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error while validating label"))
				Expect(ok).To(BeFalse())
			})
		})
	})

	AssertMetaData(&hasProhibitedLabelsCheck)
})
