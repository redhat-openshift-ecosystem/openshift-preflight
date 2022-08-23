package bundle

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BundleValidateCheck", func() {
	const (
		manifestsDir       = "manifests"
		metadataDir        = "metadata"
		annotationFilename = "annotations.yaml"
		annotations        = `annotations:
  com.redhat.openshift.versions: "v4.6-v4.9"
  operators.operatorframework.io.bundle.package.v1: testPackage
  operators.operatorframework.io.bundle.channel.default.v1: testChannel
`
	)

	Describe("Bundle validation", func() {
		var (
			imageRef   certification.ImageReference
			fakeEngine operatorSdk
		)

		BeforeEach(func() {
			// mock bundle directory
			tmpDir, err := os.MkdirTemp("", "bundle-metadata-*")
			Expect(err).ToNot(HaveOccurred())

			err = os.Mkdir(filepath.Join(tmpDir, metadataDir), 0o755)
			Expect(err).ToNot(HaveOccurred())

			err = os.Mkdir(filepath.Join(tmpDir, manifestsDir), 0o755)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(filepath.Join(tmpDir, metadataDir, annotationFilename), []byte(annotations), 0o644)
			Expect(err).ToNot(HaveOccurred())

			imageRef.ImageFSPath = tmpDir
			fakeEngine = FakeOperatorSdk{}
		})

		AfterEach(func() {
			err := os.RemoveAll(imageRef.ImageFSPath)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("the annotations file is valid", func() {
			It("should pass", func() {
				report, err := Validate(context.Background(), fakeEngine, imageRef.ImageFSPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(report).ToNot(BeNil())
			})
		})

		Context("the annotations file does not exist", func() {
			JustBeforeEach(func() {
				err := os.Remove(filepath.Join(imageRef.ImageFSPath, metadataDir, annotationFilename))
				Expect(err).ToNot(HaveOccurred())
			})
			It("should error", func() {
				report, err := Validate(context.Background(), fakeEngine, imageRef.ImageFSPath)
				Expect(err).To(HaveOccurred())
				Expect(report).To(BeNil())
			})
		})

		Context("the annotations file is malformed", func() {
			JustBeforeEach(func() {
				err := os.WriteFile(filepath.Join(imageRef.ImageFSPath, metadataDir, annotationFilename), []byte("badAnnotations"), 0o644)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should error", func() {
				report, err := Validate(context.Background(), fakeEngine, imageRef.ImageFSPath)
				Expect(err).To(HaveOccurred())
				Expect(report).To(BeNil())
			})
		})

		Context("the annotations file is valid but has no annotations", func() {
			JustBeforeEach(func() {
				err := os.WriteFile(filepath.Join(imageRef.ImageFSPath, metadataDir, annotationFilename), []byte("annotations:"), 0o644)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should fail gracefully", func() {
				report, err := Validate(context.Background(), fakeEngine, imageRef.ImageFSPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(report).ToNot(BeNil())
			})
		})
	})

	Describe("While ensuring that container util is working", func() {
		// tests: extractAnnotationsBytes
		Context("with an annotations yaml data read from disk", func() {
			Context("with the correct format", func() {
				It("should properly marshal to a map[string]string", func() {
					annotations, err := LoadAnnotations(context.TODO(), bytes.NewReader([]byte(annotations)))
					Expect(err).ToNot(HaveOccurred())
					Expect(annotations.DefaultChannelName).To(Equal("testChannel"))
				})
			})

			Context("containing no data read in from the yaml file", func() {
				data := []byte{}

				It("should return an error", func() {
					_, err := LoadAnnotations(context.TODO(), bytes.NewReader(data))
					Expect(err).To(HaveOccurred())
				})
			})

			Context("containing malformed or unexpected data", func() {
				data := []byte(`malformed`)

				It("should return an error", func() {
					_, err := LoadAnnotations(context.TODO(), bytes.NewReader(data))
					Expect(err).To(HaveOccurred())
				})
			})

			Context("a bad reader is sent to GetAnnotations", func() {
				It("should return an error", func() {
					annotations, err := LoadAnnotations(context.TODO(), errReader(0))
					Expect(err).To(HaveOccurred())
					Expect(annotations).To(BeNil())
				})
			})
		})
	})

	DescribeTable("Image Registry validation",
		func(versions string, expected string, success bool) {
			version, err := targetVersion(versions)
			if success {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
			Expect(version).To(Equal(expected))
		},

		Entry("range 4.6 to 4.8", "v4.6-v4.8", "4.8", true),
		Entry("exactly 4.8", "=v4.8", "4.8", true),
		Entry("exactly 4.9", "=v4.9", "4.9", true),
		Entry("range 4.6 to 4.9", "v4.6-v4.9", "4.9", true),
		Entry(">= 4.8", "v4.8", latestReleasedVersion, true),
		Entry(">= 4.9", "v4.9", latestReleasedVersion, true),
		Entry(">= 4.11", "v4.11", latestReleasedVersion, true),
		Entry(">= 4.13, which is more than released", "v4.13", "4.13", true),
		Entry("begins = with error", "=foo", "", false),
		Entry("bare version with error", "vfoo", "", false),
		Entry("range with error", "v4.6-vfoo", "", false),
		Entry("open-ended range is error", "v4.11-", "", false),
	)
})
