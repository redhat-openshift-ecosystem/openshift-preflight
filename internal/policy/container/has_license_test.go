package container

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
)

const (
	emptyLicense = "emptylicense.txt"
	validLicense = "mylicense.txt"
	licenses     = "licenses"
)

var _ = Describe("HasLicense", func() {
	var hasLicense HasLicenseCheck

	Describe("Checking if licenses can be found", func() {
		var imgRef image.ImageReference
		BeforeEach(func() {
			var err error
			tmpDir, err := os.MkdirTemp("", "license-check-*")
			Expect(err).ToNot(HaveOccurred())
			err = os.Mkdir(filepath.Join(tmpDir, licenses), 0o755)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(filepath.Join(tmpDir, licenses, validLicense), []byte("This is a license"), 0o644)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Create(filepath.Join(tmpDir, licenses, emptyLicense))
			Expect(err).ToNot(HaveOccurred())
			imgRef.ImageFSPath = tmpDir
		})
		Context("When license(s) are found", func() {
			It("Should pass Validate", func() {
				ok, err := hasLicense.Validate(context.TODO(), imgRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When licenses directory is not found", func() {
			It("Should not pass Validate", func() {
				badImgRef := imgRef
				badImgRef.ImageFSPath = "/invalid"
				ok, err := hasLicense.Validate(context.TODO(), badImgRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("Licenses can't be found when directory exists", func() {
			JustBeforeEach(func() {
				os.Remove(filepath.Join(imgRef.ImageFSPath, licenses, validLicense))
				os.Remove(filepath.Join(imgRef.ImageFSPath, licenses, emptyLicense))
			})
			It("Should not pass Validate", func() {
				ok, err := hasLicense.Validate(context.TODO(), imgRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("Only an empty license", func() {
			JustBeforeEach(func() {
				os.Remove(filepath.Join(imgRef.ImageFSPath, licenses, validLicense))
			})
			It("Should not pass Validate", func() {
				ok, err := hasLicense.Validate(context.TODO(), imgRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		AfterEach(func() {
			err := os.RemoveAll(imgRef.ImageFSPath)
			Expect(err).ToNot(HaveOccurred())
		})

		AssertMetaData(&hasLicense)
	})
})
