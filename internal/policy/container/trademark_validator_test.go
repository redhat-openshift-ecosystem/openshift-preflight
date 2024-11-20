package container

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TrademarkValidator", func() {
	DescribeTable("Test all presentations of `Red Hat`",
		func(trademarkText string, expected bool) {
			result := violatesRedHatTrademark(trademarkText)
			Expect(result).To(Equal(expected))
		},

		Entry("`Red Hat` should violate trademark policy", "Red Hat", true),
		Entry("`Something for Red Hat OpenShift` should not violate trademark policy", "Something for Red Hat OpenShift", false),
		Entry("`Red-Hat` should violate trademark policy", "Red-Hat", true),
		Entry("`Red_Hat` should violate trademark policy", "Red_Hat", true),
		Entry("`For-Red-Hat` should not violate trademark policy", "For-Red-Hat", false),
		Entry("`For_Red_Hat` should not violate trademark policy", "For_Red_Hat", false),
		Entry("`RED			HAT			` should violate trademark policy", "RED		HAT			", true),
		Entry("`redhat` should violate trademark policy", "redhat", true),
		Entry("`something by red hat for red hat` should violate trademark policy", "something by red hat for red hat", true),
		Entry("`red hat product for red hat` should violate trademark policy", "red hat product for red hat", true),
	)
})
