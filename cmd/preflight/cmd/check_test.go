package cmd

import (
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/lib"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/viper"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("cmd package check command", func() {
	BeforeEach(createAndCleanupDirForArtifactsAndLogs)

	Describe("Test Connect URL builders", func() {
		var (
			projectID = "this-is-my-project-id"
			imageID   = "my-image-id"
		)
		BeforeEach(func() {
			viper.Instance().SetEnvPrefix("pflt")
			viper.Instance().AutomaticEnv()
		})
		AfterEach(func() {
			os.Unsetenv("PFLT_PYXIS_ENV")
			os.Unsetenv("PFLT_PYXIS_HOST")
		})
		Context("Regular Connect URL", func() {
			It("should return a URL with just a project ID", func() {
				expected := "https://connect.redhat.com/projects/this-is-my-project-id"
				actual := lib.BuildConnectURL(projectID)
				Expect(expected).To(Equal(actual))
			})
		})
		Context("QA Connect URL", func() {
			BeforeEach(func() {
				os.Setenv("PFLT_PYXIS_ENV", "qa")
			})
			It("should return a URL for QA", func() {
				expected := "https://connect.qa.redhat.com/projects/this-is-my-project-id"
				actual := lib.BuildConnectURL(projectID)
				Expect(expected).To(Equal(actual))
			})
		})
		Context("UAT Scan Results URL", func() {
			BeforeEach(func() {
				os.Setenv("PFLT_PYXIS_ENV", "uat")
			})
			It("should return a URL for UAT", func() {
				expected := "https://connect.uat.redhat.com/projects/this-is-my-project-id/images/my-image-id/scan-results"
				actual := lib.BuildScanResultsURL(projectID, imageID)
				Expect(expected).To(Equal(actual))
			})
		})
		Context("QA Overview URL", func() {
			BeforeEach(func() {
				os.Setenv("PFLT_PYXIS_ENV", "qa")
			})
			It("should return a URL for QA", func() {
				expected := "https://connect.qa.redhat.com/projects/this-is-my-project-id/overview"
				actual := lib.BuildOverviewURL(projectID)
				Expect(expected).To(Equal(actual))
			})
		})
		Context("Override Pyxis Host", func() {
			BeforeEach(func() {
				os.Setenv("PFLT_PYXIS_HOST", "my.pyxis.host/some/path")
			})
			It("should return a Prod overview URL", func() {
				expected := "https://connect.redhat.com/projects/this-is-my-project-id/overview"
				actual := lib.BuildOverviewURL(projectID)
				Expect(expected).To(Equal(actual))
			})
			It("should return a Prod scan URL", func() {
				expected := "https://connect.redhat.com/projects/this-is-my-project-id/images/my-image-id/scan-results"
				actual := lib.BuildScanResultsURL(projectID, imageID)
				Expect(expected).To(Equal(actual))
			})
		})
	})
})
