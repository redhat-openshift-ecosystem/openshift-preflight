package openshift

import (
	"context"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OpenShift Version", func() {
	When("no KUBECONFIG is provided", func() {
		It("should return UnknownVersion and an error", func() {
			version, err := GetOpenshiftClusterVersion(context.Background(), []byte{})
			Expect(version).To(BeEquivalentTo(runtime.UnknownOpenshiftClusterVersion()))
			Expect(err).To(HaveOccurred())
		})
	})

	When("an invalid KUBECONFIG is passed", func() {
		It("should return UnkownVersion and an error", func() {
			version, err := GetOpenshiftClusterVersion(context.Background(), []byte("foo"))
			Expect(version).To(BeEquivalentTo(runtime.UnknownOpenshiftClusterVersion()))
			Expect(err).To(HaveOccurred())
		})
	})
})
