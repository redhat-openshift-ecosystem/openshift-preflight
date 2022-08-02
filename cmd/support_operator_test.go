package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("support command tests", func() {
	Context("When running the support operator cobra command", func() {
		BeforeEach(createAndCleanupDirForArtifactsAndLogs)
		Context("with valid inputs for an operator project", func() {
			It("should run without error", func() {
				_, err := executeCommand(supportOperatorCmd(), "000000000000", "https://github.com/example/pull/2")
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("with an invalid project ID", func() {
			It("should throw an error", func() {
				out, err := executeCommand(supportOperatorCmd(), "ospid-00000000", "https://github.com/example/pull/2")
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("please remove leading characters ospid-"))
			})
		})
		Context("with an operator project designation but no pull request url", func() {
			It("should throw an error", func() {
				out, err := executeCommand(supportOperatorCmd(), "000000000000")
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("Error:"))
			})
		})

		Context("with an operator project designation but an invalid pull request url", func() {
			It("should throw an error", func() {
				out, err := executeCommand(supportOperatorCmd(), "000000000000", "github.com")
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("please enter a valid url"))
			})
		})
	})
})
