package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// This file imports logrus instead of internal/log because a standalone logger is used
	// for test specs defined here.
	log "github.com/sirupsen/logrus"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/lib"
)

var _ = Describe("CLI Library function", func() {
	When("invoking preflight using the CLI library", func() {
		Context("without passing in an artifact writer ", func() {
			It("should throw an error", func() {
				err := RunPreflight(context.TODO(), func(ctx context.Context) (runtime.Results, error) { return runtime.Results{}, nil }, CheckConfig{}, nil, nil, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no artifact writer"))
			})
		})

		Context("with a preconfigured artifact writer", func() {
			var testcontext context.Context
			var artifactWriter *artifacts.FilesystemWriter
			var testFormatter formatters.ResponseFormatter

			BeforeEach(func() {
				tmpDir, err := os.MkdirTemp("", "lib-execute-*")
				Expect(err).ToNot(HaveOccurred())
				artifactWriter, err = artifacts.NewFilesystemWriter(artifacts.WithDirectory(tmpDir))
				Expect(err).ToNot(HaveOccurred())
				testcontext = artifacts.ContextWithWriter(context.Background(), artifactWriter)
				DeferCleanup(os.RemoveAll, tmpDir)

				testFormatter, err = formatters.NewByName(formatters.DefaultFormat)
				Expect(err).ToNot(HaveOccurred())
			})

			It("It should fail if the artifact writter cannot write the results file", func() {
				var err error
				// Prewrite the expected result file to cause a conflict
				_, err = artifactWriter.WriteFile(
					ResultsFilenameWithExtension(testFormatter.FileExtension()),
					strings.NewReader("written for cli test case."))
				Expect(err).ToNot(HaveOccurred())

				err = RunPreflight(testcontext, func(ctx context.Context) (runtime.Results, error) { return runtime.Results{}, nil }, CheckConfig{}, testFormatter, &runtime.ResultWriterFile{}, nil)
				Expect(err).To(HaveOccurred())
			})

			It("Should return an error if unable to successfully check execution encounters an error", func() {
				err := RunPreflight(testcontext, func(ctx context.Context) (runtime.Results, error) { return runtime.Results{}, errors.New("some error") }, CheckConfig{}, testFormatter, &runtime.ResultWriterFile{}, nil)
				Expect(err).To(HaveOccurred())
			})

			It("Should throw an error writing formatted results if the formatter returns an error", func() {
				var err error
				testFormatter, err = formatters.New("test", "test", func(ctx context.Context, r runtime.Results) (response []byte, formattingError error) {
					return []byte{}, errors.New("unable to format")
				})
				Expect(err).ToNot(HaveOccurred())

				err = RunPreflight(testcontext, func(ctx context.Context) (runtime.Results, error) { return runtime.Results{}, nil }, CheckConfig{}, testFormatter, &runtime.ResultWriterFile{}, nil)
				Expect(err).To(HaveOccurred())
			})

			When("JUnit results are requested", func() {
				It("Should write the junit results as an artifact", func() {
					c := CheckConfig{
						IncludeJUnitResults: true,
					}

					err := RunPreflight(testcontext, func(ctx context.Context) (runtime.Results, error) {
						return runtime.Results{
							TestedImage:   "testWithJUnit",
							PassedOverall: true,
							Passed: []runtime.Result{
								{
									Check: certification.NewGenericCheck(
										"testJUnitWritten",
										func(ctx context.Context, ir certification.ImageReference) (bool, error) { return true, nil },
										certification.Metadata{},
										certification.HelpText{},
									),
									ElapsedTime: 1,
								},
							},
							Failed: []runtime.Result{},
							Errors: []runtime.Result{},
						}, nil
					}, c, testFormatter, &runtime.ResultWriterFile{}, nil)
					Expect(err).ToNot(HaveOccurred())
					expectedJUnitResultFile := filepath.Join(artifactWriter.Path(), "results-junit.xml")
					Expect(expectedJUnitResultFile).To(BeAnExistingFile())
				})

				It("should return an error if the junit artifact cannot be written", func() {
					// simulate this failure by causing a conflict writing the result-junit.xml file.
					c := CheckConfig{
						IncludeJUnitResults: true,
					}

					_, err := artifactWriter.WriteFile("results-junit.xml", strings.NewReader("conflicting junit contents for testing"))
					Expect(err).ToNot(HaveOccurred())

					err = RunPreflight(testcontext, func(ctx context.Context) (runtime.Results, error) {
						return runtime.Results{
							TestedImage:   "testWithJUnit",
							PassedOverall: true,
							Passed: []runtime.Result{
								{
									Check: certification.NewGenericCheck(
										"testJUnitWritten",
										func(ctx context.Context, ir certification.ImageReference) (bool, error) { return true, nil },
										certification.Metadata{},
										certification.HelpText{},
									),
									ElapsedTime: 1,
								},
							},
							Failed: []runtime.Result{},
							Errors: []runtime.Result{},
						}, nil
					}, c, testFormatter, &runtime.ResultWriterFile{}, nil)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("could not write file to artifacts directory")) // the error message returned by FilesystemWriter.
				})
			})

			When("Submission is requested", func() {
				It("Should call the submitter", func() {
					c := CheckConfig{
						SubmitResults: true,
					}

					buf := bytes.NewBuffer([]byte{})
					submitterTestLogger := log.New()
					submitterTestLogger.SetOutput(buf)
					submitterTestLogger.SetFormatter(&log.TextFormatter{})
					testSubmitter := lib.NewNoopSubmitter(true, submitterTestLogger)

					err := RunPreflight(testcontext, func(ctx context.Context) (runtime.Results, error) {
						return runtime.Results{
							TestedImage:   "testSubmission",
							PassedOverall: true,
							Passed: []runtime.Result{
								{
									Check: certification.NewGenericCheck(
										"testSubmission",
										func(ctx context.Context, ir certification.ImageReference) (bool, error) { return true, nil },
										certification.Metadata{},
										certification.HelpText{},
									),
									ElapsedTime: 1,
								},
							},
							Failed: []runtime.Result{},
							Errors: []runtime.Result{},
						}, nil
					}, c, testFormatter, &runtime.ResultWriterFile{}, testSubmitter)
					Expect(err).ToNot(HaveOccurred())

					contents, err := io.ReadAll(buf)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(contents)).To(ContainSubstring("Results are not being sent for submission"))
				})

				It("Should return an error if the submitter is unable to submit", func() {
					c := CheckConfig{
						SubmitResults: true,
					}

					submissionError := "unable to submit"

					err := RunPreflight(testcontext, func(ctx context.Context) (runtime.Results, error) {
						return runtime.Results{
							TestedImage:   "testSubmission",
							PassedOverall: true,
							Passed: []runtime.Result{
								{
									Check: certification.NewGenericCheck(
										"testSubmission",
										func(ctx context.Context, ir certification.ImageReference) (bool, error) { return true, nil },
										certification.Metadata{},
										certification.HelpText{},
									),
									ElapsedTime: 1,
								},
							},
							Failed: []runtime.Result{},
							Errors: []runtime.Result{},
						}, nil
					}, c, testFormatter, &runtime.ResultWriterFile{}, &badResultSubmitter{submissionError})
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(submissionError))
				})
			})
		})
	})
})

