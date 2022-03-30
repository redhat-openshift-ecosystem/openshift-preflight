package engine

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Source RPM name function", func() {
	Context("With a source rpm name", func() {
		Context("And a normal source rpm name", func() {
			It("should parse bash-5.1.8-2.el9.src.rpm to bash", func() {
				expected := "bash"
				actual := getBgName("bash-5.1.8-2.el9.src.rpm")
				Expect(actual).To(Equal(expected))
			})
		})
		Context("And a slightly annoying source rpm name", func() {
			It("should parse python3.9-3.9.6-6.el9.src.rpm to python3.9", func() {
				expected := "python3.9"
				actual := getBgName("python3.9-3.9.6-6.el9.src.rpm")
				Expect(actual).To(Equal(expected))
			})
		})
		Context("And a source rpm name with a bunch of -'s", func() {
			It("should parse python-pip-21.0.1-6.el9.src.rpm to bash", func() {
				expected := "python-pip"
				actual := getBgName("python-pip-21.0.1-6.el9.src.rpm")
				Expect(actual).To(Equal(expected))
			})
		})
	})
})
