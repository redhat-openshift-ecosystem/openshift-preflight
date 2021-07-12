package shell

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

var _ = Describe("HasNoProhibitedPackages", func() {
	var (
		HasNoProhibitedPackages HasNoProhibitedPackagesCheck
		fakeEngine              cli.PodmanEngine
	)

	BeforeEach(func() {
		fakeEngine = FakePodmanEngine{
			RunReportStdout: "",
			RunReportStderr: "",
		}
	})
	Describe("Checking if it has an prohibited packages", func() {
		Context("When there are no prohibited packages found", func() {
			BeforeEach(func() {
				podmanEngine = fakeEngine
			})
			It("should pass Validate", func() {
				ok, err := HasNoProhibitedPackages.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When there was a prohibited packages found", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				engine.RunReportStdout = "grub"
				podmanEngine = engine
			})
			It("should not pass Validate", func() {
				ok, err := HasNoProhibitedPackages.Validate("dummy/image")
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
				ok, err := HasNoProhibitedPackages.Validate("dummy/image")
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
