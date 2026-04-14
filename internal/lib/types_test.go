package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path"
	"strings"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/viper"
)

var _ = Describe("Pyxis Client Instantiation", func() {
	Context("When establishing a pyxis client.", func() {
		Context("with none of the required values", func() {
			It("Should return a nil pyxis client", func() {
				pc := NewPyxisClient(context.TODO(), "", "", "")
				Expect(pc).To(BeNil())
			})
		})

		Context("Missing any of the required values", func() {
			It("Should return a nil pyxis client", func() {
				pc := NewPyxisClient(context.TODO(), "projectID", "", "host")
				Expect(pc).To(BeNil())

				pc = NewPyxisClient(context.TODO(), "projectID", "token", "")
				Expect(pc).To(BeNil())

				pc = NewPyxisClient(context.TODO(), "", "token", "host")
				Expect(pc).To(BeNil())
			})
		})

		Context("With all the required values", func() {
			It("should return a pyxis client", func() {
				pc := NewPyxisClient(context.TODO(), "projectID", "token", "host")
				Expect(pc).ToNot(BeNil())
			})
		})
	})
})

var _ = Describe("Policy Resolution", func() {
	Context("When determining container policy exceptions", func() {
		It("should return a scratch policy exception if the project has type flag", func() {
			certProject := &pyxis.CertProject{
				Container: pyxis.Container{
					Type: "scratch",
				},
			}
			p := GetContainerPolicyExceptions(certProject)
			Expect(p).To(Equal(policy.PolicyScratchNonRoot))
		})

		It("should return a scratch policy exception if the project has os_content_type flag", func() {
			certProject := &pyxis.CertProject{
				Container: pyxis.Container{
					OsContentType: "Scratch Image",
				},
			}
			p := GetContainerPolicyExceptions(certProject)
			Expect(p).To(Equal(policy.PolicyScratchNonRoot))
		})

		It("should return a root policy exception if the project has the flag", func() {
			certProject := &pyxis.CertProject{
				Container: pyxis.Container{
					DockerConfigJSON: "",
					Privileged:       true,
				},
			}
			p := GetContainerPolicyExceptions(certProject)
			Expect(p).To(Equal(policy.PolicyRoot))
		})

		It("should return a scratch plus root policy exception if the project has the flag", func() {
			certProject := &pyxis.CertProject{
				Container: pyxis.Container{
					DockerConfigJSON: "",
					OsContentType:    "Scratch Image",
					Privileged:       true,
				},
			}
			p := GetContainerPolicyExceptions(certProject)
			Expect(p).To(Equal(policy.PolicyScratchRoot))
		})

		It("should return a container policy exception if the project has no exceptions", func() {
			certProject := &pyxis.CertProject{
				Container: pyxis.Container{
					Type:       "",
					Privileged: false,
				},
			}
			p := GetContainerPolicyExceptions(certProject)
			Expect(p).To(Equal(policy.PolicyContainer))
		})

		It("should return a container policy exception if called with nil container project", func() {
			p := GetContainerPolicyExceptions(nil)
			Expect(p).To(Equal(policy.PolicyContainer))
		})
	})
})

var _ = Describe("Submitter Resolution", func() {
	Context("When resolving the submitter", func() {
		Context("with a valid pyxis client", func() {
			pc := NewPyxisClient(context.TODO(), "projectID", "token", "host")
			Expect(pc).ToNot(BeNil())

			It("should return a containerCertificationSubmitter", func() {
				submitter := ResolveSubmitter(pc, "projectID", "dockerconfig", "logfile")
				typed, ok := submitter.(*ContainerCertificationSubmitter)
				Expect(typed).ToNot(BeNil())
				Expect(ok).To(BeTrue())
			})
		})

		Context("With no pyxis client", func() {
			It("should return a no-op submitter", func() {
				submitter := ResolveSubmitter(nil, "", "", "")
				typed, ok := submitter.(*NoopSubmitter)
				Expect(typed).ToNot(BeNil())
				Expect(ok).To(BeTrue())
			})
		})
	})
})

