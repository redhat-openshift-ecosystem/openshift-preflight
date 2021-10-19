package operator

import (
	"os"
	"path/filepath"
	"time"

	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
)

var _ = Describe("DeployableByOLMCheck", func() {
	var (
		deployableByOLMCheck DeployableByOlmCheck
		engine               cli.OpenshiftEngine
		imageRef             certification.ImageReference
		tmpDockerDir         string
	)
	const (
		metadataDir            = "metadata"
		registryConfigDir      = ".docker"
		annotationFilename     = "annotations.yaml"
		registryConfigFilename = "config.json"
		annotations            = `annotations:
  operators.operatorframework.io.bundle.package.v1: testPackage
  operators.operatorframework.io.bundle.channel.default.v1: testChannel
`
		registryAuthToken = `{
"auths": {
  "quay.io": {
    "auth": "auth-token-test"
    }
  }
}`
	)
	BeforeEach(func() {
		// override default timeout
		subscriptionTimeout = 1 * time.Second
		csvTimeout = 1 * time.Second

		// mock bundle directory
		tmpDir, err := os.MkdirTemp("", "bundle-metadata-*")
		Expect(err).ToNot(HaveOccurred())

		err = os.Mkdir(filepath.Join(tmpDir, metadataDir), 0755)
		Expect(err).ToNot(HaveOccurred())

		err = os.WriteFile(filepath.Join(tmpDir, metadataDir, annotationFilename), []byte(annotations), 0644)
		Expect(err).ToNot(HaveOccurred())

		// mock docker config file
		tmpDockerDir, err = os.MkdirTemp("", "docker-config-*")
		Expect(err).ToNot(HaveOccurred())

		err = os.Mkdir(filepath.Join(tmpDockerDir, registryConfigDir), 0755)
		Expect(err).ToNot(HaveOccurred())

		err = os.WriteFile(filepath.Join(tmpDockerDir, registryConfigDir, registryConfigFilename), []byte(registryAuthToken), 0644)
		Expect(err).ToNot(HaveOccurred())

		fakeImage := fakecranev1.FakeImage{}
		imageRef.ImageInfo = &fakeImage
		imageRef.ImageFSPath = tmpDir

		// set env var for index image
		os.Setenv("PFLT_INDEXIMAGE", "test_indeximage")
		os.Setenv("PFLT_ARTIFACTS", tmpDir)
		os.Setenv("PFLT_DOCKERCONFIG", filepath.Join(tmpDockerDir, registryConfigDir, registryConfigFilename))

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
	DescribeTable("Image Registry validation",
		func(bundleImages []string, expected bool) {
			ok := checkImageSource(bundleImages)
			Expect(ok).To(Equal(expected))
		},
		Entry("registry.connect.dev.redhat.com", []string{"registry.connect.dev.redhat.com/"}, true),
		Entry("registry.connect.qa.redhat.com", []string{"registry.connect.qa.redhat.com/"}, true),
		Entry("registry.connect.stage.redhat.com", []string{"registry.connect.stage.redhat.com/"}, true),
		Entry("registry.connect.redhat.com", []string{"registry.connect.redhat.com"}, true),
		Entry("registry.redhat.io", []string{"registry.redhat.io"}, true),
		Entry("registry.access.redhat.com", []string{"registry.access.redhat.com/ubi8/ubi"}, true),
		Entry("quay.io", []string{"quay.io/rocrisp/preflight-operator-bundle:v1"}, false),
	)
	AfterEach(func() {
		err := os.RemoveAll(imageRef.ImageFSPath)
		Expect(err).ToNot(HaveOccurred())

		err = os.RemoveAll(tmpDockerDir)
		Expect(err).ToNot(HaveOccurred())
	})
})
