package cmd

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
})
