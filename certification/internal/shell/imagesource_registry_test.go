package shell

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("imageSourceRegistry", func() {
	var (
		imageSourceRegistryCheck ImageSourceRegistryCheck
	)
	DescribeTable("Image Registry validation",
		func(bundleImage string, expected bool) {
			ok, _ := imageSourceRegistryCheck.Validate(bundleImage)
			Expect(ok).To(Equal(expected))
		},
		Entry("registry.connect.dev.redhat.com", "registry.connect.dev.redhat.com/", true),
		Entry("registry.connect.qa.redhat.com", "registry.connect.qa.redhat.com/", true),
		Entry("registry.connect.stage.redhat.com","registry.connect.stage.redhat.com/", true),
		Entry("registry.connect.redhat.com","registry.connect.redhat.com", true),
		Entry("registry.redhat.io","registry.redhat.io", true),
		Entry("registry.access.redhat.com", "registry.access.redhat.com/ubi8/ubi", true),
		Entry("quay.io", "quay.io/rocrisp/preflight-operator-bundle:v1", false),
		Entry("badcode","badcode/morebadcode/and,,,&&&:v1", false),
	)
})
