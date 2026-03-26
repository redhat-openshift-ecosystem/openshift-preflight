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

func setupTmpDir() string {
	tmpDir, err := os.MkdirTemp("", "license-check-*")
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(func() {
		os.RemoveAll(tmpDir)
	})
	return tmpDir
}

func createLicenseDir(tmpDir string) {
	err := os.Mkdir(filepath.Join(tmpDir, licenses), 0o755)
	Expect(err).ToNot(HaveOccurred())
}

func createLicenseFile(tmpDir, path string) {
	fullPath := filepath.Join(tmpDir, licenses, path)
	err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
	Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(fullPath, []byte("This is a license"), 0o644)
	Expect(err).ToNot(HaveOccurred())
}

func createEmptyFile(tmpDir, path string) {
	fullPath := filepath.Join(tmpDir, licenses, path)
	_, err := os.Create(fullPath)
	Expect(err).ToNot(HaveOccurred())
}

var _ = Describe("HasLicense", func() {
	var hasLicense HasLicenseCheck

	Describe("Checking if licenses can be found", func() {
		Context("When license(s) are found at top level", func() {
			It("Should pass Validate", func() {
				tmpDir := setupTmpDir()
				createLicenseDir(tmpDir)
				createLicenseFile(tmpDir, validLicense)
				createEmptyFile(tmpDir, emptyLicense)

				ok, err := hasLicense.Validate(context.TODO(), image.ImageReference{ImageFSPath: tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})

		Context("When licenses directory is not found", func() {
			It("Should not pass Validate", func() {
				ok, err := hasLicense.Validate(context.TODO(), image.ImageReference{ImageFSPath: "/invalid"})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		// This shouldn't happen in practice, since the untar extraction
		// logic will prune/not create empty directories.
		Context("When licenses directory exists but is empty", func() {
			It("Should not pass Validate", func() {
				tmpDir := setupTmpDir()
				createLicenseDir(tmpDir)

				ok, err := hasLicense.Validate(context.TODO(), image.ImageReference{ImageFSPath: tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		Context("When only an empty license file exists", func() {
			It("Should not pass Validate", func() {
				tmpDir := setupTmpDir()
				createLicenseDir(tmpDir)
				createEmptyFile(tmpDir, emptyLicense)

				ok, err := hasLicense.Validate(context.TODO(), image.ImageReference{ImageFSPath: tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		// This shouldn't happen in practice, since the untar extraction
		// logic will prune/not create empty directories.
		Context("When only directories exist in the license folder", func() {
			It("Should not pass Validate", func() {
				tmpDir := setupTmpDir()
				createLicenseDir(tmpDir)
				err := os.MkdirAll(filepath.Join(tmpDir, licenses, "just-another-dir"), 0o755)
				Expect(err).ToNot(HaveOccurred())

				ok, err := hasLicense.Validate(context.TODO(), image.ImageReference{ImageFSPath: tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		Context("When license is found only in nested subdirectory", func() {
			It("Should pass Validate", func() {
				tmpDir := setupTmpDir()
				createLicenseDir(tmpDir)
				createLicenseFile(tmpDir, filepath.Join("a/b/c", validLicense))

				ok, err := hasLicense.Validate(context.TODO(), image.ImageReference{ImageFSPath: tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})

		AssertMetaData(&hasLicense)
	})
})
