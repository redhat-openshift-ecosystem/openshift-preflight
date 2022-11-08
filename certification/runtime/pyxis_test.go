package runtime

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pyxis Host Lookup", func() {
	When("Resolving the pyxis host", func() {
		Context("having no host override", func() {
			It("should return the default value for the requested environment", func() {
				val := PyxisHostLookup("prod", "")
				Expect(val).To(Equal("catalog.redhat.com/api/containers"))
			})
			Context("having an invalid environment value", func() {
				It("sholud return the prod endpoint", func() {
					val := PyxisHostLookup("invalid", "")
					Expect(val).To(Equal("catalog.redhat.com/api/containers"))
				})
			})
		})

		Context("with a host override", func() {
			It("should return the override", func() {
				val := PyxisHostLookup("prod", "overridden")
				Expect(val).To(Equal("overridden"))
			})
		})
	})
})
