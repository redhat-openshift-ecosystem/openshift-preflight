package container

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

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

		Context("When /licenses is a symlink to a directory that contains license files", func() {
			It("Should pass Validate and count each regular file, not the symlink alone", func() {
				if runtime.GOOS == "windows" {
					Skip("symlink fixture not portable on Windows")
				}
				tmpDir := setupTmpDir()
				targetDir := filepath.Join(tmpDir, "licenses-target")
				Expect(os.Mkdir(targetDir, 0o755)).To(Succeed())
				licenseNames := []string{validLicense, "second-license.txt", "third-license.txt"}
				for _, name := range licenseNames {
					Expect(os.WriteFile(filepath.Join(targetDir, name), []byte("This is a license"), 0o644)).To(Succeed())
				}
				Expect(os.Symlink(targetDir, filepath.Join(tmpDir, licenses))).To(Succeed())

				entries, err := hasLicense.getDataToValidate(context.TODO(), tmpDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(entries).To(HaveLen(len(licenseNames)), "buggy walk counted the /licenses symlink as one file instead of walking the target directory")

				ok, err := hasLicense.Validate(context.TODO(), image.ImageReference{ImageFSPath: tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})

		Context("When a license path is a symlink to an empty regular file", func() {
			It("Should not pass Validate based on the target file size, not the symlink", func() {
				if runtime.GOOS == "windows" {
					Skip("symlink fixture not portable on Windows")
				}
				tmpDir := setupTmpDir()
				createLicenseDir(tmpDir)
				target := filepath.Join(tmpDir, "empty-target.txt")
				f, err := os.Create(target)
				Expect(err).ToNot(HaveOccurred())
				Expect(f.Close()).To(Succeed())
				Expect(os.Symlink(target, filepath.Join(tmpDir, licenses, validLicense))).To(Succeed())

				ok, err := hasLicense.Validate(context.TODO(), image.ImageReference{ImageFSPath: tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		Context("When a license path is a symlink to an empty directory", func() {
			It("Should not pass Validate", func() {
				if runtime.GOOS == "windows" {
					Skip("symlink fixture not portable on Windows")
				}
				tmpDir := setupTmpDir()
				emptyTarget := filepath.Join(tmpDir, "empty-license-target")
				Expect(os.Mkdir(emptyTarget, 0o755)).To(Succeed())
				Expect(os.Symlink(emptyTarget, filepath.Join(tmpDir, licenses))).To(Succeed())

				entries, err := hasLicense.getDataToValidate(context.TODO(), tmpDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(entries).To(BeEmpty())

				ok, err := hasLicense.Validate(context.TODO(), image.ImageReference{ImageFSPath: tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		Context("When a license file is a symlink pointing outside the image root", func() {
			It("Should not count the external target", func() {
				if runtime.GOOS == "windows" {
					Skip("symlink fixture not portable on Windows")
				}
				tmpDir := setupTmpDir()
				createLicenseDir(tmpDir)
				outside := filepath.Join(tmpDir, "..", "outside-license-"+filepath.Base(tmpDir)+".txt")
				Expect(os.WriteFile(outside, []byte("This is a license"), 0o644)).To(Succeed())
				DeferCleanup(func() { _ = os.Remove(outside) })
				Expect(os.Symlink(outside, filepath.Join(tmpDir, licenses, validLicense))).To(Succeed())

				entries, err := hasLicense.getDataToValidate(context.TODO(), tmpDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(entries).To(BeEmpty())

				ok, err := hasLicense.Validate(context.TODO(), image.ImageReference{ImageFSPath: tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		Context("When a license path is a symlink to a directory containing only directories (no files)", func() {
			It("Should not pass Validate", func() {
				if runtime.GOOS == "windows" {
					Skip("symlink fixture not portable on Windows")
				}
				tmpDir := setupTmpDir()
				createLicenseDir(tmpDir)
				nestedOnly := filepath.Join(tmpDir, "nested-only-target")
				Expect(os.MkdirAll(filepath.Join(nestedOnly, "inner", "deep"), 0o755)).To(Succeed())
				Expect(os.Symlink(nestedOnly, filepath.Join(tmpDir, licenses, "license-link-dirs-only"))).To(Succeed())

				entries, err := hasLicense.getDataToValidate(context.TODO(), tmpDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(entries).To(BeEmpty())

				ok, err := hasLicense.Validate(context.TODO(), image.ImageReference{ImageFSPath: tmpDir})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})

		AssertMetaData(&hasLicense)
	})
})

var _ = Describe("pathWithinMount", func() {
	It("returns true when resolved equals the mount root after filepath.Clean", func() {
		mount := filepath.Join("scratch", "image-root")
		Expect(pathWithinMount(mount, mount)).To(BeTrue())

		withTrailing := mount + string(filepath.Separator)
		Expect(pathWithinMount(mount, withTrailing)).To(BeTrue())
		Expect(pathWithinMount(withTrailing, mount)).To(BeTrue())
	})

	It("returns false when filepath.Rel cannot relate mount to resolved", func() {
		// filepath.Rel errors when base is relative and target is absolute (or vice versa),
		// which exercises the err != nil branch in pathWithinMount.
		tmpDir := setupTmpDir()
		Expect(pathWithinMount("image-root", filepath.Join(tmpDir, "any"))).To(BeFalse())
	})
})
