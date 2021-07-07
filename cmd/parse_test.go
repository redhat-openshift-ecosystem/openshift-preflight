package cmd

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config parsing functions", func() {

	Describe("Checking user configuration precedence", func() {
		var (
			emptyFlagValue, emptyEnvValue = "", ""
			flagValue                     = "flagValue"
			envValue                      = "envValue"
			defaultValue                  = "defaultValue"
		)

		Context("When the user provides a value via flag", func() {
			res := getConfigStringValueFrom(flagValue, envValue, defaultValue)
			It("should always take precedence", func() {
				Expect(res).To(Equal(flagValue))
			})
		})

		Context("When the user does not provides a flag value but provides an environment variable", func() {
			res := getConfigStringValueFrom(emptyFlagValue, envValue, defaultValue)
			It("should return the environment variable", func() {
				Expect(res).To(Equal(envValue))
			})
		})

		Context("When the user provides neither flag nor environment variable values", func() {
			res := getConfigStringValueFrom(emptyFlagValue, emptyEnvValue, defaultValue)
			It("should return the default value", func() {
				Expect(res).To(Equal(defaultValue))
			})
		})
	})
})
