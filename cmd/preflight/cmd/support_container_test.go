package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("support container command tests", func() {
	Context("When running the support container cobra command", func() {
		BeforeEach(createAndCleanupDirForArtifactsAndLogs)
		Context("with valid inputs", func() {
			It("should run without error", func() {
				_, err := executeCommand(supportContainerCmd(), "000000000000")
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("with an invalid project ID", func() {
			It("should throw an error", func() {
				out, err := executeCommand(supportContainerCmd(), "ospid-00000000")
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("please remove leading characters ospid-"))
			})
		})
		Context("with no project ID", func() {
			It("should throw an error", func() {
				out, err := executeCommand(supportContainerCmd())
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("Error:"))
			})
		})
	})
})
