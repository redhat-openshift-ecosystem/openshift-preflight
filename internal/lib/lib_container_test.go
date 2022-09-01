package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/pyxis"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var _ = Describe("Lib Container Functions", func() {
	BeforeEach(createAndCleanupDirForArtifactsAndLogs)

	Context("When determining container policy exceptions", func() {
		var fakePC *FakePyxisClient
		BeforeEach(func() {
			// reset the fake pyxis client before each execution
			// as a precaution.
			fakePC = &FakePyxisClient{
				findImagesByDigestFunc: fidbFuncNoop,
				getProjectsFunc:        gpFuncNoop,
				submitResultsFunc:      srFuncNoop,
			}
		})

		It("should throw an error if unable to get the project from the API", func() {
			fakePC.getProjectsFunc = gpFuncReturnError
			_, err := GetContainerPolicyExceptions(context.TODO(), fakePC)
			Expect(err).To(HaveOccurred())
		})

		It("should return a scratch policy exception if the project has the flag in the API", func() {
			fakePC.getProjectsFunc = gpFuncReturnScratchNonRootException
			p, err := GetContainerPolicyExceptions(context.TODO(), fakePC)
			Expect(p).To(Equal(policy.PolicyScratchNonRoot))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return a scratch and root policy exception if the project has the flag in the API", func() {
			fakePC.getProjectsFunc = gpFuncReturnScratchRootException
			p, err := GetContainerPolicyExceptions(context.TODO(), fakePC)
			Expect(p).To(Equal(policy.PolicyScratchRoot))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return a root policy exception if the project has the flag in the API", func() {
			fakePC.getProjectsFunc = gpFuncReturnRootException
			p, err := GetContainerPolicyExceptions(context.TODO(), fakePC)
			Expect(p).To(Equal(policy.PolicyRoot))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return a container policy exception if the project no exceptions in the API", func() {
			fakePC.getProjectsFunc = gpFuncReturnNoException
			p, err := GetContainerPolicyExceptions(context.TODO(), fakePC)
			Expect(p).To(Equal(policy.PolicyContainer))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When using the containerCertificationSubmitter", func() {
		var sbmt *ContainerCertificationSubmitter
		var fakePC *FakePyxisClient
		var dockerConfigPath string
		var preflightLogPath string

		preflightLogFilename := "preflight.log"
		dockerconfigFilename := "dockerconfig.json"
		BeforeEach(func() {
			dockerConfigPath = path.Join(artifacts.Path(), dockerconfigFilename)
			preflightLogPath = path.Join(artifacts.Path(), preflightLogFilename)
			// Normalize a FakePyxisClient with noop functions.
			fakePC = NewFakePyxisClientNoop()

			// Most tests will need a passing getProjects func so set that to
			// avoid having to perform multiple BeforeEaches
			fakePC.setGPFuncReturnBaseProject("")

			// configure the submitter
			sbmt = &ContainerCertificationSubmitter{
				CertificationProjectID: fakePC.baseProject("").ID,
				Pyxis:                  fakePC,
				DockerConfig:           dockerConfigPath,
				PreflightLogFile:       preflightLogPath,
			}

			certImageJSONBytes, err := json.Marshal(pyxis.CertImage{
				ID: "111111111111",
			})
			Expect(err).ToNot(HaveOccurred())

			preflightTestResultsJSONBytes, err := json.Marshal(runtime.Results{
				TestedImage:   "foo",
				PassedOverall: true,
			})
			Expect(err).ToNot(HaveOccurred())

			rpmManifestJSONBytes, err := json.Marshal(pyxis.RPMManifest{
				ID:      "foo",
				ImageID: "foo",
			})
			Expect(err).ToNot(HaveOccurred())

			// Create expected files. Use of Gomega's Expect here (without a subsequent test) is intentional.
			// Expect automatically checks that additional return values are nil, and thus will fail if they
			// are not.
			Expect(artifacts.WriteFile(dockerconfigFilename, strings.NewReader("dockerconfig")))
			Expect(artifacts.WriteFile(preflightLogFilename, strings.NewReader("preflight log")))
			Expect(artifacts.WriteFile(certification.DefaultCertImageFilename, bytes.NewReader(certImageJSONBytes)))
			Expect(artifacts.WriteFile(certification.DefaultTestResultsFilename, bytes.NewReader(preflightTestResultsJSONBytes)))
			Expect(artifacts.WriteFile(certification.DefaultRPMManifestFilename, bytes.NewReader(rpmManifestJSONBytes)))
		})

		Context("and project cannot be obtained from the API", func() {
			BeforeEach(func() {
				fakePC.getProjectsFunc = gpFuncReturnError
			})
			It("should throw an error", func() {
				err := sbmt.Submit(context.TODO())
				Expect(err).To(HaveOccurred())
			})
		})

		Context("and the provided docker config cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(dockerConfigPath)
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(dockerconfigFilename))
			})
		})

		Context("and no docker config command argument was provided", func() {
			BeforeEach(func() {
				fakePC.setSRFuncSubmitSuccessfully("", "")
			})
			It("should not throw an error", func() {
				sbmt.DockerConfig = ""
				err := os.Remove(dockerConfigPath)
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(context.TODO())
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("and certProject.Container.hosted_registry=true", func() {
			BeforeEach(func() {
				fakePC.setSRFuncSubmitSuccessfully("", "")
				fakePC.getProjectsFunc = gpFuncReturnHostedRegistry
			})
			It("should not throw an error", func() {
				err := sbmt.Submit(context.TODO())
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("and the cert image cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(path.Join(artifacts.Path(), certification.DefaultCertImageFilename))
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(certification.DefaultCertImageFilename))
			})
		})

		Context("and the preflight results cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(path.Join(artifacts.Path(), certification.DefaultTestResultsFilename))
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(certification.DefaultTestResultsFilename))
			})
		})

		Context("and the rpmManifest cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(path.Join(artifacts.Path(), certification.DefaultRPMManifestFilename))
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(certification.DefaultRPMManifestFilename))
			})
		})

		Context("and the preflight logfile cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(preflightLogPath)
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(preflightLogFilename))
			})
		})

		Context("and the submission fails", func() {
			BeforeEach(func() {
				fakePC.submitResultsFunc = srFuncReturnError
			})

			It("should throw an error", func() {
				err := sbmt.Submit(context.TODO())
				Expect(err).To(HaveOccurred())
			})
		})

		Context("and the certproject returned from pyxis is nil, but no error was returned", func() {
			BeforeEach(func() {
				fakePC.getProjectsFunc = gpFuncNoop
			})

			It("should throw an error", func() {
				err := sbmt.Submit(context.TODO())
				Expect(err).To(HaveOccurred())
			})
		})

		Context("and one of the submission artifacts is malformed", func() {
			JustBeforeEach(func() {
				afs := afero.NewBasePathFs(afero.NewOsFs(), artifacts.Path())
				Expect(afs.Remove(certification.DefaultRPMManifestFilename)).To(Succeed())
				Expect(artifacts.WriteFile(certification.DefaultRPMManifestFilename, strings.NewReader("malformed"))).To(ContainSubstring(certification.DefaultRPMManifestFilename))
			})

			It("should throw an error finalizing the submission", func() {
				err := sbmt.Submit(context.TODO())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unable to finalize data"))
			})
		})

		Context("and the submission succeeds", func() {
			BeforeEach(func() {
				fakePC.setSRFuncSubmitSuccessfully("", "")
			})
			It("should not throw an error", func() {
				err := sbmt.Submit(context.TODO())
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("When using the noop submitter", func() {
		var bf *bytes.Buffer
		var noop *NoopSubmitter

		BeforeEach(func() {
			bufferLogger := logrus.New()
			bf = bytes.NewBuffer([]byte{})
			bufferLogger.SetOutput(bf)

			noop = NewNoopSubmitter(false, bufferLogger)
		})

		Context("and enabling log emitting", func() {
			BeforeEach(func() {
				noop.SetEmitLog(true)
			})

			It("should include the reason in the emitted log if specified", func() {
				testReason := "test reason"
				noop.SetReason(testReason)
				err := noop.Submit(context.TODO())
				Expect(err).ToNot(HaveOccurred())
				Expect(bf.String()).To(ContainSubstring(testReason))
			})

			It("should emit logs when calling submit", func() {
				err := noop.Submit(context.TODO())
				Expect(err).ToNot(HaveOccurred())
				Expect(bf.String()).To(ContainSubstring("Results are not being sent for submission."))
			})
		})

		Context("and disabling log emitting", func() {
			It("should not emit logs when calling submit", func() {
				noop.SetEmitLog(false)
				err := noop.Submit(context.TODO())
				Expect(err).ToNot(HaveOccurred())
				Expect(bf.String()).To(BeEmpty())
			})
		})
	})

	Context("When resolving the submitter", func() {
		Context("with a valid pyxis client", func() {
			cfg := runtime.Config{
				CertificationProjectID: "projectid",
				PyxisHost:              "host",
				PyxisAPIToken:          "apitoken",
				DockerConfig:           "dockercfg",
				LogFile:                "logfile",
			}

			pc := NewPyxisClient(context.TODO(), cfg.ReadOnly())
			Expect(pc).ToNot(BeNil())

			It("should return a containerCertificationSubmitter", func() {
				submitter := ResolveSubmitter(pc, cfg.ReadOnly())
				typed, ok := submitter.(*ContainerCertificationSubmitter)
				Expect(typed).ToNot(BeNil())
				Expect(ok).To(BeTrue())
			})
		})

		Context("With no pyxis client", func() {
			cfg := runtime.Config{}
			It("should return a no-op submitter", func() {
				submitter := ResolveSubmitter(nil, cfg.ReadOnly())
				typed, ok := submitter.(*NoopSubmitter)
				Expect(typed).ToNot(BeNil())
				Expect(ok).To(BeTrue())
			})
		})
	})

	Context("When establishing a pyxis client.", func() {
		Context("with none of the required values", func() {
			cfgNoCertProjectID := runtime.Config{}

			It("Should return a nil pyxis client", func() {
				pc := NewPyxisClient(context.TODO(), cfgNoCertProjectID.ReadOnly())
				Expect(pc).To(BeNil())
			})
		})

		Context("Missing any of the required values", func() {
			cfgMissingCertProjectID := runtime.Config{
				PyxisHost:     "foo",
				PyxisAPIToken: "bar",
			}

			cfgMissingPyxisHost := runtime.Config{
				CertificationProjectID: "foo",
				PyxisAPIToken:          "bar",
			}

			cfgMissingPyxisAPIToken := runtime.Config{
				CertificationProjectID: "foo",
				PyxisHost:              "bar",
			}

			It("Should return a nil pyxis client", func() {
				pc := NewPyxisClient(context.TODO(), cfgMissingCertProjectID.ReadOnly())
				Expect(pc).To(BeNil())

				pc = NewPyxisClient(context.TODO(), cfgMissingPyxisHost.ReadOnly())
				Expect(pc).To(BeNil())

				pc = NewPyxisClient(context.TODO(), cfgMissingPyxisAPIToken.ReadOnly())
				Expect(pc).To(BeNil())
			})
		})

		Context("With all the required values", func() {
			cfgValid := runtime.Config{
				CertificationProjectID: "foo",
				PyxisHost:              "bar",
				PyxisAPIToken:          "baz",
			}

			It("should return a pyxis client", func() {
				pc := NewPyxisClient(context.TODO(), cfgValid.ReadOnly())
				Expect(pc).ToNot(BeNil())
			})
		})
	})

	Context("When instantiating a checkContainerRunner", func() {
		var cfg *runtime.Config

		BeforeEach(func() {
			cfg = &runtime.Config{
				Image:          "quay.io/example/foo:latest",
				ResponseFormat: formatters.DefaultFormat,
			}
		})

		Context("and the user passed the submit flag, but no credentials", func() {
			It("should return a noop submitter as credentials are required for submission", func() {
				runner, err := NewCheckContainerRunner(context.TODO(), cfg, false)
				Expect(err).ToNot(HaveOccurred())
				_, rsIsCorrectType := runner.Rs.(*NoopSubmitter)
				Expect(rsIsCorrectType).To(BeTrue())
			})
		})

		Context("with a valid policy formatter", func() {
			It("should return with no error, and the appropriate formatter", func() {
				cfg.ResponseFormat = "xml"
				runner, err := NewCheckContainerRunner(context.TODO(), cfg, false)
				Expect(err).ToNot(HaveOccurred())
				expectedFormatter, err := formatters.NewByName(cfg.ResponseFormat)
				Expect(err).ToNot(HaveOccurred())
				Expect(runner.Formatter.PrettyName()).To(Equal(expectedFormatter.PrettyName()))
			})
		})

		Context("with an invalid policy definition", func() {
			It("should return the container policy engine anyway", func() {
				runner, err := NewCheckContainerRunner(context.TODO(), cfg, false)
				Expect(err).ToNot(HaveOccurred())

				expectedEngine, err := engine.NewForConfig(context.TODO(), cfg.ReadOnly())
				Expect(runner.Eng).To(BeEquivalentTo(expectedEngine))
				Expect(err).ToNot(HaveOccurred())
			})
		})
		// NOTE(): There's no way to test policy exceptions here because
		// without valid credentials to pyxis.

		Context("with an invalid formatter definition", func() {
			It("should return an error", func() {
				cfg.ResponseFormat = "foo"
				_, err := NewCheckContainerRunner(context.TODO(), cfg, false)
				Expect(err).To(HaveOccurred())
			})
		})

		It("should contain a ResultWriterFile ResultWriter", func() {
			runner, err := NewCheckContainerRunner(context.TODO(), cfg, false)
			Expect(err).ToNot(HaveOccurred())
			_, rwIsExpectedType := runner.Rw.(*runtime.ResultWriterFile)
			Expect(rwIsExpectedType).To(BeTrue())
		})
	})

	Describe("JUnit", func() {
		var results *runtime.Results
		var junitfile string

		BeforeEach(func() {
			results = &runtime.Results{
				TestedImage:       "registry.example.com/example/image:0.0.1",
				PassedOverall:     true,
				TestedOn:          runtime.UnknownOpenshiftClusterVersion(),
				CertificationHash: "sha256:deadb33f",
				Passed:            []runtime.Result{},
				Failed:            []runtime.Result{},
				Errors:            []runtime.Result{},
			}
			junitfile = filepath.Join(artifacts.Path(), "results-junit.xml")
		})

		When("The additional JUnitXML results file is requested", func() {
			It("should be written to the artifacts directory without error", func() {
				Expect(writeJUnit(context.TODO(), *results)).To(Succeed())
				_, err := os.Stat(junitfile)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
