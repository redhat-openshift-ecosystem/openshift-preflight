package runtime

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Runtime tests", func() {
	Context("when the cluster version is unknown", func() {
		It("will get unknown/unknown in the version struct", func() {
			version := UnknownOpenshiftClusterVersion()
			Expect(version.Name).To(Equal("unknown"))
			Expect(version.Version).To(Equal("unknown"))
		})
	})
})
