package container

import (
	"context"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

var _ = Describe("RunSystemContainerCheck", func() {
	var (
		engine                  cli.PodmanEngine
		runSystemContainerCheck RunSystemContainerCheck
		imageRef                certification.ImageReference
	)

	BeforeEach(func() {
		imageRef.ImageURI = "test-image:test-tag"
	})

	Describe("Checking that the container can run as a systemd service", func() {
		Context("When container service starts successfully", func() {
			It("should pass Validate", func() {
				engine = GoodPodmanEngine{}
				runSystemContainerCheck = *NewRunSystemContainerCheck(&engine)
				ok, err := runSystemContainerCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When container service does not start successfully", func() {
			It("should not pass Validate", func() {
				engine = BadPodmanEngine{}
				runSystemContainerCheck = *NewRunSystemContainerCheck(&engine)
				ok, err := runSystemContainerCheck.Validate(context.TODO(), imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
