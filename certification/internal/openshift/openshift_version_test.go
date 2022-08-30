package openshift

import (
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OpenShift Version", func() {
	When("no KUBECONFIG is provided", func() {
		It("should return UnknownVersion and an error", func() {
			version, err := GetOpenshiftClusterVersion()
			Expect(version).To(BeEquivalentTo(runtime.UnknownOpenshiftClusterVersion()))
			Expect(err).To(HaveOccurred())
		})
	})

	When("an invalid KUBECONFIG is passed", func() {
		BeforeEach(func() {
			os.Setenv("KUBECONFIG", "/bad/kubeconfig")
			DeferCleanup(os.Unsetenv, "KUBECONFIG")
		})
		It("should return UnkownVersion and an error", func() {
			version, err := GetOpenshiftClusterVersion()
			Expect(version).To(BeEquivalentTo(runtime.UnknownOpenshiftClusterVersion()))
			Expect(err).To(HaveOccurred())
		})
	})
})
