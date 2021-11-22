package container

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

const (
	emptyLicense = "emptylicense.txt"
	validLicense = "mylicense.txt"
	licenses     = "licenses"
)

var _ = Describe("HasLicense", func() {
	var (
		HasLicense HasLicenseCheck
	)

	Describe("Checking if licenses can be found", func() {
		var (
			imgRef certification.ImageReference
		)
		BeforeEach(func() {
			var err error
			tmpDir, err := os.MkdirTemp("", "license-check-*")
			Expect(err).ToNot(HaveOccurred())
			err = os.Mkdir(filepath.Join(tmpDir, licenses), 0755)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(filepath.Join(tmpDir, licenses, validLicense), []byte("This is a license"), 0644)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Create(filepath.Join(tmpDir, licenses, emptyLicense))
			Expect(err).ToNot(HaveOccurred())
			imgRef.ImageFSPath = tmpDir
		})
		Context("When license(s) are found", func() {
			It("Should pass Validate", func() {
				ok, err := HasLicense.Validate(imgRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When licenses directory is not found", func() {
			JustBeforeEach(func() {
				imgRef.ImageFSPath = "/invalid"
			})
			It("Should not pass Validate", func() {
				ok, err := HasLicense.Validate(imgRef)
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
				ok, err := HasLicense.Validate(imgRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("Only an empty license", func() {
			JustBeforeEach(func() {
				os.Remove(filepath.Join(imgRef.ImageFSPath, licenses, validLicense))
			})
			It("Should not pass Validate", func() {
				ok, err := HasLicense.Validate(imgRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		AfterEach(func() {
			err := os.RemoveAll(imgRef.ImageFSPath)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
