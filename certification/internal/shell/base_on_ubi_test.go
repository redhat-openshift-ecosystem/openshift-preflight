package shell

import (
	. "github.com/onsi/ginkgo"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

// podmanEngine is a package-level variable. In some tests, we
// override it with a "happy path" engine, that returns good data.
// In the unhappy path, we override it with an engine that returns
// nothing but errors.

var _ = Describe("BaseOnUBI", func() {
	var (
		baseOnUbiCheck BaseOnUBICheck
		fakeEngine     cli.PodmanEngine
	)

	BeforeEach(func() {
		fakeEngine = FakePodmanEngine{
			RunReportStdout: `ID="rhel"
NAME="Red Hat Enterprise Linux"
`,
			RunReportStderr: "",
		}
	})
	Describe("Checking for UBI as a base", func() {
		Context("When it is based on UBI", func() {
			BeforeEach(func() {
				podmanEngine = fakeEngine
			})
			checkShouldPassValidate(&baseOnUbiCheck)()
		})
		Context("When it is not based on UBI", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				engine.RunReportStdout = `ID="notrhel"`
				podmanEngine = engine
			})
			checkShouldNotPassValidate(&baseOnUbiCheck)()
		})
	})
	checkPodmanErrors(&baseOnUbiCheck)()
})
