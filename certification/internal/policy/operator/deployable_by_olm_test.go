package operator

import (
	"os"
	"path/filepath"

	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
)

var _ = Describe("DeployableByOLMCheck", func() {
	var (
		deployableByOLMCheck DeployableByOlmCheck
		engine               cli.OpenshiftEngine
		imageRef             certification.ImageReference
	)
	const (
		metadataDir        = "metadata"
		annotationFilename = "annotations.yaml"
		annotations        = `annotations:
  operators.operatorframework.io.bundle.package.v1: testPackage
  operators.operatorframework.io.bundle.channel.default.v1: testChannel
`
	)
	BeforeEach(func() {
		// override default timeout
		subscriptionTimeout = 1
		csvTimeout = 1

		// mock bundle directory
		tmpDir, err := os.MkdirTemp("", "bundle-metadata-*")
		Expect(err).ToNot(HaveOccurred())

		err = os.Mkdir(filepath.Join(tmpDir, metadataDir), 0755)
		Expect(err).ToNot(HaveOccurred())

		err = os.WriteFile(filepath.Join(tmpDir, metadataDir, annotationFilename), []byte(annotations), 0644)
		Expect(err).ToNot(HaveOccurred())

		fakeImage := fakecranev1.FakeImage{}
		imageRef.ImageInfo = &fakeImage
		imageRef.ImageFSPath = tmpDir

		// set env var for index image
		os.Setenv("PFLT_INDEXIMAGE", "test_indeximage")
		os.Setenv("PFLT_ARTIFACTS", tmpDir)
	})
	Describe("When deploying an operator using OLM", func() {
		Context("When CSV has been created successfully", func() {
			BeforeEach(func() {
				engine = FakeOpenshiftEngine{}
				deployableByOLMCheck = *NewDeployableByOlmCheck(&engine)
			})

			It("Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When installedCSV field of Subscription is not set", func() {
			BeforeEach(func() {
				engine = BadOpenshiftEngine{}
				deployableByOLMCheck = *NewDeployableByOlmCheck(&engine)
			})

			It("Should fail Validate", func() {
				ok, err := deployableByOLMCheck.Validate(imageRef)
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
	AfterEach(func() {
		err := os.RemoveAll(imageRef.ImageFSPath)
		Expect(err).ToNot(HaveOccurred())

	})
})
