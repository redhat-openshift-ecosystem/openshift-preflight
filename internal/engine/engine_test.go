package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	goruntime "runtime"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Execute Checks tests", func() {
	var src string
	var engine CraneEngine
	var testcontext context.Context
	BeforeEach(func() {
		// Set up a fake registry.
		registryLogger := log.New(io.Discard, "", log.Ldate)
		s := httptest.NewServer(registry.New(registry.Logger(registryLogger)))
		DeferCleanup(func() {
			s.Close()
		})
		u, err := url.Parse(s.URL)
		Expect(err).ToNot(HaveOccurred())

		src = fmt.Sprintf("%s/test/crane", u.Host)

		// Expected values.
		img, err := random.Image(1024, 5)
		Expect(err).ToNot(HaveOccurred())

		err = crane.Push(img, src)
		Expect(err).ToNot(HaveOccurred())

		tmpDir, err := os.MkdirTemp("", "preflight-engine-test-*")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpDir)
		aw, err := artifacts.NewFilesystemWriter(artifacts.WithDirectory(tmpDir))
		Expect(err).ToNot(HaveOccurred())
		testcontext = artifacts.ContextWithWriter(context.Background(), aw)

		goodCheck := check.NewGenericCheck(
			"testcheck",
			func(context.Context, image.ImageReference) (bool, error) {
				return true, nil
			},
			check.Metadata{},
			check.HelpText{},
		)

		errorCheck := check.NewGenericCheck(
			"errorCheck",
			func(context.Context, image.ImageReference) (bool, error) {
				return false, errors.New("errorCheck")
			},
			check.Metadata{},
			check.HelpText{},
		)

		failedCheck := check.NewGenericCheck(
			"failedCheck",
			func(context.Context, image.ImageReference) (bool, error) {
				return false, nil
			},
			check.Metadata{},
			check.HelpText{},
		)

		optionalCheckPassing := check.NewGenericCheck(
			"optionalCheckPassing",
			func(context.Context, image.ImageReference) (bool, error) {
				return true, nil
			},
			check.Metadata{Level: "optional"},
			check.HelpText{},
		)

		optionalCheckFailing := check.NewGenericCheck(
			"optionalCheckFailing",
			func(context.Context, image.ImageReference) (bool, error) {
				return false, fmt.Errorf("optionalError")
			},
			check.Metadata{Level: "optional"},
			check.HelpText{},
		)

		emptyConfig := runtime.Config{}
		engine = CraneEngine{
			DockerConfig: emptyConfig.DockerConfig,
			Image:        src,
			Checks: []check.Check{
				goodCheck,
				errorCheck,
				failedCheck,
				optionalCheckPassing,
				optionalCheckFailing,
			},
			IsBundle:  false,
			IsScratch: false,
		}
	})
	Context("Run the checks", func() {
		It("should succeed", func() {
			err := engine.ExecuteChecks(testcontext)
			Expect(err).ToNot(HaveOccurred())
			Expect(engine.results.Passed).To(HaveLen(1))
			Expect(engine.results.Failed).To(HaveLen(1))
			Expect(engine.results.Errors).To(HaveLen(1))
			Expect(engine.results.CertificationHash).To(BeEmpty())
		})
		Context("it is a bundle", func() {
			It("should succeed and generate a bundle hash", func() {
				engine.IsBundle = true
				err := engine.ExecuteChecks(testcontext)
				Expect(err).ToNot(HaveOccurred())
				Expect(engine.results.CertificationHash).ToNot(BeEmpty())
			})
		})
		Context("the image is invalid", func() {
			It("should throw a crane error on pull", func() {
				engine.Image = "does.not/exist/anywhere:ever"
				err := engine.ExecuteChecks(testcontext)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

var _ = Describe("Source RPM name function", func() {
	Context("With a source rpm name", func() {
		Context("And a normal source rpm name", func() {
			It("should parse bash-5.1.8-2.el9.src.rpm to bash", func() {
				expected := "bash"
				actual := getBgName("bash-5.1.8-2.el9.src.rpm")
				Expect(actual).To(Equal(expected))
			})
		})
		Context("And a slightly annoying source rpm name", func() {
			It("should parse python3.9-3.9.6-6.el9.src.rpm to python3.9", func() {
				expected := "python3.9"
				actual := getBgName("python3.9-3.9.6-6.el9.src.rpm")
				Expect(actual).To(Equal(expected))
			})
		})
		Context("And a source rpm name with a bunch of -'s", func() {
			It("should parse python-pip-21.0.1-6.el9.src.rpm to bash", func() {
				expected := "python-pip"
				actual := getBgName("python-pip-21.0.1-6.el9.src.rpm")
				Expect(actual).To(Equal(expected))
			})
		})
	})
})

var _ = Describe("Tag and digest binding information function", func() {
	Context("with a digest as the user-provided identifier", func() {
		It("should return a message indicating that no tag will be associated", func() {
			m, _ := tagDigestBindingInfo("sha256:5031aedc52578c68277ef127ef0f2a941e12d280722f1c19ee83932b6efd2f3b", "sha256:5031aedc52578c68277ef127ef0f2a941e12d280722f1c19ee83932b6efd2f3b")
			Expect(m).To(ContainSubstring("You've provided an image by digest"))
		})
		Context("with a tag as the user-proivded identifier", func() {
			It("should return a message indicating the tag and digest are bound", func() {
				t := "mytag"
				d := "sha256:5031aedc52578c68277ef127ef0f2a941e12d280722f1c19ee83932b6efd2f3b"
				m, _ := tagDigestBindingInfo(t, d)
				Expect(m).To(ContainSubstring(fmt.Sprintf("This image's tag %s will be paired with digest %s", t, d)))
			})
		})
	})
})

var _ = Describe("CheckInitialization", func() {
	When("initializing the engine", func() {
		It("should not return an error", func() {
			_, err := New(context.TODO(), "example.com/some/image:latest", []check.Check{}, nil, "", false, false, false, goruntime.GOARCH)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("Check Initialization", func() {
	When("initializing container checks", func() {
		It("should properly return checks for default container policy", func() {
			_, err := InitializeContainerChecks(context.TODO(), policy.PolicyContainer, ContainerCheckConfig{})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should properly return checks for the scratch policy", func() {
			_, err := InitializeContainerChecks(context.TODO(), policy.PolicyScratch, ContainerCheckConfig{})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should properly return checks for the root policy", func() {
			_, err := InitializeContainerChecks(context.TODO(), policy.PolicyRoot, ContainerCheckConfig{})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should throw an error if the policy is unknown", func() {
			_, err := InitializeContainerChecks(context.TODO(), policy.Policy("foo"), ContainerCheckConfig{})
			Expect(err).To(HaveOccurred())
		})
	})

	When("initializing operator checks", func() {
		It("should properly return checks for the root policy", func() {
			_, err := InitializeOperatorChecks(context.TODO(), policy.PolicyOperator, OperatorCheckConfig{})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should throw an error if the policy is unknown", func() {
			_, err := InitializeOperatorChecks(context.TODO(), policy.Policy("bar"), OperatorCheckConfig{})
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("Check Name Queries", func() {
	DescribeTable("The checks associated with valid policy should return the expected check names",
		func(queryFunc func(context.Context) []string, expected []string) {
			c := queryFunc(context.TODO())
			Expect(queryFunc(context.TODO())).To(ContainElements(expected))
			Expect(len(c)).To(Equal(len(expected)))
		},
		Entry("default container policy", ContainerPolicy, []string{
			"HasLicense",
			"HasUniqueTag",
			"LayerCountAcceptable",
			"HasNoProhibitedPackages",
			"HasRequiredLabel",
			"RunAsNonRoot",
			"HasModifiedFiles",
			"BasedOnUbi",
		}),
		Entry("default operator policy", OperatorPolicy, []string{
			"ScorecardBasicSpecCheck",
			"ScorecardOlmSuiteCheck",
			"DeployableByOLM",
			"ValidateOperatorBundle",
			"BundleImageRefsAreCertified",
			"SecurityContextConstraintsInCSV",
			"AllImageRefsInRelatedImages",
		}),
		Entry("scratch container policy", ScratchContainerPolicy, []string{
			"HasLicense",
			"HasUniqueTag",
			"LayerCountAcceptable",
			"HasRequiredLabel",
			"RunAsNonRoot",
		}),
		Entry("root container policy", RootExceptionContainerPolicy, []string{
			"HasLicense",
			"HasUniqueTag",
			"LayerCountAcceptable",
			"HasNoProhibitedPackages",
			"HasRequiredLabel",
			"HasModifiedFiles",
			"BasedOnUbi",
		}),
	)

	When("the policy is unknown", func() {
		It("should return an empty list", func() {
			c := checkNamesFor(context.TODO(), policy.Policy("does not exist"))
			Expect(c).To(Equal([]string{}))
		})
	})
})
