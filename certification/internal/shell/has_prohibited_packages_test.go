package shell

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

var _ = Describe("HasNoProhibitedPackages", func() {
	var (
		HasNoProhibitedPackages HasNoProhibitedPackagesMountedCheck
		fakeEngine              cli.PodmanEngine
		pkgList                 []string
	)

	BeforeEach(func() {
		fakeEngine = FakePodmanEngine{
			RunReportStdout: "",
			RunReportStderr: "",
		}
		pkgList = []string{
			"this",
			"is",
			"not",
			"prohibitted",
		}
	})
	Describe("Checking if it has an prohibited packages", func() {
		Context("When there are no prohibited packages found", func() {
			BeforeEach(func() {
				podmanEngine = fakeEngine
			})
			It("should pass validate", func() {
				ok, err := HasNoProhibitedPackages.validate(pkgList)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When there was a prohibited packages found", func() {
			var pkgs []string
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				engine.RunReportStdout = "grub"
				podmanEngine = engine
				pkgs = append(pkgList, "grub")
			})
			It("should not pass Validate", func() {
				ok, err := HasNoProhibitedPackages.validate(pkgs)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Context("When there is a prohibited package in the glob list found", func() {
			var pkgs []string
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				engine.RunReportStdout = "kpatch2121"
				podmanEngine = engine
				pkgs = append(pkgList, "kpatch2121")
			})
			It("should not pass Validate", func() {
				ok, err := HasNoProhibitedPackages.validate(pkgs)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
	Describe("Checking that PodMan errors are handled correctly", func() {
		BeforeEach(func() {
			fakeEngine = BadPodmanEngine{}
			podmanEngine = fakeEngine
			os.Setenv("PREFLIGHT_EXEC_CHECK", HasNoProhibitedPackages.Name())
		})
		AfterEach(func() {
			os.Unsetenv("PREFLIGHT_EXEC_CHECK")
		})
		Context("When calling the Unshare", func() {
			Context("When PodMan throws an error", func() {
				It("should fail Validate and return an error", func() {
					ok, err := HasNoProhibitedPackages.Validate("dummy/image")
					Expect(err).To(HaveOccurred())
					Expect(ok).To(BeFalse())
				})
			})
		})
	})
})
