package container

import (
	"context"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo/v2"
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

	Context("When checking metadata", func() {
		Context("The check name should not be empty", func() {
			Expect(hasRequiredLabelsCheck.Name()).ToNot(BeEmpty())
		})

		Context("The metadata keys should not be empty", func() {
			meta := hasRequiredLabelsCheck.Metadata()
			Expect(meta.CheckURL).ToNot(BeEmpty())
			Expect(meta.Description).ToNot(BeEmpty())
			Expect(meta.KnowledgeBaseURL).ToNot(BeEmpty())
			// Level is optional.
		})

		Context("The help text should not be empty", func() {
			help := hasRequiredLabelsCheck.Help()
			Expect(help.Message).ToNot(BeEmpty())
			Expect(help.Suggestion).ToNot(BeEmpty())
		})
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
					ConfigFileStub: getBadConfigFile,
				}
				imageRef.ImageInfo = &fakeImage
			})
			It("should not succeed the check", func() {
				ok, err := hasRequiredLabelsCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
