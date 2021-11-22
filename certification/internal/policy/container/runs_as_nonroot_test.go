package container

import (
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

func userConfigFile(user string) (*cranev1.ConfigFile, error) {
	return &cranev1.ConfigFile{
		Config: cranev1.Config{
			User: user,
		},
	}, nil
}

func configFileWithEmptyUser() (*cranev1.ConfigFile, error) {
	return userConfigFile("")
}

func configFileWithGoodUser() (*cranev1.ConfigFile, error) {
	return userConfigFile("1000")
}

func configFileWithRootUid() (*cranev1.ConfigFile, error) {
	return userConfigFile("0")
}

func configFileWithRootUsername() (*cranev1.ConfigFile, error) {
	return userConfigFile("root")
}

var _ = Describe("RunAsNonRoot", func() {
	var (
		runAsNonRoot RunAsNonRootCheck
		imageRef     certification.ImageReference
	)

	BeforeEach(func() {
		fakeImage := fakecranev1.FakeImage{
			ConfigFileStub: configFileWithGoodUser,
		}
		imageRef.ImageInfo = &fakeImage
	})

	Describe("Checking manifest user is not root", func() {
		Context("When manifest user is not root", func() {
			It("should pass Validate", func() {
				ok, err := runAsNonRoot.Validate(imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
	})
	Describe("Checking manifest user is root", func() {
		Context("When manifest user is empty", func() {
			BeforeEach(func() {
				fakeImage := fakecranev1.FakeImage{
					ConfigFileStub: configFileWithEmptyUser,
				}
				imageRef.ImageInfo = &fakeImage
			})
			It("should not pass Validate", func() {
				ok, err := runAsNonRoot.Validate(imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("When manifest user is string root", func() {
			BeforeEach(func() {
				fakeImage := fakecranev1.FakeImage{
					ConfigFileStub: configFileWithRootUsername,
				}
				imageRef.ImageInfo = &fakeImage
			})
			It("should not pass Validate", func() {
				ok, err := runAsNonRoot.Validate(imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("When manifest user is UID 0", func() {
			BeforeEach(func() {
				fakeImage := fakecranev1.FakeImage{
					ConfigFileStub: configFileWithRootUid,
				}
				imageRef.ImageInfo = &fakeImage
			})
			It("should not pass Validate", func() {
				ok, err := runAsNonRoot.Validate(imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
