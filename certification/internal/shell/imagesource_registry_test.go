package shell

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("imageSourceRegistry", func() {
	var (
		imageSourceRegistryCheck ImageSourceRegistryCheck
	)
	Describe("Checking for valid image source registry ", func() {
		Context("When the image source is in the approved registry", func() {
			It("should pass Validate", func() {
				ok, err := imageSourceRegistryCheck.Validate("registry.access.redhat.com/ubi8/ubi")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When image source is not in the approved registry", func() {
			It("should not pass Validate", func() {
				log.Errorf("Run Report: %s", skopeoEngine)
				ok, err := imageSourceRegistryCheck.Validate("quay.io/rocrisp/preflight-operator-bundle:v1")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})