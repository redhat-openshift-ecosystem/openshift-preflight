package cmd

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	"github.com/spf13/viper"
)

var _ = Describe("cmd package check command", func() {
	DescribeTable("Checking overall pass/fail",
		func(result bool, expected string) {
			Expect(convertPassedOverall(result)).To(Equal(expected))
		},
		Entry("when passing true", true, "PASSED"),
		Entry("when passing false", false, "FAILED"),
	)

	Describe("Test Connect URL builders", func() {
		var (
			projectId string = "this-is-my-project-id"
			imageId   string = "my-image-id"
		)
		BeforeEach(func() {
			viper.SetEnvPrefix("pflt")
			viper.AutomaticEnv()
		})
		AfterEach(func() {
			os.Unsetenv("PFLT_PYXIS_ENV")
			os.Unsetenv("PFLT_PYXIS_HOST")
		})
		Context("Regular Connect URL", func() {
			It("should return a URL with just a project ID", func() {
				expected := "https://connect.redhat.com/projects/this-is-my-project-id"
				actual := buildConnectURL(projectId)
				Expect(expected).To(Equal(actual))
			})
		})
		Context("QA Connect URL", func() {
			BeforeEach(func() {
				os.Setenv("PFLT_PYXIS_ENV", "qa")
			})
			It("should return a URL for QA", func() {
				expected := "https://connect.qa.redhat.com/projects/this-is-my-project-id"
				actual := buildConnectURL(projectId)
				Expect(expected).To(Equal(actual))
			})
		})
		Context("UAT Scan Results URL", func() {
			BeforeEach(func() {
				os.Setenv("PFLT_PYXIS_ENV", "uat")
			})
			It("should return a URL for UAT", func() {
				expected := "https://connect.uat.redhat.com/projects/this-is-my-project-id/images/my-image-id/scan-results"
				actual := buildScanResultsURL(projectId, imageId)
				Expect(expected).To(Equal(actual))
			})
		})
		Context("QA Overview URL", func() {
			BeforeEach(func() {
				os.Setenv("PFLT_PYXIS_ENV", "qa")
			})
			It("should return a URL for QA", func() {
				expected := "https://connect.qa.redhat.com/projects/this-is-my-project-id/overview"
				actual := buildOverviewURL(projectId)
				Expect(expected).To(Equal(actual))
			})
		})
		Context("Override Pyxis Host", func() {
			BeforeEach(func() {
				os.Setenv("PFLT_PYXIS_HOST", "my.pyxis.host/some/path")
			})
			It("should return a Prod overview URL", func() {
				expected := "https://connect.redhat.com/projects/this-is-my-project-id/overview"
				actual := buildOverviewURL(projectId)
				Expect(expected).To(Equal(actual))
			})
			It("should return a Prod scan URL", func() {
				expected := "https://connect.redhat.com/projects/this-is-my-project-id/images/my-image-id/scan-results"
				actual := buildScanResultsURL(projectId, imageId)
				Expect(expected).To(Equal(actual))
			})
		})
	})

	Describe("JUnit", func() {
		var results *runtime.Results
		var artifactsDir string
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
			var err error
			artifactsDir, err = os.MkdirTemp("", "junit-test-*")
			Expect(err).ToNot(HaveOccurred())
			// change the artifacts dir to our test dir
			artifacts.SetDir(artifactsDir)
			junitfile = filepath.Join(artifactsDir, "results-junit.xml")
		})
		AfterEach(func() {
			Expect(os.RemoveAll(artifactsDir)).To(Succeed())
			// BeforeEach changes the default artifacts dir, so reset it.
			artifacts.Reset()
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
