package container

import (
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

func getLabels(bad bool) map[string]string {
	labels := map[string]string{
		"name":        "name",
		"vendor":      "vendor",
		"version":     "version",
		"release":     "release",
		"summary":     "summary",
		"description": "description",
	}

	if bad {
		delete(labels, "description")
	}

	return labels
}

func getConfigFile() (*cranev1.ConfigFile, error) {
	return &cranev1.ConfigFile{
		Config: cranev1.Config{
			Labels: getLabels(false),
		},
	}, nil
}

func getBadConfigFile() (*cranev1.ConfigFile, error) {
	return &cranev1.ConfigFile{
		Config: cranev1.Config{
			Labels: getLabels(true),
		},
	}, nil
}

var _ = Describe("HasRequiredLabels", func() {
	var (
		hasRequiredLabelsCheck HasRequiredLabelsCheck
		imageRef               certification.ImageReference
	)

	BeforeEach(func() {
		fakeImage := fakecranev1.FakeImage{
			ConfigFileStub: getConfigFile,
		}
		imageRef.ImageInfo = &fakeImage
	})

	Describe("Checking for required labels", func() {
		Context("When it has required labels", func() {
			It("should pass Validate", func() {
				ok, err := hasRequiredLabelsCheck.Validate(imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When it does not have required labels", func() {
			BeforeEach(func() {
				fakeImage := fakecranev1.FakeImage{
					ConfigFileStub: getBadConfigFile,
				}
				imageRef.ImageInfo = &fakeImage
			})
			It("should not succeed the check", func() {
				ok, err := hasRequiredLabelsCheck.Validate(imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
