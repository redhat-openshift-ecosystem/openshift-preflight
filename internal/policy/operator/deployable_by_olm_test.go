package operator

import (
	"context"
	"os"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/openshift"

	fakecranev1 "github.com/google/go-containerregistry/pkg/v1/fake"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("DeployableByOLMCheck", func() {
	var (
		deployableByOLMCheck DeployableByOlmCheck
		imageRef             image.ImageReference
		testcontext          context.Context
		clientBuilder        *fake.ClientBuilder
	)

	BeforeEach(func() {
		// override default timeout
		subscriptionTimeout = 1 * time.Second
		csvTimeout = 1 * time.Second

		fakeImage := fakecranev1.FakeImage{}
		imageRef.ImageInfo = &fakeImage
		imageRef.ImageFSPath = "./testdata/all_namespaces"

		now := metav1.Now()
		og.Status.LastUpdated = &now
		deployableByOLMCheck = *NewDeployableByOlmCheck("test_indeximage", "", "")
		scheme := apiruntime.NewScheme()
		Expect(openshift.AddSchemes(scheme)).To(Succeed())
		clientBuilder = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(&csvDefault, &csvMarketplace, &ns, &secret, &sub, &og).
			WithLists(&pods, &isList)

		deployableByOLMCheck.client = clientBuilder.
			WithObjects(&csv).
			Build()

		// Temp artifacts dir
		tmpDir, err := os.MkdirTemp("", "deployable-by-olm-*")
		Expect(err).ToNot(HaveOccurred())
		aw, err := artifacts.NewFilesystemWriter(artifacts.WithDirectory(tmpDir))
		Expect(err).ToNot(HaveOccurred())
		testcontext = artifacts.ContextWithWriter(context.Background(), aw)
		DeferCleanup(os.RemoveAll, tmpDir)
	})

	Describe("When deploying an operator using OLM", func() {
		Context("When the only supported install mode is AllNamespaces", func() {
			It("Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(testcontext, imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When the supported install modes are OwnNamespace and SingleNamespace", func() {
			BeforeEach(func() {
				imageRef.ImageFSPath = "./testdata/own_namespace"

				// changing the namespace since OwnNamespace operators CSV get applied to `InstallNamespace`
				ownCSV := csv.DeepCopy()
				ownCSV.Namespace = "testPackage"

				deployableByOLMCheck.client = clientBuilder.
					WithObjects(ownCSV).
					Build()
			})
			It("OperatorGroup should use InstallNamespace and Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(testcontext, imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When the only supported install mode is SingleNamespace", func() {
			BeforeEach(func() {
				imageRef.ImageFSPath = "./testdata/single_namespace"
			})
			It("OperatorGroup should use InstallNamespace and Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(testcontext, imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When the only supported install mode is MultiNamespace", func() {
			BeforeEach(func() {
				imageRef.ImageFSPath = "./testdata/multi_namespace"
			})
			It("OperatorGroup should use InstallNamespace and Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(testcontext, imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When installedCSV field of Subscription is not set", func() {
			BeforeEach(func() {
				badSub := sub
				Expect(deployableByOLMCheck.client.Get(testcontext, crclient.ObjectKey{
					Name:      "testPackage",
					Namespace: "testPackage",
				}, &badSub)).To(Succeed())
				badSub.Status.InstalledCSV = ""
				Expect(deployableByOLMCheck.client.Update(testcontext, &badSub, &crclient.UpdateOptions{})).To(Succeed())
			})
			It("Should fail Validate", func() {
				ok, err := deployableByOLMCheck.Validate(testcontext, imageRef)
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("When index image is in a custom namespace and CSV has been created successfully", func() {
			BeforeEach(func() {
				deployableByOLMCheck.indexImage = "image-registry.openshift-image-registry.svc/namespace/indeximage:v0.0.0"
			})
			It("Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(testcontext, imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When index image is in a private registry and CSV has been created successfully", func() {
			BeforeEach(func() {
				// dockerconfig.json is just an empty file. It just needs to exist.
				deployableByOLMCheck.dockerConfig = "./testdata/dockerconfig.json"
			})
			It("Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(testcontext, imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When the non-default channel is being tested", func() {
			BeforeEach(func() {
				deployableByOLMCheck.channel = "non-default-channel"
			})
			It("Should pass Validate", func() {
				ok, err := deployableByOLMCheck.Validate(testcontext, imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
	})

	AssertMetaData(&deployableByOLMCheck)

	DescribeTable("Image Registry validation",
		func(bundleImages []string, expected bool) {
			ok := checkImageSource(context.Background(), bundleImages)
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
})
