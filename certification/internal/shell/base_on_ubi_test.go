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

var _ = Describe("BaseOnUBI", func() {
	var (
		baseOnUbiCheck BasedOnUBICheck
		fakeEngine     cli.PodmanEngine
	)

	BeforeEach(func() {
		fakeEngine = FakePodmanEngine{
			RunReportStdout: `ID="rhel"
NAME="Red Hat Enterprise Linux"
`,
			RunReportStderr: "",
		}
		baseOnUbiCheck = *NewBasedOnUBICheck(&fakeEngine)
	})
	Describe("Checking for UBI as a base", func() {
		Context("When it is based on UBI", func() {
			It("should pass Validate", func() {
				ok, err := baseOnUbiCheck.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When it is not based on UBI", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				engine.RunReportStdout = `ID="notrhel"`
				baseOnUbiCheck.PodmanEngine = engine
			})
			It("should not pass Validate", func() {
				log.Errorf("Run Report: %s", podmanEngine)
				ok, err := baseOnUbiCheck.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
			AfterEach(func() {
				baseOnUbiCheck.PodmanEngine = fakeEngine
			})
		})
	})
	Describe("Checking that PodMan errors are handled correctly", func() {
		BeforeEach(func() {
			badEngine := BadPodmanEngine{}
			baseOnUbiCheck.PodmanEngine = badEngine
		})
		Context("When PodMan throws an error", func() {
			It("should fail Validate and return an error", func() {
				ok, err := baseOnUbiCheck.Validate("dummy/image")
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		AfterEach(func() {
			baseOnUbiCheck.PodmanEngine = fakeEngine
		})
	})
})
