package container

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
)

var _ = Describe("RunnableContainerCheck", func() {
	var (
		engine                 cli.PodmanEngine
		runnableContainerCheck RunnableContainerCheck
		imageRef               certification.ImageReference
	)

	BeforeEach(func() {
		imageRef.ImageURI = "test-image:test-tag"
	})

	Describe("Checking that the container runs within a specified timeframe", func() {
		Context("When container starts successfully", func() {
			It("should pass Validate", func() {
				engine = GoodPodmanEngine{}
				runnableContainerCheck = *NewRunnableContainerCheck(&engine)
				ok, err := runnableContainerCheck.Validate(imageRef)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When container does not run successfully", func() {
			It("should not pass Validate", func() {
				engine = BadPodmanEngine{}
				runnableContainerCheck = *NewRunnableContainerCheck(&engine)
				ok, err := runnableContainerCheck.Validate(imageRef)
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
