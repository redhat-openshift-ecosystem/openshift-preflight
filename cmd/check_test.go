package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("cmd package check command", func() {
	Describe("Test Flags", func() {
		Context("Docker Config", func() {
			It("docker-config from Flags() should be nil, but PersistentFlags() should be set", func() {
				expected := "/my/docker/config.json"
				checkCmd.PersistentFlags().Set("docker-config", expected)
				Expect(checkCmd.Flags().Lookup("docker-config")).To(BeNil())
				Expect(checkCmd.PersistentFlags().Lookup("docker-config").Value.String()).To(Equal(expected))
			})
		})
	})
})
