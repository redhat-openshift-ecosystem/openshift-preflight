package shell

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

var _ = Describe("RunAsNonRoot", func() {
	var (
		RunAsNonRoot RunAsNonRootCheck
		fakeEngine   cli.PodmanEngine
	)

	BeforeEach(func() {
		fakeEngine = FakePodmanEngine{
			RunReportStdout: "1",
			RunReportStderr: "",
		}
	})

	Describe("Checking runtime user is not root", func() {
		Context("When runtime user is not root", func() {
			BeforeEach(func() {
				podmanEngine = fakeEngine
			})
			It("should pass Validate", func() {
				ok, err := RunAsNonRoot.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When runtime user is root", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				engine.RunReportStdout = "0"
				podmanEngine = engine
			})
			It("should not pass Validate", func() {
				ok, err := RunAsNonRoot.Validate("dummy/image")
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
				ok, err := RunAsNonRoot.Validate("dummy/image")
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
