package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("support command tests", func() {
	Context("When running the support cobra command", func() {
		BeforeEach(createAndCleanupDirForArtifactsAndLogs)
		Context("with an invalid project type", func() {
			It("should throw an error", func() {
				out, err := executeCommand(supportCmd(), "fooproject", "000000000000")
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("Error: unknown command"))
			})
		})
	})
	Context("When validating a pull request URL", func() {
		urlNoScheme := "example.com"
		urlNoHost := "https:///foo"
		urlNoPath := "https://example.com"
		urlCorrect := "https://example.com/pull/example"

		It("should fail when no scheme is provided", func() {
			err := pullRequestURLValidation(urlNoScheme)
			Expect(err).To(HaveOccurred())
		})
		It("should fail when no host is provided", func() {
			err := pullRequestURLValidation(urlNoHost)
			Expect(err).To(HaveOccurred())
		})
		It("should fail when no path is provided", func() {
			err := pullRequestURLValidation(urlNoPath)
			Expect(err).To(HaveOccurred())
		})

		It("should succeed when the url is in the proper format", func() {
			err := pullRequestURLValidation(urlCorrect)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When validating a project ID", func() {
		projectIDEmpty := ""
		projectIDStartWithP := "p1"
		projectIDStartWithOSPID := "ospid-0000"
		projectIDContainSpecialChar := "$c$"
		projectIDCorrect := "000011112222"

		It("should fail if the input is empty", func() {
			err := projectIDValidation(projectIDEmpty)
			Expect(err).To(HaveOccurred())
		})

		It("should fail if if the input contains a leading p", func() {
			err := projectIDValidation(projectIDStartWithP)
			Expect(err).To(HaveOccurred())
		})

		It("should fail if the input begins with ospid-", func() {
			err := projectIDValidation(projectIDStartWithOSPID)
			Expect(err).To(HaveOccurred())
		})

		It("should fail if the input has special characters", func() {
			err := projectIDValidation(projectIDContainSpecialChar)
			Expect(err).To(HaveOccurred())
		})

		It("should succeed if the format is correct", func() {
			err := projectIDValidation(projectIDCorrect)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
