package engine

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"net/url"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Execute Checks tests", func() {
	var src string
	var engine craneEngine
	var testcontext context.Context
	var s *httptest.Server
	var u *url.URL
	BeforeEach(func() {
		// Set up a fake registry.
		registryLogger := log.New(io.Discard, "", log.Ldate)
		s = httptest.NewServer(registry.New(registry.Logger(registryLogger)))
		DeferCleanup(func() {
			s.Close()
		})

		var err error
		u, err = url.Parse(s.URL)
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
			nil,
		)

		errorCheck := check.NewGenericCheck(
			"errorCheck",
			func(context.Context, image.ImageReference) (bool, error) {
				return false, errors.New("errorCheck")
			},
			check.Metadata{},
			check.HelpText{},
			nil,
		)

		failedCheck := check.NewGenericCheck(
			"failedCheck",
			func(context.Context, image.ImageReference) (bool, error) {
				return false, nil
			},
			check.Metadata{},
			check.HelpText{},
			nil,
		)

		optionalCheckPassing := check.NewGenericCheck(
			"optionalCheckPassing",
			func(context.Context, image.ImageReference) (bool, error) {
				return true, nil
			},
			check.Metadata{Level: "optional"},
			check.HelpText{},
			nil,
		)

		optionalCheckFailing := check.NewGenericCheck(
			"optionalCheckFailing",
			func(context.Context, image.ImageReference) (bool, error) {
				return false, fmt.Errorf("optionalError")
			},
			check.Metadata{Level: "optional"},
			check.HelpText{},
			nil,
		)

		warningCheckPassing := check.NewGenericCheck(
			"warnCheckPassing",
			func(context.Context, image.ImageReference) (bool, error) {
				return true, nil
			},
			check.Metadata{Level: check.LevelWarn},
			check.HelpText{},
			nil,
		)

		warningCheckFailing := check.NewGenericCheck(
			"warnCheckFailing",
			func(context.Context, image.ImageReference) (bool, error) {
				return false, nil
			},
			check.Metadata{Level: check.LevelWarn},
			check.HelpText{},
			nil,
		)

		emptyConfig := runtime.Config{}
		engine = craneEngine{
			dockerConfig: emptyConfig.DockerConfig,
			image:        src,
			checks: []check.Check{
				goodCheck,
				errorCheck,
				failedCheck,
				optionalCheckPassing,
				optionalCheckFailing,
				warningCheckPassing,
				warningCheckFailing,
			},
			isBundle:  false,
			isScratch: false,
		}
	})
	Context("Run the checks", func() {
		It("should succeed", func() {
			err := engine.ExecuteChecks(testcontext)
			Expect(err).ToNot(HaveOccurred())
			Expect(engine.results.Passed).To(HaveLen(2))
			Expect(engine.results.Failed).To(HaveLen(1))
			Expect(engine.results.Errors).To(HaveLen(1))
			Expect(engine.results.Warned).To(HaveLen(1))
			Expect(engine.results.CertificationHash).To(BeEmpty())
		})
		Context("it is a bundle", func() {
			It("should succeed and generate a bundle hash", func() {
				engine.isBundle = true
				err := engine.ExecuteChecks(testcontext)
				Expect(err).ToNot(HaveOccurred())
				Expect(engine.results.CertificationHash).ToNot(BeEmpty())
			})
		})
		Context("the image is invalid", func() {
			It("should throw a crane error on pull", func() {
				engine.image = "does.not/exist/anywhere:ever"
				err := engine.ExecuteChecks(testcontext)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("it is a bundle made with GNU tar layer", func() {
			BeforeEach(func() {
				var buf bytes.Buffer

				err := writeTarball(&buf, []byte("mycontent"), "myfile", 10)
				Expect(err).ToNot(HaveOccurred())

				layer := static.NewLayer(buf.Bytes(), types.MediaType("application/vnd.docker.image.rootfs.diff.tar"))
				img, err := mutate.AppendLayers(empty.Image, layer)
				Expect(err).ToNot(HaveOccurred())

				src = fmt.Sprintf("%s/test/crane", u.Host)

				err = crane.Push(img, src)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should just hang forever err and hash mean nothing", func() {
				engine.isBundle = true
				err := engine.ExecuteChecks(testcontext)
				Expect(err).ToNot(HaveOccurred())
				Expect(engine.results.CertificationHash).ToNot(BeEmpty())
			})
		})
		Context("it is a bundle made with tar layer", func() {
			BeforeEach(func() {
				engine.isBundle = true

				var buf bytes.Buffer

				err := writeTarball(&buf, []byte("mycontent"), "myfile", 0)
				Expect(err).ToNot(HaveOccurred())

				layer := static.NewLayer(buf.Bytes(), types.MediaType("application/vnd.docker.image.rootfs.diff.tar"))
				img, err := mutate.AppendLayers(empty.Image, layer)
				Expect(err).ToNot(HaveOccurred())

				src = fmt.Sprintf("%s/test/crane", u.Host)

				err = crane.Push(img, src)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should succeed and generate a bundle hash", func() {
				engine.isBundle = true
				err := engine.ExecuteChecks(testcontext)
				Expect(err).ToNot(HaveOccurred())
				Expect(engine.results.CertificationHash).ToNot(BeEmpty())
			})
		})
		Context("it is a bundle made and one of the layers is not a tar", func() {
			BeforeEach(func() {
				engine.isBundle = true

				want := []byte(`{"foo":"bar"}`)
				layer := static.NewLayer(want, types.MediaType("application/json"))
				img, err := mutate.AppendLayers(empty.Image, layer)
				Expect(err).ToNot(HaveOccurred())

				src = fmt.Sprintf("%s/test/crane", u.Host)

				err = crane.Push(img, src)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should throw an EOF error on untar", func() {
				engine.isBundle = true
				err := engine.ExecuteChecks(testcontext)
				Expect(err).To(HaveOccurred())
				Expect(engine.results.CertificationHash).To(BeEmpty())
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
			cfg := runtime.Config{}
			_, err := New(context.TODO(), []check.Check{}, nil, cfg)
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
			_, err := InitializeContainerChecks(context.TODO(), policy.PolicyScratchNonRoot, ContainerCheckConfig{})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should properly return checks for the scratch and root policy", func() {
			_, err := InitializeContainerChecks(context.TODO(), policy.PolicyScratchRoot, ContainerCheckConfig{})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should properly return checks for the root policy", func() {
			_, err := InitializeContainerChecks(context.TODO(), policy.PolicyRoot, ContainerCheckConfig{})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should properly return checks for the konflux policy", func() {
			_, err := InitializeContainerChecks(context.TODO(), policy.PolicyKonflux, ContainerCheckConfig{})
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
			"HasNoProhibitedLabels",
			"RunAsNonRoot",
			"HasModifiedFiles",
			"BasedOnUbi",
			"HasProhibitedContainerName",
		}),
		Entry("default operator policy", OperatorPolicy, []string{
			"ScorecardBasicSpecCheck",
			"ScorecardOlmSuiteCheck",
			"DeployableByOLM",
			"ValidateOperatorBundle",
			"BundleImageRefsAreCertified",
			"SecurityContextConstraintsInCSV",
			"AllImageRefsInRelatedImages",
			"FollowsRestrictedNetworkEnablementGuidelines",
			"RequiredAnnotations",
		}),
		Entry("scratch nonroot container policy", ScratchNonRootContainerPolicy, []string{
			"HasLicense",
			"HasUniqueTag",
			"LayerCountAcceptable",
			"HasRequiredLabel",
			"HasNoProhibitedLabels",
			"RunAsNonRoot",
			"HasProhibitedContainerName",
		}),
		Entry("scratch root container policy", ScratchRootContainerPolicy, []string{
			"HasLicense",
			"HasUniqueTag",
			"LayerCountAcceptable",
			"HasRequiredLabel",
			"HasNoProhibitedLabels",
			"HasProhibitedContainerName",
		}),
		Entry("root container policy", RootExceptionContainerPolicy, []string{
			"HasLicense",
			"HasUniqueTag",
			"LayerCountAcceptable",
			"HasNoProhibitedPackages",
			"HasRequiredLabel",
			"HasNoProhibitedLabels",
			"HasModifiedFiles",
			"BasedOnUbi",
			"HasProhibitedContainerName",
		}),
		Entry("konflux container policy", KonfluxContainerPolicy, []string{
			"HasLicense",
			"HasUniqueTag",
			"LayerCountAcceptable",
			"HasNoProhibitedPackages",
			"HasRequiredLabel",
			"RunAsNonRoot",
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

// writeTarball writes a tar archive to out with filename containing contents at the base path
// with extra bytes written at the end of length extraBytes.
// note: this should only be used as a helper function in tests
func writeTarball(out io.Writer, contents []byte, filename string, extraBytes uint) error {
	tw := tar.NewWriter(out)
	defer tw.Close()

	header := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     filename,
		Size:     int64(len(contents)),
		Mode:     420,
		Format:   tar.FormatPAX,
	}

	err := tw.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(tw, bytes.NewReader(contents))
	if err != nil {
		return err
	}

	if extraBytes > 0 {
		extra := make([]byte, extraBytes)
		_, err = out.Write(extra)
		if err != nil {
			return err
		}
	}

	return nil
}
