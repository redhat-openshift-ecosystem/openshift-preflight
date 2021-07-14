package shell

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

var _ = Describe("HasLicense", func() {
	var (
		HasLicense HasLicenseCheck
		fakeEngine cli.PodmanEngine
	)

	BeforeEach(func() {
		fakeEngine = FakePodmanEngine{
			RunReportStdout: `/licenses`,
			RunReportStderr: "",
		}
	})
	Describe("Checking if license can be found", func() {
		Context("When license is found", func() {
			BeforeEach(func() {
				podmanEngine = fakeEngine
			})
			It("should pass Validate", func() {
				ok, err := HasLicense.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When license is not found", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				engine.RunReportStdout = "No such file or directory"
				podmanEngine = engine
			})
			It("Should not pass Validate", func() {
				ok, err := HasLicense.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("License can't be found when directory exists", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				engine.RunReportStdout = ""
				podmanEngine = engine
			})
			It("Should not pass Validate", func() {
				ok, err := HasLicense.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
	Describe("Checking that PodMan errors are handled correctly", func() {
		BeforeEach(func() {
			fakeEngine = BadPodmanEngine{}
			podmanEngine = fakeEngine
		})
		Context("When PodMan throws an error", func() {
			It("should fail Validate and return an error", func() {
				ok, err := HasLicense.Validate("dummy/image")
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