var _ = Describe("JUnit", func() {
	var results *runtime.Results
	var junitfile string
	var artifactWriter *artifacts.FilesystemWriter
	var testcontext context.Context

	BeforeEach(func() {
		tmpDir, err := os.MkdirTemp("", "junit-*")
		Expect(err).ToNot(HaveOccurred())
		artifactWriter, err = artifacts.NewFilesystemWriter(artifacts.WithDirectory(tmpDir))
		Expect(err).ToNot(HaveOccurred())
		testcontext = artifacts.ContextWithWriter(context.Background(), artifactWriter)
		DeferCleanup(os.RemoveAll, tmpDir)

		results = &runtime.Results{
			TestedImage:       "registry.example.com/example/image:0.0.1",
			PassedOverall:     true,
			TestedOn:          runtime.UnknownOpenshiftClusterVersion(),
			CertificationHash: "sha256:deadb33f",
			Passed:            []runtime.Result{},
			Failed:            []runtime.Result{},
			Errors:            []runtime.Result{},
		}
		junitfile = filepath.Join(artifactWriter.Path(), "results-junit.xml")
	})

	When("The additional JUnitXML results file is requested", func() {
		It("should be written to the artifacts directory without error", func() {
			Expect(writeJUnit(testcontext, *results)).To(Succeed())
			_, err := os.Stat(junitfile)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = DescribeTable("Checking overall pass/fail",
	func(result bool, expected string) {
		Expect(convertPassedOverall(result)).To(Equal(expected))
	},
	Entry("when passing true", true, "PASSED"),
	Entry("when passing false", false, "FAILED"),
)
