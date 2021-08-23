package shell

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/migration"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

var _ = Describe("HasRequiredLabels", func() {
	var (
		hasRequiredLabelsCheck HasRequiredLabelsCheck
		fakeEngine             cli.PodmanEngine
	)

	BeforeEach(func() {
		labels := map[string]string{
			"name":        "name",
			"vendor":      "vendor",
			"version":     "version",
			"release":     "release",
			"summary":     "summary",
			"description": "description",
		}
		fakeEngine = FakePodmanEngine{
			ImageInspectReport: cli.ImageInspectReport{
				Images: []cli.PodmanImage{
					{
						Id: "imageid",
						Config: cli.PodmanImageConfig{
							Labels: labels,
						},
					},
				},
			},
		}
		podmanEngine = fakeEngine
	})

	Describe("Checking for required labels", func() {
		Context("When it has required labels", func() {
			It("should pass Validate", func() {
				ok, err := hasRequiredLabelsCheck.Validate(migration.ImageToImageReference("dummy/image"))
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When it does not have required labels", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				delete(engine.ImageInspectReport.Images[0].Config.Labels, "description")
			})
			It("should not succeed the check", func() {
				ok, err := hasRequiredLabelsCheck.Validate(migration.ImageToImageReference("dummy/image"))
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
				ok, err := hasRequiredLabelsCheck.Validate(migration.ImageToImageReference("dummy/image"))
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