var _ = Describe("The NoopSubmitter", func() {
	Context("When using the noop submitter", func() {
		var bf *bytes.Buffer
		var noop *NoopSubmitter

		BeforeEach(func() {
			bf = bytes.NewBuffer([]byte{})
			bufferLogger := logr.Logger{}.WithSink(log.NewBufferSink(bf))

			noop = NewNoopSubmitter(false, &bufferLogger)
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
})

var _ = Describe("BuildConnectURL", func() {
	Context("when pyxis_env is not set", func() {
		It("should return the prod URL", func() {
			viper.Instance().Set("pyxis_env", "")
			DeferCleanup(func() { viper.Instance().Set("pyxis_env", "") })
			url := BuildConnectURL("12345")
			Expect(url).To(Equal("https://connect.redhat.com/component/view/12345"))
		})
	})

	Context("when pyxis_env is set to prod", func() {
		It("should return the prod URL", func() {
			viper.Instance().Set("pyxis_env", "prod")
			DeferCleanup(func() { viper.Instance().Set("pyxis_env", "") })
			url := BuildConnectURL("12345")
			Expect(url).To(Equal("https://connect.redhat.com/component/view/12345"))
		})
	})

	Context("when pyxis_env is set to a non-prod value", func() {
		It("should return the environment-specific URL", func() {
			viper.Instance().Set("pyxis_env", "stage")
			DeferCleanup(func() { viper.Instance().Set("pyxis_env", "") })
			url := BuildConnectURL("12345")
			Expect(url).To(Equal("https://connect.stage.redhat.com/component/view/12345"))
		})
	})
})

var _ = Describe("Container Certification Submitter", func() {
	Context("When using the containerCertificationSubmitter", func() {
		var sbmt *ContainerCertificationSubmitter
		var fakePC *FakePyxisClient
		var dockerConfigPath string
		var preflightLogPath string
		var tmpdir string
		var aw *artifacts.FilesystemWriter
		var testcontext context.Context

		preflightLogFilename := "preflight.log"
		dockerconfigFilename := "dockerconfig.json"
		BeforeEach(func() {
			var err error
			tmpdir, err = os.MkdirTemp("", "libtypes-tests-*")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(os.RemoveAll, tmpdir)
			aw, err = artifacts.NewFilesystemWriter(artifacts.WithDirectory(tmpdir))
			Expect(err).ToNot(HaveOccurred())

			testcontext = artifacts.ContextWithWriter(context.Background(), aw)

			dockerConfigPath = path.Join(aw.Path(), dockerconfigFilename)
			preflightLogPath = path.Join(aw.Path(), preflightLogFilename)
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

			preflightTestResultsJSONBytes, err := json.Marshal(certification.Results{
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
			Expect(aw.WriteFile(dockerconfigFilename, strings.NewReader("dockerconfig")))
			Expect(aw.WriteFile(preflightLogFilename, strings.NewReader("preflight log")))
			Expect(aw.WriteFile(check.DefaultCertImageFilename, bytes.NewReader(certImageJSONBytes)))
			Expect(aw.WriteFile(check.DefaultTestResultsFilename, bytes.NewReader(preflightTestResultsJSONBytes)))
			Expect(aw.WriteFile(check.DefaultRPMManifestFilename, bytes.NewReader(rpmManifestJSONBytes)))
		})

		Context("and project cannot be obtained from the API", func() {
			BeforeEach(func() {
				fakePC.getProjectsFunc = gpFuncReturnError
			})
			It("should throw an error", func() {
				err := sbmt.Submit(testcontext)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("and the provided docker config cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(dockerConfigPath)
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(testcontext)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(dockerconfigFilename))
			})
		})

		Context("and no docker config command argument was provided", func() {
			BeforeEach(func() {
				fakePC.setSRFuncSubmitSuccessfully("", "")
				fakePC.getProjectsFunc = gpFuncReturnScratchException
			})
			It("should not throw an error", func() {
				sbmt.DockerConfig = ""
				err := os.Remove(dockerConfigPath)
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(testcontext)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("and certProject.Container.hosted_registry=true", func() {
			BeforeEach(func() {
				fakePC.setSRFuncSubmitSuccessfully("", "")
				fakePC.getProjectsFunc = gpFuncReturnHostedRegistry
			})
			It("should not throw an error", func() {
				err := sbmt.Submit(testcontext)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("and the artifact writer is missing from the context", func() {
			It("should return an error about missing artifact writer", func() {
				ctxNoWriter := context.Background()
				err := sbmt.Submit(ctxNoWriter)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("artifact writer was either missing or was not supported"))
			})
		})

		Context("and the artifact writer is not a FilesystemWriter", func() {
			It("should return an error about unsupported artifact writer", func() {
				mapWriter, err := artifacts.NewMapWriter()
				Expect(err).ToNot(HaveOccurred())
				ctxMapWriter := artifacts.ContextWithWriter(context.Background(), mapWriter)
				err = sbmt.Submit(ctxMapWriter)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("artifact writer was either missing or was not supported"))
			})
		})

		Context("and the cert image cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(path.Join(aw.Path(), check.DefaultCertImageFilename))
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(testcontext)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(check.DefaultCertImageFilename))
			})
		})

		Context("and the preflight results cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(path.Join(aw.Path(), check.DefaultTestResultsFilename))
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(testcontext)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(check.DefaultTestResultsFilename))
			})
		})

		Context("and the rpmManifest cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(path.Join(aw.Path(), check.DefaultRPMManifestFilename))
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(testcontext)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(check.DefaultRPMManifestFilename))
			})
		})

		Context("and scratch policy was executed, so no rpmManifest exists on disk", func() {
			BeforeEach(func() {
				fakePC.setSRFuncSubmitSuccessfully("12345", "12345")
				fakePC.getProjectsFunc = gpFuncReturnScratchException
			})
			It("should not throw an error", func() {
				err := os.Remove(path.Join(aw.Path(), check.DefaultRPMManifestFilename))
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(testcontext)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("and the preflight logfile cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(preflightLogPath)
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(testcontext)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(preflightLogFilename))
			})
		})

		Context("and the submission fails", func() {
			BeforeEach(func() {
				fakePC.submitResultsFunc = srFuncReturnError
				fakePC.getProjectsFunc = gpFuncReturnScratchException
			})

			It("should throw an error", func() {
				err := sbmt.Submit(testcontext)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("and the certproject returned from pyxis is nil, but no error was returned", func() {
			BeforeEach(func() {
				fakePC.getProjectsFunc = gpFuncNoop
			})

			It("should throw an error", func() {
				err := sbmt.Submit(testcontext)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("and one of the submission artifacts is malformed", func() {
			JustBeforeEach(func() {
				afs := afero.NewBasePathFs(afero.NewOsFs(), aw.Path())
				Expect(afs.Remove(check.DefaultRPMManifestFilename)).To(Succeed())
				Expect(aw.WriteFile(check.DefaultRPMManifestFilename, strings.NewReader("malformed"))).To(ContainSubstring(check.DefaultRPMManifestFilename))
			})

			It("should throw an error finalizing the submission", func() {
				err := sbmt.Submit(testcontext)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unable to finalize data"))
			})
		})

		Context("and the submission succeeds", func() {
			BeforeEach(func() {
				fakePC.setSRFuncSubmitSuccessfully("", "")
				fakePC.getProjectsFunc = gpFuncReturnScratchException
			})
			It("should not throw an error", func() {
				err := sbmt.Submit(testcontext)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
