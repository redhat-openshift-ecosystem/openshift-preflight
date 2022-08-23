package cmd

import (
	"context"
	"os"
	"path"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/engine"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Preflight Check Func", func() {
	Context("When running the preflight check logic", func() {
		// This test customizes the artifactsDir in testing functions,
		// so we set a custom temp dir outside of the top-level just for
		// this test.
		var localTempDir string
		var localArtifactsDir string

		var cfg *runtime.Config
		var pc pyxisClient
		var eng engine.CheckEngine
		var fmttr formatters.ResponseFormatter
		var rw resultWriter
		var rs resultSubmitter

		BeforeEach(func() {
			// instantiate err to make sure we can equal-assign in the following line.
			var err error
			localTempDir, err = os.MkdirTemp(os.TempDir(), "preflight-check-local-tempdir-*")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(localTempDir)).ToNot(BeZero())
			localArtifactsDir = path.Join(localTempDir, "artifacts")
			// Don't set the artifacts dir here! This is handled by the function under test.

			img := "quay.io/example/foo:latest"
			// create a base config
			cfg = &runtime.Config{
				Image:     img,
				Artifacts: localArtifactsDir,
			}

			pc = &fakePyxisClient{
				findImagesByDigestFunc: fidbFuncNoop,
				getProjectsFunc:        gpFuncNoop,
				submitResultsFunc:      srFuncNoop,
			}

			eng = fakeCheckEngine{
				image:  img,
				passed: true,
			}

			fmttr, _ = formatters.NewByName(formatters.DefaultFormat)
			rw = &runtime.ResultWriterFile{}
			rs = &noopSubmitter{}

			DeferCleanup(os.RemoveAll, localTempDir)
			DeferCleanup(os.RemoveAll, localArtifactsDir)
			DeferCleanup(artifacts.Reset)
		})

		Context("with a customized artifacts directory", func() {
			It("should set the artifacts directory accordingly", func() {
				// it's possible this will throw an error, but we dont' care for this test.
				_ = preflightCheck(context.TODO(), cfg, pc, eng, fmttr, rw, rs)
				Expect(artifacts.Path()).To(Equal(localArtifactsDir))
			})
		})

		Context("and the results file fails to open", func() {
			BeforeEach(func() {
				rw = &badResultWriter{errmsg: "some result writer error"}
			})

			It("should throw an error", func() {
				err := preflightCheck(context.TODO(), cfg, pc, eng, fmttr, rw, rs)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("some result writer error"))
			})
		})

		Context("with an engine that encounters an error while executing checks", func() {
			var msg string
			BeforeEach(func() {
				msg = "some internal engine error"
				eng = fakeCheckEngine{errorRunningChecks: true, errorMsg: msg}
			})
			It("should thrown an error", func() {
				err := preflightCheck(context.TODO(), cfg, pc, eng, fmttr, rw, rs)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(msg))
			})
		})

		Context("with a formatter that cannot properly format the results", func() {
			var msg string
			BeforeEach(func() {
				msg = "some error formatting results"
				fmttr = &badFormatter{errormsg: msg}
			})

			It("should throw an error", func() {
				err := preflightCheck(context.TODO(), cfg, pc, eng, fmttr, rw, rs)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(msg))
			})
		})

		Context("and the user has requested JUnit output", func() {
			BeforeEach(func() {
				cfg.WriteJUnit = true
			})
			It("should write a junit file in the artifacts directory", func() {
				err := preflightCheck(context.TODO(), cfg, pc, eng, fmttr, rw, rs)
				Expect(err).ToNot(HaveOccurred())
				Expect(path.Join(artifacts.Path(), "results-junit.xml")).To(BeAnExistingFile())
			})
		})

		Context("and submission encounters an error", func() {
			var msg string
			BeforeEach(func() {
				msg = "some error submitting"
				rs = &badResultSubmitter{errmsg: msg}
				// TODO(): This is the package level variable, and isn't fantastic to have to evaluate in tests.
				// It would make sense to rely solely on the cfg.Submit value instead of the global variable.
				cfg.Submit = true
			})

			It("should throw an error", func() {
				err := preflightCheck(context.TODO(), cfg, pc, eng, fmttr, rw, rs)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(msg))
			})
		})

		Context("and there are no errors encountered in execution", func() {
			BeforeEach(func() {
				cfg.Submit = true
			})

			It("should complete with no errors", func() {
				err := preflightCheck(context.TODO(), cfg, pc, eng, fmttr, rw, rs)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
