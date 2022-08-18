package operator

import (
	"context"
	"os"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/openshift"

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
		imageRef             certification.ImageReference
	)

	BeforeEach(func() {
		// override default timeout
		subscriptionTimeout = 1 * time.Second
		csvTimeout = 1 * time.Second

		fakeImage := fakecranev1.FakeImage{}
		imageRef.ImageInfo = &fakeImage
		imageRef.ImageFSPath = "./testdata/valid_bundle"

		now := metav1.Now()
		og.Status.LastUpdated = &now
		deployableByOLMCheck = *NewDeployableByOlmCheck("test_indeximage", "", "")
		scheme := apiruntime.NewScheme()
		Expect(openshift.AddSchemes(scheme)).To(Succeed())
		var client crclient.Client //nolint:gosimple // explicitly make var with interface
		client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(&csv, &csvDefault, &csvMarketplace, &ns, &secret, &sub, &og).
			WithLists(&pods, &isList).
			Build()
		deployableByOLMCheck.client = client

		// Temp artifacts dir
		tmpDir, err := os.MkdirTemp("", "deployable-by-olm-*")
		Expect(err).ToNot(HaveOccurred())
		artifacts.SetDir(tmpDir)
		DeferCleanup(os.RemoveAll, tmpDir)
		DeferCleanup(artifacts.Reset)
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
				Expect(deployableByOLMCheck.client.Get(context.TODO(), crclient.ObjectKey{
					Name:      "testPackage",
					Namespace: "testPackage",
				}, &badSub)).To(Succeed())
				badSub.Status.InstalledCSV = ""
				Expect(deployableByOLMCheck.client.Update(context.TODO(), &badSub, &crclient.UpdateOptions{})).To(Succeed())
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
				// dockerconfig.json is just an empty file. It just needs to exist.
				deployableByOLMCheck.dockerConfig = "./testdata/dockerconfig.json"
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

	AssertMetaData(&deployableByOLMCheck)

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
})
