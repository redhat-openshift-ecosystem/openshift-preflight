// go:test !race
package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("support command tests", func() {
	Context("When running the support cobra command", func() {
		BeforeEach(createAndCleanupDirForArtifactsAndLogs)
		Context("with valid inputs for a container project", func() {
			It("should run without error", func() {
				_, err := executeCommand(rootCmd, "support", "container", "000000000000")
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("with valid inputs for an operator project", func() {
			It("should run without error", func() {
				_, err := executeCommand(rootCmd, "support", "operator", "000000000000", "https://github.com/example/pull/2")
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with an invalid project type", func() {
			It("should throw an error", func() {
				out, err := executeCommand(rootCmd, "support", "fooproject", "000000000000")
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("the project type must be"))
			})
		})

		Context("with an invalid project ID", func() {
			It("should throw an error", func() {
				out, err := executeCommand(rootCmd, "support", "container", "ospid-00000000")
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("please remove leading characters ospid-"))
			})
		})

		Context("with an operator project designation but no pull request url", func() {
			It("should throw an error", func() {
				out, err := executeCommand(rootCmd, "support", "operator", "000000000000")
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("a pull request URL is required"))
			})
		})

		Context("with an operator project designation but an invalid pull request url", func() {
			It("should throw an error", func() {
				out, err := executeCommand(rootCmd, "support", "operator", "000000000000", "github.com")
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("please enter a valid url"))
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
