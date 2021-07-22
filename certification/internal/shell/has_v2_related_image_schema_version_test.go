package shell

import (
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RelatedImagesAreSchemaVersion2", func() {
	check := RelatedImagesAreSchemaVersion2Check{}

	// tests check.validate(...) - this check does not currently have a case
	// that throws an error, so no test case exists for it.
	Context("When validating a mapping of images to their image manifest schema versions", func() {
		Context("with images that all have image manifests in schema version 2", func() {
			goodImageSchemaVersMap := map[string]int{
				"dummy/image1": 2,
			}

			It("should pass validation", func() {
				passed, err := check.validate(goodImageSchemaVersMap)
				Expect(err).ToNot(HaveOccurred())
				Expect(passed).To(BeTrue())
			})
		})

		Context("with an image that does not have schema version 2", func() {
			badImageSchemaVersMap := map[string]int{
				"dummy/image1": 1,
				"dummy/image2": 2,
			}

			It("should fail validation", func() {
				passed, err := check.validate(badImageSchemaVersMap)
				Expect(err).ToNot(HaveOccurred())
				Expect(passed).To(BeFalse())
			})
		})
	})

	// test getRelatedImagesForCSV(...) - this function doesn't currently have a case
	// that returns an error.
	Context("When extracting related images from a ClusterServiceVersion resource", func() {
		Context("With a ClusterServiceVersion that has related images", func() {
			relatedImages := []operatorsv1alpha1.RelatedImage{
				{Image: "dummy/image1"},
				{Image: "dummy/image2"},
			}

			csvWithRelatedImages := operatorsv1alpha1.ClusterServiceVersion{
				Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
					RelatedImages: relatedImages,
				},
			}

			images, err := check.getRelatedImagesForCSV(&csvWithRelatedImages)
			It("should successfully return the related images", func() {
				Expect(len(images)).To(Equal(len(relatedImages)))
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("With a ClusterServiceVersion that does not have related images", func() {
			csvWithoutRelatedImages := operatorsv1alpha1.ClusterServiceVersion{}
			images, err := check.getRelatedImagesForCSV(&csvWithoutRelatedImages)
			It("should successfully return the related images", func() {
				Expect(images).To(BeNil())
				// not having related images does not constitute an error case
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	// tests check.getSchemaVersionFromRawManifest(...)
	Context("When extracting the schemaVersion value from a raw manifest", func() {
		Context("with a manifest that does not marshal to a map[string]interface{}", func() {
			blob := []byte(`[]`) // valid JSON

			_, err := check.getSchemaVersionFromRawManifest(blob)
			It("should throw an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with a manifest that does not have a schemaVersion key", func() {
			blob := []byte(`{"this": "that"}`)

			_, err := check.getSchemaVersionFromRawManifest(blob)
			It("should throw an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with a manifest that has a schemaVersion key with a non-numerical value", func() {
			blob := []byte(`{"schemaVersion": "2"}`)

			_, err := check.getSchemaVersionFromRawManifest(blob)
			It("should throw an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with a manifest that has the expected schemaVersion key with a numerical value", func() {
			blob := []byte(`{"schemaVersion": 2}`)

			vers, err := check.getSchemaVersionFromRawManifest(blob)
			It("should succeed and return the schemaVersion as an integer", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(vers).To(Equal(2))
			})
		})
	})

	// tests check.extractBundleFromImage - this makes multiple calls using the podman engine,
	// so those calls are replaced with fakes. stdout/stderr doesn't have much bearing here as
	// copying data from the container to the host does not produce any output at default verbosity.
	Context("When extracting a bundle from an image", func() {
		bundleImage := "dummy/image"

		BeforeEach(func() {
			podmanEngine = FakePodmanEngine{
				CreateReport: cli.PodmanCreateReport{
					Stdout:      "abcdefghijklmnopqrstuvwxyz\n",
					Stderr:      "",
					ContainerID: "abcdefghijklmnopqrstuvwxyz", // simulated container ID
				},
				CopyFromReport: cli.PodmanCopyReport{},
				RemoveReport: cli.PodmanRemoveReport{
					Stdout: "abcdefghijklmnopqrstuvwxyz",
				},
			}
		})

		Context("with no issues creating, copying or removing the container", func() {
			It("should succeed", func() {
				_, err := check.extractBundleFromImage(bundleImage)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
