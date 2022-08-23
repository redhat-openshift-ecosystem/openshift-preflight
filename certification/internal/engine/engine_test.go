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

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Execute Checks tests", func() {
	var src string
	var engine CraneEngine
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
		artifacts.SetDir(tmpDir)

		goodCheck := certification.NewGenericCheck(
			"testcheck",
			func(context.Context, certification.ImageReference) (bool, error) {
				return true, nil
			},
			certification.Metadata{},
			certification.HelpText{},
		)

		errorCheck := certification.NewGenericCheck(
			"errorCheck",
			func(context.Context, certification.ImageReference) (bool, error) {
				return false, errors.New("errorCheck")
			},
			certification.Metadata{},
			certification.HelpText{},
		)

		failedCheck := certification.NewGenericCheck(
			"failedCheck",
			func(context.Context, certification.ImageReference) (bool, error) {
				return false, nil
			},
			certification.Metadata{},
			certification.HelpText{},
		)

		optionalCheckPassing := certification.NewGenericCheck(
			"optionalCheckPassing",
			func(context.Context, certification.ImageReference) (bool, error) {
				return true, nil
			},
			certification.Metadata{Level: "optional"},
			certification.HelpText{},
		)

		optionalCheckFailing := certification.NewGenericCheck(
			"optionalCheckFailing",
			func(context.Context, certification.ImageReference) (bool, error) {
				return false, fmt.Errorf("optionalError")
			},
			certification.Metadata{Level: "optional"},
			certification.HelpText{},
		)

		emptyConfig := runtime.Config{}
		engine = CraneEngine{
			Config: emptyConfig.ReadOnly(), // must pass a config to avoid nil pointer errors
			Image:  src,
			Checks: []certification.Check{
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
	AfterEach(func() { artifacts.Reset() }) // reset the artifacts dir back to defaults.
	Context("Run the checks", func() {
		It("should succeed", func() {
			err := engine.ExecuteChecks(context.TODO())
			Expect(err).ToNot(HaveOccurred())
			Expect(engine.results.Passed).To(HaveLen(1))
			Expect(engine.results.Failed).To(HaveLen(1))
			Expect(engine.results.Errors).To(HaveLen(1))
			Expect(engine.results.CertificationHash).To(BeEmpty())
		})
		Context("it is a bundle", func() {
			It("should succeed and generate a bundle hash", func() {
				engine.IsBundle = true
				err := engine.ExecuteChecks(context.TODO())
				Expect(err).ToNot(HaveOccurred())
				Expect(engine.results.CertificationHash).ToNot(BeEmpty())
			})
		})
		Context("the image is invalid", func() {
			It("should throw a crane error on pull", func() {
				engine.Image = "does.not/exist/anywhere:ever"
				err := engine.ExecuteChecks(context.TODO())
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
