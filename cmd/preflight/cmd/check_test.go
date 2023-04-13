package cmd

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/lib"

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
		Context("Regular Connect URL", func() {
			It("should return a URL with just a project ID", func() {
				expected := "https://connect.redhat.com/projects/this-is-my-project-id"
				actual := lib.BuildConnectURL(projectID, "prod")
				Expect(expected).To(Equal(actual))
			})
		})
		Context("QA Connect URL", func() {
			It("should return a URL for QA", func() {
				expected := "https://connect.qa.redhat.com/projects/this-is-my-project-id"
				actual := lib.BuildConnectURL(projectID, "qa")
				Expect(expected).To(Equal(actual))
			})
		})
		Context("UAT Scan Results URL", func() {
			It("should return a URL for UAT", func() {
				expected := "https://connect.uat.redhat.com/projects/this-is-my-project-id/images/my-image-id/scan-results"
				actual := lib.BuildScanResultsURL(projectID, imageID, "uat")
				Expect(expected).To(Equal(actual))
			})
		})
		Context("QA Overview URL", func() {
			It("should return a URL for QA", func() {
				expected := "https://connect.qa.redhat.com/projects/this-is-my-project-id/overview"
				actual := lib.BuildOverviewURL(projectID, "qa")
				Expect(expected).To(Equal(actual))
			})
		})
	})
})
