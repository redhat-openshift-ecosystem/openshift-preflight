package container

import (
	"context"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
)

func getLabels(override string) map[string]string {
	labels := map[string]string{
		"name":        "Something for Red Hat OpenShift",
		"maintainer":  "maintainer",
		"vendor":      "vendor",
		"version":     "version",
		"release":     "release",
		"summary":     "summary",
		"description": "description",
	}

	switch override {
	case "remove-label":
		delete(labels, "description")
	case "violates-trademark":
		labels["name"] = "Red Hat"
	}

	return labels
}

func getConfigFile() (*cranev1.ConfigFile, error) {
	return &cranev1.ConfigFile{
		Config: cranev1.Config{
			Labels: getLabels(""),
		},
	}, nil
}

func getRemoveLabelConfigFile() (*cranev1.ConfigFile, error) {
	return &cranev1.ConfigFile{
		Config: cranev1.Config{
			Labels: getLabels("remove-label"),
		},
	}, nil
}

func getViolatesTrademarkConfigFile() (*cranev1.ConfigFile, error) {
	return &cranev1.ConfigFile{
		Config: cranev1.Config{
			Labels: getLabels("violates-trademark"),
		},
	}, nil
}

var _ = Describe("HasRequiredLabels", func() {
	var (
		hasRequiredLabelsCheck HasRequiredLabelsCheck
		imageRef               image.ImageReference
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
				ok, err := hasRequiredLabelsCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When it does not have required labels", func() {
			BeforeEach(func() {
				fakeImage := fakecranev1.FakeImage{
					ConfigFileStub: getRemoveLabelConfigFile,
				}
				imageRef.ImageInfo = &fakeImage
			})
			It("should not succeed the check", func() {
				ok, err := hasRequiredLabelsCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("When label.name violates Red Hat Trademark", func() {
			BeforeEach(func() {
				fakeImage := fakecranev1.FakeImage{
					ConfigFileStub: getViolatesTrademarkConfigFile,
				}
				imageRef.ImageInfo = &fakeImage
			})
			It("should not succeed the check and throw an error", func() {
				ok, err := hasRequiredLabelsCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})

	AssertMetaData(&hasRequiredLabelsCheck)
})
