package bundle

import (
	"bytes"
	"context"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
)

var _ = Describe("BundleValidateCheck", func() {
	Describe("Bundle validation", func() {
		Context("the annotations file is valid", func() {
			It("should pass", func() {
				imageRef := image.ImageReference{
					ImageFSPath: "./testdata/valid_bundle",
				}
				report, err := Validate(context.Background(), imageRef.ImageFSPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(report).ToNot(BeNil())
			})
		})

		Context("the annotations file does not exist", func() {
			It("should error", func() {
				imageRef := image.ImageReference{
					ImageFSPath: "./testdata/no_annotations_file",
				}
				report, err := Validate(context.Background(), imageRef.ImageFSPath)
				Expect(err).To(HaveOccurred())
				Expect(report).To(BeNil())
			})
		})

		Context("the annotations file is malformed", func() {
			It("should error", func() {
				imageRef := image.ImageReference{
					ImageFSPath: "./testdata/malformed_annotations_file",
				}
				report, err := Validate(context.Background(), imageRef.ImageFSPath)
				Expect(err).To(HaveOccurred())
				Expect(report).To(BeNil())
			})
		})

		Context("the annotations file is valid but has no annotations", func() {
			It("should fail gracefully", func() {
				imageRef := image.ImageReference{
					ImageFSPath: "./testdata/invalid_bundle",
				}
				report, err := Validate(context.Background(), imageRef.ImageFSPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(report).ToNot(BeNil())
			})
		})
	})

	Describe("While ensuring that container util is working", func() {
		// tests: extractAnnotationsBytes
		Context("with an annotations yaml data read from disk", func() {
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
