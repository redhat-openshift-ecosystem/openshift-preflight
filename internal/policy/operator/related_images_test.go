package operator

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
)

var _ = Describe("RelatedImages", func() {
	const (
		manifestsDir                  = "manifests"
		clusterServiceVersionFilename = "myoperator.clusterserviceversion.yaml"
	)
	var (
		relatedImagesCheck *RelatedImagesCheck
		imageRef           image.ImageReference
		csvContents        = `kind: ClusterServiceVersion
apiVersion: operators.coreos.com/v1alpha1
spec:
  install:
    spec:
      deployments:
      - spec:
          template:
            spec:
              containers:
              - image: registry.example.io/foo/bar@sha256:f000432f07cd187469f0310e3ed9dcf9a5db2be14b8bab9c5293dd1ee8518176
                name: the-operator
  relatedImages:
  - name: the-operator
    image: registry.example.io/foo/bar@sha256:f000432f07cd187469f0310e3ed9dcf9a5db2be14b8bab9c5293dd1ee8518176
  - name: the-proxy
    image: registry.example.io/foo/proxy@sha256:5e33f9d095952866b9743cc8268fb740cce6d93439f00ce333a2de1e5974837e`
	)

	BeforeEach(func() {
		relatedImagesCheck = &RelatedImagesCheck{}
		tmpDir, err := os.MkdirTemp("", "related-images-bundle-*")
		Expect(err).ToNot(HaveOccurred())
		imageRef.ImageFSPath = tmpDir
		DeferCleanup(os.RemoveAll, tmpDir)

		err = os.Mkdir(filepath.Join(tmpDir, manifestsDir), 0o755)
		Expect(err).ToNot(HaveOccurred())

		err = os.WriteFile(filepath.Join(tmpDir, manifestsDir, clusterServiceVersionFilename), []byte(csvContents), 0o644)
		Expect(err).ToNot(HaveOccurred())
	})
	When("given a good CSV", func() {
		It("should succeed", func() {
			result, err := relatedImagesCheck.Validate(context.TODO(), imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})
	})
	When("there is no CSV", func() {
		It("should fail", func() {
			Expect(os.RemoveAll(imageRef.ImageFSPath)).To(Succeed())
			result, err := relatedImagesCheck.Validate(context.TODO(), imageRef)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeFalse())
		})
	})
	When("the CSV is malformed", func() {
		It("should fail", func() {
			csvContents := `kind: ClusterServiceVersion
apiVersion: operators.coreos.com/v1alpha1
spec:
`
			Expect(os.WriteFile(filepath.Join(imageRef.ImageFSPath, manifestsDir, clusterServiceVersionFilename), []byte(csvContents), 0o644)).To(Succeed())
			result, err := relatedImagesCheck.Validate(context.TODO(), imageRef)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeFalse())
		})
	})
	When("the image is not in RelatedImages", func() {
		It("should still pass", func() {
			csvContents := `kind: ClusterServiceVersion
apiVersion: operators.coreos.com/v1alpha1
spec:
  install:
    spec:
      deployments:
      - spec:
          template:
            spec:
              containers:
              - image: registry.example.io/foo/bar@sha256:f000432f07cd187469f0310e3ed9dcf9a5db2be14b8bab9c5293dd1ee8518176
                name: the-operator
  relatedImages:
  - name: the-proxy
    image: registry.example.io/foo/proxy@sha256:5e33f9d095952866b9743cc8268fb740cce6d93439f00ce333a2de1e5974837e`
			Expect(os.WriteFile(filepath.Join(imageRef.ImageFSPath, manifestsDir, clusterServiceVersionFilename), []byte(csvContents), 0o644)).To(Succeed())
			result, err := relatedImagesCheck.Validate(context.TODO(), imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})
	})
	AssertMetaData(relatedImagesCheck)
})
