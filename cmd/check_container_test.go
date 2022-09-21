package cmd

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
	"github.com/spf13/viper"
)

var _ = Describe("Check Container Command", func() {
	BeforeEach(createAndCleanupDirForArtifactsAndLogs)

	Context("when running the check container subcommand", func() {
		Context("With all of the required parameters", func() {
			It("should reach the core logic, but throw an error because of the placeholder values for the container image", func() {
				_, err := executeCommand(checkContainerCmd(), "example.com/example/image:mytag")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("When determining container policy exceptions", func() {
		var fakePC *fakePyxisClient
		BeforeEach(func() {
			// reset the fake pyxis client before each execution
			// as a precaution.
			fakePC = &fakePyxisClient{
				findImagesByDigestFunc: fidbFuncNoop,
				getProjectsFunc:        gpFuncNoop,
				submitResultsFunc:      srFuncNoop,
			}
		})

		It("should throw an error if unable to get the project from the API", func() {
			fakePC.getProjectsFunc = gpFuncReturnError
			_, err := getContainerPolicyExceptions(context.TODO(), fakePC)
			Expect(err).To(HaveOccurred())
		})

		It("should return a scratch policy exception if the project has the flag in the API", func() {
			fakePC.getProjectsFunc = gpFuncReturnScratchException
			p, err := getContainerPolicyExceptions(context.TODO(), fakePC)
			Expect(p).To(Equal(policy.PolicyScratch))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return a root policy exception if the project has the flag in the API", func() {
			fakePC.getProjectsFunc = gpFuncReturnRootException
			p, err := getContainerPolicyExceptions(context.TODO(), fakePC)
			Expect(p).To(Equal(policy.PolicyRoot))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return a container policy exception if the project no exceptions in the API", func() {
			fakePC.getProjectsFunc = gpFuncReturnNoException
			p, err := getContainerPolicyExceptions(context.TODO(), fakePC)
			Expect(p).To(Equal(policy.PolicyContainer))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When using the containerCertificationSubmitter", func() {
		var sbmt *containerCertificationSubmitter
		var fakePC *fakePyxisClient
		var dockerConfigPath string
		var preflightLogPath string

		preflightLogFilename := "preflight.log"
		dockerconfigFilename := "dockerconfig.json"
		BeforeEach(func() {
			dockerConfigPath = path.Join(artifacts.Path(), dockerconfigFilename)
			preflightLogPath = path.Join(artifacts.Path(), preflightLogFilename)
			// Normalize a fakePyxisClient with noop functions.
			fakePC = &fakePyxisClient{
				findImagesByDigestFunc: fidbFuncNoop,
				getProjectsFunc:        gpFuncNoop,
				submitResultsFunc:      srFuncNoop,
			}

			// Most tests will need a passing getProjects func so set that to
			// avoid having to perform multiple BeforeEaches
			fakePC.setGPFuncReturnBaseProject("")

			// configure the submitter
			sbmt = &containerCertificationSubmitter{
				certificationProjectID: fakePC.baseProject("").ID,
				pyxis:                  fakePC,
				dockerConfig:           dockerConfigPath,
				preflightLogFile:       preflightLogPath,
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
				sbmt.dockerConfig = ""
				err := os.Remove(dockerConfigPath)
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(context.TODO())
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
		var noop *noopSubmitter

		BeforeEach(func() {
			bufferLogger := logrus.New()
			bf = bytes.NewBuffer([]byte{})
			bufferLogger.SetOutput(bf)

			noop = &noopSubmitter{log: bufferLogger}
		})

		Context("and enabling log emitting", func() {
			BeforeEach(func() {
				noop.emitLog = true
			})

			It("should include the reason in the emitted log if specified", func() {
				testReason := "test reason"
				noop.reason = testReason
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
				noop.emitLog = false
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

			pc := newPyxisClient(context.TODO(), cfg.ReadOnly())
			Expect(pc).ToNot(BeNil())

			It("should return a containerCertificationSubmitter", func() {
				submitter := resolveSubmitter(pc, cfg.ReadOnly())
				typed, ok := submitter.(*containerCertificationSubmitter)
				Expect(typed).ToNot(BeNil())
				Expect(ok).To(BeTrue())
			})
		})

		Context("With no pyxis client", func() {
			cfg := runtime.Config{}
			It("should return a no-op submitter", func() {
				submitter := resolveSubmitter(nil, cfg.ReadOnly())
				typed, ok := submitter.(*noopSubmitter)
				Expect(typed).ToNot(BeNil())
				Expect(ok).To(BeTrue())
			})
		})
	})

	Context("When establishing a pyxis client.", func() {
		Context("with none of the required values", func() {
			cfgNoCertProjectID := runtime.Config{}

			It("Should return a nil pyxis client", func() {
				pc := newPyxisClient(context.TODO(), cfgNoCertProjectID.ReadOnly())
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
				pc := newPyxisClient(context.TODO(), cfgMissingCertProjectID.ReadOnly())
				Expect(pc).To(BeNil())

				pc = newPyxisClient(context.TODO(), cfgMissingPyxisHost.ReadOnly())
				Expect(pc).To(BeNil())

				pc = newPyxisClient(context.TODO(), cfgMissingPyxisAPIToken.ReadOnly())
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
				pc := newPyxisClient(context.TODO(), cfgValid.ReadOnly())
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
				runner, err := newCheckContainerRunner(context.TODO(), cfg)
				Expect(err).ToNot(HaveOccurred())
				_, rsIsCorrectType := runner.rs.(*noopSubmitter)
				Expect(rsIsCorrectType).To(BeTrue())
			})
		})

		Context("and the user did not pass the submit flag", func() {
			var origSubmitValue bool
			BeforeEach(func() {
				origSubmitValue = submit
				submit = false
			})

			AfterEach(func() {
				submit = origSubmitValue
			})
			It("should return a noopSubmitter resultSubmitter", func() {
				runner, err := newCheckContainerRunner(context.TODO(), cfg)
				Expect(err).ToNot(HaveOccurred())
				_, rsIsCorrectType := runner.rs.(*noopSubmitter)
				Expect(rsIsCorrectType).To(BeTrue())
			})
		})

		Context("with a valid policy formatter", func() {
			It("should return with no error, and the appropriate formatter", func() {
				cfg.ResponseFormat = "xml"
				runner, err := newCheckContainerRunner(context.TODO(), cfg)
				Expect(err).ToNot(HaveOccurred())
				expectedFormatter, err := formatters.NewByName(cfg.ResponseFormat)
				Expect(err).ToNot(HaveOccurred())
				Expect(runner.formatter.PrettyName()).To(Equal(expectedFormatter.PrettyName()))
			})
		})

		Context("with an invalid policy definition", func() {
			It("should return the container policy engine anyway", func() {
				runner, err := newCheckContainerRunner(context.TODO(), cfg)
				Expect(err).ToNot(HaveOccurred())

				expectedEngine, err := engine.NewForConfig(context.TODO(), cfg.ReadOnly())
				Expect(runner.eng).To(BeEquivalentTo(expectedEngine))
				Expect(err).ToNot(HaveOccurred())
			})
		})
		// NOTE(): There's no way to test policy exceptions here because
		// without valid credentials to pyxis.

		Context("with an invalid formatter definition", func() {
			It("should return an error", func() {
				cfg.ResponseFormat = "foo"
				_, err := newCheckContainerRunner(context.TODO(), cfg)
				Expect(err).To(HaveOccurred())
			})
		})

		It("should contain a ResultWriterFile resultWriter", func() {
			runner, err := newCheckContainerRunner(context.TODO(), cfg)
			Expect(err).ToNot(HaveOccurred())
			_, rwIsExpectedType := runner.rw.(*runtime.ResultWriterFile)
			Expect(rwIsExpectedType).To(BeTrue())
		})
	})

	Context("When validating check container arguments and flags", func() {
		Context("and the user provided more than 1 positional arg", func() {
			It("should fail to run", func() {
				_, err := executeCommand(checkContainerCmd(), "foo", "bar")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("and the user provided less than 1 positional arg", func() {
			It("should fail to run", func() {
				_, err := executeCommand(checkContainerCmd())
				Expect(err).To(HaveOccurred())
			})
		})

		DescribeTable("and the user has enabled the submit flag",
			func(errString string, args []string) {
				out, err := executeCommand(checkContainerCmd(), args...)
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring(errString))
			},
			Entry("certification-project-id and pyxis-api-token are not supplied", "certification Project ID must be specified when --submit is present", []string{"--submit", "foo"}),
			Entry("pyxis-api-token is not supplied", "pyxis API Token must be specified when --submit is present", []string{"foo", "--submit", "--certification-project-id=fooid"}),
			Entry("certification-project-id is not supplied", "certification Project ID must be specified when --submit is present", []string{"--submit", "foo", "--pyxis-api-token=footoken"}),
			Entry("pyxis-api-token flag is present but empty because of '='", "cannot be empty when --submit is present", []string{"foo", "--submit", "--certification-project-id=fooid", "--pyxis-api-token="}),
			Entry("certification-project-id flag is present but empty because of '='", "cannot be empty when --submit is present", []string{"foo", "--submit", "--certification-project-id=", "--pyxis-api-token=footoken"}),
			Entry("submit is passed after empty api token", "pyxis API token and certification ID are required when --submit is present", []string{"foo", "--certification-project-id=fooid", "--pyxis-api-token", "--submit"}),
			Entry("submit is passed with explicit value after empty api token", "pyxis API token and certification ID are required when --submit is present", []string{"foo", "--certification-project-id=fooid", "--pyxis-api-token", "--submit=true"}),
		)

		When("the user enables the submit flag", func() {
			When("environment variables are used for certification ID and api token", func() {
				BeforeEach(func() {
					os.Setenv("PFLT_CERTIFICATION_PROJECT_ID", "certid")
					os.Setenv("PFLT_PYXIS_API_TOKEN", "tokenid")
					DeferCleanup(os.Unsetenv, "PFLT_CERTIFICATION_PROJECT_ID")
					DeferCleanup(os.Unsetenv, "PFLT_PYXIS_API_TOKEN")
				})
				It("should still execute with no error", func() {
					submit = true

					err := checkContainerPositionalArgs(checkContainerCmd(), []string{"foo"})
					Expect(err).ToNot(HaveOccurred())
					Expect(viper.GetString("pyxis_api_token")).To(Equal("tokenid"))
					Expect(viper.GetString("certification_project_id")).To(Equal("certid"))
				})
			})
			When("a config file is used", func() {
				BeforeEach(func() {
					config := `pyxis_api_token: mytoken
certification_project_id: mycertid`
					tempDir, err := os.MkdirTemp("", "check-container-submit-*")
					Expect(err).ToNot(HaveOccurred())
					err = os.WriteFile(filepath.Join(tempDir, "config.yaml"), bytes.NewBufferString(config).Bytes(), 0o644)
					Expect(err).ToNot(HaveOccurred())
					viper.AddConfigPath(tempDir)
					DeferCleanup(os.RemoveAll, tempDir)
				})
				It("should still execute with no error", func() {
					// Make sure that we've read the config file
					initConfig()
					submit = true

					err := checkContainerPositionalArgs(checkContainerCmd(), []string{"foo"})
					Expect(err).ToNot(HaveOccurred())
					Expect(viper.GetString("pyxis_api_token")).To(Equal("mytoken"))
					Expect(viper.GetString("certification_project_id")).To(Equal("mycertid"))
				})
			})
		})
	})

	Context("When validating the certification-project-id flag", func() {
		Context("and the flag is set properly", func() {
			BeforeEach(func() {
				viper.Set("certification_project_id", "123456789")
			})
			It("should not change the flag value", func() {
				err := validateCertificationProjectID(checkContainerCmd(), []string{"foo"})
				Expect(err).ToNot(HaveOccurred())
				Expect(viper.GetString("certification_project_id")).To(Equal("123456789"))
			})
		})
		Context("and a valid ospid format is provided", func() {
			BeforeEach(func() {
				viper.Set("certification_project_id", "ospid-123456789")
			})
			It("should strip ospid- from the flag value", func() {
				err := validateCertificationProjectID(checkContainerCmd(), []string{"foo"})
				Expect(err).ToNot(HaveOccurred())
				Expect(viper.GetString("certification_project_id")).To(Equal("123456789"))
			})
		})
		Context("and a legacy format with ospid is provided", func() {
			BeforeEach(func() {
				viper.Set("certification_project_id", "ospid-62423-f26c346-6cc1dc7fae92")
			})
			It("should throw an error", func() {
				err := validateCertificationProjectID(checkContainerCmd(), []string{"foo"})
				Expect(err).To(HaveOccurred())
			})
		})
		Context("and a legacy format without ospid is provided", func() {
			BeforeEach(func() {
				viper.Set("certification_project_id", "62423-f26c346-6cc1dc7fae92")
			})
			It("should throw an error", func() {
				err := validateCertificationProjectID(checkContainerCmd(), []string{"foo"})
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
