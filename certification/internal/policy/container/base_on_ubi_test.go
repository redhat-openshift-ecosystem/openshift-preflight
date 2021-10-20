package container

import (
	"os"
	"path/filepath"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

func labelsForUbiCheck(label string, bad bool) map[string]string {
	labels := map[string]string{
		"name":                 "name",
		"vendor":               "vendor",
		"version":              "version",
		"release":              "release",
		"summary":              "summary",
		"description":          "description",
		"com.redhat.component": label,
	}

	if bad {
		delete(labels, "com.redhat.component")
	}

	return labels
}

func goodLabelsConfigFile() (*cranev1.ConfigFile, error) {
	return &cranev1.ConfigFile{
		Config: cranev1.Config{
			Labels: labelsForUbiCheck("ubi8-container", false),
		},
	}, nil
}

func missingUbiLableConfigFile() (*cranev1.ConfigFile, error) {
	return &cranev1.ConfigFile{
		Config: cranev1.Config{
			Labels: labelsForUbiCheck("", true),
		},
	}, nil
}

func badLabelsConfigFile() (*cranev1.ConfigFile, error) {
	return &cranev1.ConfigFile{
		Config: cranev1.Config{
			Labels: labelsForUbiCheck("doesnothavestring", false),
		},
	}, nil
}

var _ = Describe("BaseOnUBI", func() {
	var (
		basedOnUbiCheck BasedOnUBICheck
		imageRef        certification.ImageReference
	)

	const osrelease = "os-release"

	BeforeEach(func() {
		fakeImage := fakecranev1.FakeImage{
			ConfigFileStub: goodLabelsConfigFile,
		}
		imageRef.ImageInfo = &fakeImage
		var err error
		tmpDir, err := os.MkdirTemp("", "based-on-ubi-*")
		Expect(err).ToNot(HaveOccurred())
		err = os.Mkdir(filepath.Join(tmpDir, "etc"), 0755)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(filepath.Join(tmpDir, "etc", osrelease), []byte(`ID="rhel"
NAME="Red Hat Enterprise Linux"
`), 0644)
		Expect(err).ToNot(HaveOccurred())
		imageRef.ImageFSPath = tmpDir
	})
	AfterEach(func() {
		os.RemoveAll(imageRef.ImageFSPath)
	})
	Describe("Checking for UBI as a base", func() {
		Context("When it is based on UBI 8", func() {
			Context("and has the correct os-release", func() {
				It("should pass Validate", func() {
					ok, err := basedOnUbiCheck.Validate(imageRef)
					Expect(err).ToNot(HaveOccurred())
					Expect(ok).To(BeTrue())
				})
			})
		})
		Context("When it is based on UBI 7", func() {
			Context("and has the correct os-release", func() {
				JustBeforeEach(func() {
					err := os.WriteFile(filepath.Join(imageRef.ImageFSPath, "etc", osrelease), []byte(`ID="rhel"
NAME="Red Hat Enterprise Linux Server"
`), 0644)
					Expect(err).ToNot(HaveOccurred())
				})
				It("should pass Validate", func() {
					ok, err := basedOnUbiCheck.Validate(imageRef)
					Expect(err).ToNot(HaveOccurred())
					Expect(ok).To(BeTrue())
				})
			})
		})
		Context("When it is not based on UBI", func() {
			Context("and has a bad os-release", func() {
				JustBeforeEach(func() {
					err := os.WriteFile(filepath.Join(imageRef.ImageFSPath, "etc", osrelease), []byte("Not a good file"), 0644)
					Expect(err).ToNot(HaveOccurred())
				})
				It("should not pass Validate", func() {
					ok, err := basedOnUbiCheck.Validate(imageRef)
					Expect(err).ToNot(HaveOccurred())
					Expect(ok).To(BeFalse())
				})
			})
			Context("and os-release is missing", func() {
				JustBeforeEach(func() {
					os.Remove(filepath.Join(imageRef.ImageFSPath, "etc", osrelease))
				})
				It("should not pass Validate", func() {
					ok, err := basedOnUbiCheck.Validate(imageRef)
					Expect(err).To(HaveOccurred())
					Expect(ok).To(BeFalse())
				})
			})
		})
	})
})
