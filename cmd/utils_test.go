package cmd

import (
	"os"

	. "github.com/onsi/ginkgo"
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
				Expect(actual).To(Equal(expected))
			})
		})

		Context("with an extension of txt", func() {
			extension := "json"
			expected := "results.json"

			It("should be results.json", func() {
				actual := resultsFilenameWithExtension(extension)
				Expect(actual).To(Equal(expected))
			})
		})
	})

	Describe("Test Connect URL builders", func() {
		BeforeEach(func() {
			viper.SetEnvPrefix("pflt")
			viper.AutomaticEnv()
		})
		Context("Regular Connect URL", func() {
			expected := "https://connect.redhat.com/projects/this-is-my-project-id"
			It("should return a URL with just a project ID", func() {
				actual := buildConnectURL("this-is-my-project-id")
				Expect(actual).To(Equal(expected))
			})
		})
		Context("QA Connect URL", func() {
			BeforeEach(func() {
				os.Setenv("PFLT_PYXIS_HOST", "catalog.qa.redhat.com")
			})
			expected := "https://connect.qa.redhat.com/projects/this-is-my-project-id"
			It("should return a URL for QA", func() {
				actual := buildConnectURL("this-is-my-project-id")
				Expect(actual).To(Equal(expected))
			})
		})
		Context("UAT Scan Results URL", func() {
			BeforeEach(func() {
				os.Setenv("PFLT_PYXIS_HOST", "catalog.uat.redhat.com")
			})
			expected := "https://connect.uat.redhat.com/projects/this-is-my-project-id/images/my-image-id/scan-results"
			It("should return a URL for UAT", func() {
				actual := buildScanResultsURL("this-is-my-project-id", "my-image-id")
				Expect(actual).To(Equal(expected))
			})
		})
		Context("QA Overview URL", func() {
			BeforeEach(func() {
				os.Setenv("PFLT_PYXIS_HOST", "catalog.qa.redhat.com")
			})
			expected := "https://connect.qa.redhat.com/projects/this-is-my-project-id/overview"
			It("should return a URL for QA", func() {
				actual := buildOverviewURL("this-is-my-project-id")
				Expect(actual).To(Equal(expected))
			})
		})
	})
})
