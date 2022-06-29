package operator

import (
	"context"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/openshift"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/operatorsdk"
)

var _ = Describe("DeployableByOLMCheck", func() {
	var (
		deployableByOLMCheck DeployableByOlmCheck
		fakeEngine           operatorSdk
		imageRef             certification.ImageReference
		tmpDockerDir         string
		client               crclient.Client
	)
	const (
		metadataDir            = "metadata"
		manifestDir            = "manifests"
		registryConfigDir      = ".docker"
		annotationFilename     = "annotations.yaml"
		csvFilename            = "test-operator.clusterserviceversion.yaml"
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

		csvStr = `
    spec:
      installModes:
        - supported: false
          type: OwnNamespace
        - supported: false
          type: SingleNamespace
        - supported: false
          type: MultiNamespace
        - supported: true
          type: AllNamespaces
`
	)
	BeforeEach(func() {
		// override default timeout
		subscriptionTimeout = 1 * time.Second
		csvTimeout = 1 * time.Second

		// mock bundle directory
		tmpDir, err := os.MkdirTemp("", "bundle-metadata-*")
		Expect(err).ToNot(HaveOccurred())

		Expect(os.Mkdir(filepath.Join(tmpDir, metadataDir), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(tmpDir, metadataDir, annotationFilename), []byte(annotations), 0o644)).To(Succeed())

		// mock csv file
		Expect(os.Mkdir(filepath.Join(tmpDir, manifestDir), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(tmpDir, manifestDir, csvFilename), []byte(csvStr), 0o644)).To(Succeed())

		// mock docker config file
		tmpDockerDir, err = os.MkdirTemp("", "docker-config-*")
		Expect(err).ToNot(HaveOccurred())

		Expect(os.Mkdir(filepath.Join(tmpDockerDir, registryConfigDir), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(
			tmpDockerDir,
			registryConfigDir,
			registryConfigFilename),
			[]byte(registryAuthToken),
			0o644)).To(Succeed())

		fakeImage := fakecranev1.FakeImage{}
		imageRef.ImageInfo = &fakeImage
		imageRef.ImageFSPath = tmpDir

		report := operatorsdk.OperatorSdkBundleValidateReport{
			Passed:  true,
			Outputs: []operatorsdk.OperatorSdkBundleValidateOutput{},
		}
		fakeEngine = FakeOperatorSdk{
			OperatorSdkBVReport: report,
		}

		now := metav1.Now()
		og.Status.LastUpdated = &now
		deployableByOLMCheck = *NewDeployableByOlmCheck(fakeEngine, "test_indeximage", "", "")
		scheme := apiruntime.NewScheme()
		Expect(openshift.AddSchemes(scheme)).To(Succeed())
		client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(&csv, &csvDefault, &csvMarketplace, &ns, &secret, &sub, &og).
			WithLists(&pods, &isList).
			Build()
		deployableByOLMCheck.client = client

		artifacts.SetDir(tmpDir)
	})
	AfterEach(func() {
		artifacts.Reset()
	})
	Describe("When deploying an operator using OLM", func() {
		Context("When CSV has been created successfully", func() {
			It("Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When installedCSV field of Subscription is not set", func() {
			BeforeEach(func() {
				badSub := sub
				Expect(client.Get(context.TODO(), crclient.ObjectKey{
					Name:      "testPackage",
					Namespace: "testPackage",
				}, &badSub)).To(Succeed())
				badSub.Status.InstalledCSV = ""
				Expect(client.Update(context.TODO(), &badSub, &crclient.UpdateOptions{})).To(Succeed())
			})
			It("Should fail Validate", func() {
				ok, err := deployableByOLMCheck.Validate(context.TODO(), imageRef)
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("When index image is in a custom namespace and CSV has been created successfully", func() {
			BeforeEach(func() {
				deployableByOLMCheck.indexImage = "image-registry.openshift-image-registry.svc/namespace/indeximage:v0.0.0"
			})
			It("Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When index image is in a private registry and CSV has been created successfully", func() {
			BeforeEach(func() {
				deployableByOLMCheck.dockerConfig = filepath.Join(tmpDockerDir, registryConfigDir, registryConfigFilename)
			})
			It("Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When the only supported install mode is AllNamespaces", func() {
			It("Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When the non-default channel is being tested", func() {
			BeforeEach(func() {
				deployableByOLMCheck.channel = "non-default-channel"
			})
			It("Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
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
