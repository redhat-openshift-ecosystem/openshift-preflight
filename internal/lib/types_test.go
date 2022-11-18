package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/policy"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/pyxis"
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
			fakePC.getProjectsFunc = gpFuncReturnScratchException
			p, err := GetContainerPolicyExceptions(context.TODO(), fakePC)
			Expect(p).To(Equal(policy.PolicyScratch))
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
			Expect(aw.WriteFile(certification.DefaultCertImageFilename, bytes.NewReader(certImageJSONBytes)))
			Expect(aw.WriteFile(certification.DefaultTestResultsFilename, bytes.NewReader(preflightTestResultsJSONBytes)))
			Expect(aw.WriteFile(certification.DefaultRPMManifestFilename, bytes.NewReader(rpmManifestJSONBytes)))
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

		Context("and the cert image cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(path.Join(aw.Path(), certification.DefaultCertImageFilename))
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(testcontext)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(certification.DefaultCertImageFilename))
			})
		})

		Context("and the preflight results cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(path.Join(aw.Path(), certification.DefaultTestResultsFilename))
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(testcontext)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(certification.DefaultTestResultsFilename))
			})
		})

		Context("and the rpmManifest cannot be read from disk", func() {
			It("should throw an error", func() {
				err := os.Remove(path.Join(aw.Path(), certification.DefaultRPMManifestFilename))
				Expect(err).ToNot(HaveOccurred())

				err = sbmt.Submit(testcontext)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(certification.DefaultRPMManifestFilename))
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
				Expect(afs.Remove(certification.DefaultRPMManifestFilename)).To(Succeed())
				Expect(aw.WriteFile(certification.DefaultRPMManifestFilename, strings.NewReader("malformed"))).To(ContainSubstring(certification.DefaultRPMManifestFilename))
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
			})
			It("should not throw an error", func() {
				err := sbmt.Submit(testcontext)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
