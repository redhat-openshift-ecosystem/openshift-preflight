package operator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type certifiedImageFinder struct{}

func (c *certifiedImageFinder) FindImagesByDigest(ctx context.Context, digests []string) ([]pyxis.CertImage, error) {
	var matchingImages []pyxis.CertImage
	matchingImages = append(matchingImages, pyxis.CertImage{
		Certified: true,
		Repositories: []pyxis.Repository{
			{
				Registry:   "registry.example.io",
				Repository: "foo/bar",
			},
		},
		DockerImageDigest: "sha256:f000432f07cd187469f0310e3ed9dcf9a5db2be14b8bab9c5293dd1ee8518176",
	})
	matchingImages = append(matchingImages, pyxis.CertImage{
		Certified: true,
		Repositories: []pyxis.Repository{
			{
				Registry:   "registry.example.io",
				Repository: "foo/proxy",
			},
		},
		DockerImageDigest: "sha256:5e33f9d095952866b9743cc8268fb740cce6d93439f00ce333a2de1e5974837e",
	})
	return matchingImages, nil
}

type uncertifiedImageFinder struct{}

func (c *uncertifiedImageFinder) FindImagesByDigest(ctx context.Context, digests []string) ([]pyxis.CertImage, error) {
	var matchingImages []pyxis.CertImage
	matchingImages = append(matchingImages, pyxis.CertImage{
		Certified: false,
		Repositories: []pyxis.Repository{
			{
				Registry:   "registry.example.io",
				Repository: "foo/bar",
			},
		},
		DockerImageDigest: "sha256:f000432f07cd187469f0310e3ed9dcf9a5db2be14b8bab9c5293dd1ee8518176",
	})
	matchingImages = append(matchingImages, pyxis.CertImage{
		Certified: true,
		Repositories: []pyxis.Repository{
			{
				Registry:   "registry.example.io",
				Repository: "foo/proxy",
			},
		},
		DockerImageDigest: "sha256:5e33f9d095952866b9743cc8268fb740cce6d93439f00ce333a2de1e5974837e",
	})
	return matchingImages, nil
}

type missingImageFinder struct{}

func (c *missingImageFinder) FindImagesByDigest(ctx context.Context, digests []string) ([]pyxis.CertImage, error) {
	var matchingImages []pyxis.CertImage
	matchingImages = append(matchingImages, pyxis.CertImage{
		Certified: true,
		Repositories: []pyxis.Repository{
			{
				Registry:   "registry.example.io",
				Repository: "foo/bar",
			},
		},
		DockerImageDigest: "sha256:f000432f07cd187469f0310e3ed9dcf9a5db2be14b8bab9c5293dd1ee8518176",
	})
	return matchingImages, nil
}

type badCertifiedImageFinder struct{}

func (c *badCertifiedImageFinder) FindImagesByDigest(ctx context.Context, digests []string) ([]pyxis.CertImage, error) {
	return nil, fmt.Errorf("Pyxis error")
}

var _ = Describe("CertifiedImages", func() {
	const (
		manifestsDir                  = "manifests"
		clusterServiceVersionFilename = "myoperator.clusterserviceversion.yaml"
	)
	var (
		certifiedImagesCheck *certifiedImagesCheck
		imageRef             image.ImageReference
		csvContents          = `kind: ClusterServiceVersion
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
		certifiedImagesCheck = NewCertifiedImagesCheck(&certifiedImageFinder{})
		tmpDir, err := os.MkdirTemp("", "certified-images-bundle-*")
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
			result, err := certifiedImagesCheck.Validate(context.TODO(), imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
			Expect(certifiedImagesCheck.nonCertifiedImages).To(HaveLen(0))
		})
	})
	When("an image in the CSV is not certified", func() {
		AfterEach(func() {
			certifiedImagesCheck.imageFinder = &certifiedImageFinder{}
		})
		It("should still succeed", func() {
			certifiedImagesCheck.imageFinder = &uncertifiedImageFinder{}
			result, err := certifiedImagesCheck.Validate(context.TODO(), imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
			Expect(certifiedImagesCheck.nonCertifiedImages).To(HaveLen(1))
		})
	})
	When("an image in the CSV is not in Pyxis", func() {
		AfterEach(func() {
			certifiedImagesCheck.imageFinder = &certifiedImageFinder{}
		})
		It("should still succeed", func() {
			certifiedImagesCheck.imageFinder = &missingImageFinder{}
			result, err := certifiedImagesCheck.Validate(context.TODO(), imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
			Expect(certifiedImagesCheck.nonCertifiedImages).To(HaveLen(1))
		})
	})
	When("there is no CSV", func() {
		It("should fail", func() {
			Expect(os.RemoveAll(imageRef.ImageFSPath)).To(Succeed())
			result, err := certifiedImagesCheck.Validate(context.TODO(), imageRef)
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
			result, err := certifiedImagesCheck.Validate(context.TODO(), imageRef)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeFalse())
		})
	})
	When("the images in the CSV aren't pinned", func() {
		It("should succeed, but mark the image as non-certified", func() {
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
              - image: registry.example.io/foo/bar:latest
                name: the-operator`
			Expect(os.WriteFile(filepath.Join(imageRef.ImageFSPath, manifestsDir, clusterServiceVersionFilename), []byte(csvContents), 0o644)).To(Succeed())
			result, err := certifiedImagesCheck.Validate(context.TODO(), imageRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
			Expect(certifiedImagesCheck.nonCertifiedImages).To(HaveLen(1))
		})
	})
	When("Pyxis has an error", func() {
		AfterEach(func() {
			certifiedImagesCheck.imageFinder = &certifiedImageFinder{}
		})
		It("should fail", func() {
			certifiedImagesCheck.imageFinder = &badCertifiedImageFinder{}
			result, err := certifiedImagesCheck.Validate(context.TODO(), imageRef)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeFalse())
		})
	})
	AssertMetaData(certifiedImagesCheck)
})
