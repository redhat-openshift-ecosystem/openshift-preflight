package operator

import (
	"os"
	"path/filepath"
	"time"

	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("DeployableByOLMCheck", func() {
	var (
		deployableByOLMCheck DeployableByOlmCheck
		k8sclient            client.Client
		imageRef             certification.ImageReference
		tmpDockerDir         string
		scheme               *runtime.Scheme
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

		fakeImage := fakecranev1.FakeImage{}
		imageRef.ImageInfo = &fakeImage
		imageRef.ImageFSPath = tmpDir

		scheme = runtime.NewScheme()
		operatorv1alpha1.AddToScheme(scheme)
		operatorv1.AddToScheme(scheme)
		corev1.AddToScheme(scheme)
		rbacv1.AddToScheme(scheme)

		// set artifacts to tmpDir
		viper.Set("artifacts", tmpDir)
	})
	Describe("When deploying an operator using OLM", func() {
		Context("When CSV has been created successfully", func() {
			BeforeEach(func() {
				csv := &operatorv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "testPackage",
						Name:      "testPackage",
					},
					Spec: operatorv1alpha1.ClusterServiceVersionSpec{},
					Status: operatorv1alpha1.ClusterServiceVersionStatus{
						Phase: operatorv1alpha1.CSVPhaseSucceeded,
					},
				}
				sub := &operatorv1alpha1.Subscription{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "testPackage",
						Name:      "testPackage",
					},
					Status: operatorv1alpha1.SubscriptionStatus{
						InstalledCSV: "testPackage",
					},
				}
				objects := []runtime.Object{
					csv,
					sub,
				}
				k8sclient = fakeclient.NewClientBuilder().
					WithScheme(scheme).
					WithRuntimeObjects(objects...).
					Build()
				deployableByOLMCheck = *NewDeployableByOlmCheck(&k8sclient)
			})

			It("Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
			Context("When index image is in a custom namespace and CSV has been created successfully", func() {
				BeforeEach(func() {
					os.Setenv("PFLT_INDEXIMAGE", "image-registry.openshift-image-registry.svc/namespace/indeximage:v0.0.0")
				})

				It("Should pass Validate", func() {
					ok, err := deployableByOLMCheck.Validate(imageRef)
					Expect(err).ToNot(HaveOccurred())
					Expect(ok).To(BeTrue())
				})
			})
		})
		Context("When installedCSV field of Subscription is not set", func() {
			BeforeEach(func() {
				csv := &operatorv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "testPackage",
						Name:      "testPackage",
					},
				}
				objects := []runtime.Object{
					csv,
				}
				k8sclient = fakeclient.NewClientBuilder().
					WithRuntimeObjects(objects...).
					WithScheme(scheme).
					Build()
				deployableByOLMCheck = *NewDeployableByOlmCheck(&k8sclient)
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
