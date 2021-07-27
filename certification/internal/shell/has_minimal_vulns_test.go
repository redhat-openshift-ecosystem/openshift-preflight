package shell

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
)

// podmanEngine is a package-level variable. In some tests, we
// override it with a "happy path" engine, that returns good data.
// In the unhappy path, we override it with an engine that returns
// nothing but errors.

var _ = Describe("HasMinimalVulns", func() {
	var (
		hasMinimalVulnCheck HasMinimalVulnerabilitiesCheck
		fakeEngine          cli.PodmanEngine
	)

	BeforeEach(func() {
		fakeEngine = FakePodmanEngine{
			ImageScanReport: cli.ImageScanReport{
				Stdout: "",
				Stderr: "",
			},
		}
	})
	Describe("Checking for unacceptable vulnerabilities", func() {
		Context("When it does not has a vulnerability", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				engine.ImageScanReport.Stdout = "Definition oval:com.redhat.rhsa:def:20212660: false\nDefinition oval:com.redhat.rhsa:def:20212599: false\n"
				podmanEngine = engine
			})
			It("should pass Validate", func() {
				ok, err := hasMinimalVulnCheck.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When it has a vulnerability", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				engine.ImageScanReport.Stdout = "Definition oval:com.redhat.rhsa:def:20212660: true\nDefinition oval:com.redhat.rhsa:def:20212599: false\n"
				podmanEngine = engine
			})
			It("should not pass Validate", func() {
				log.Errorf("Run Report: %s", podmanEngine)
				ok, err := hasMinimalVulnCheck.Validate("dummy/image")
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
				ok, err := hasMinimalVulnCheck.Validate("dummy/image")
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
