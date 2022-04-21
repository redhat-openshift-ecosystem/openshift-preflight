package cmd

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

var _ = Describe("cmd package utility functions", func() {
	Describe("Determine filename to which to write test results", func() {
		// Ensure resultsFilenameWithExtension accurately joins the
		// expected default filename of "results" with the extension
		// that is provided.
		Context("with an extension of txt", func() {
			extension := "txt"
			expected := "results.txt"

			It("should be results.txt", func() {
				actual := resultsFilenameWithExtension(extension)
				Expect(expected).To(Equal(actual))
			})
		})

		Context("with an extension of txt", func() {
			extension := "json"
			expected := "results.json"

			It("should be results.json", func() {
				actual := resultsFilenameWithExtension(extension)
				Expect(expected).To(Equal(actual))
			})
		})
	})

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
})
