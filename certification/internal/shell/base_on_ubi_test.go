package shell

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BaseOnUBI", func() {
	var (
		engine      FakePodmanEngine
		base_on_ubi BaseOnUBICheck
	)

	BeforeEach(func() {
		engine = FakePodmanEngine{
			RunReportStdout: `ID="rhel"
NAME="Red Hat Enterprise Linux"
`,
			RunReportStderr: "",
		}
	})

	Describe("Checking for UBI as a base", func() {
		Context("When it is based on UBI", func() {
			It("should succeed the check", func() {
				ok, err := base_on_ubi.validate(engine, "dummy/image", logger)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When it is not based on UBI", func() {
			BeforeEach(func() {
				engine.RunReportStdout = `ID=notrhel
NAME="Some Other Linux"
`
			})
			It("should not succeed the check", func() {
				ok, err := base_on_ubi.validate(engine, "dummy/image", logger)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
