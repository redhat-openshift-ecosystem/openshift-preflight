package shell

import (
	. "github.com/onsi/ginkgo"
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
			It("should pass Validate", checkShouldPassValidate(&hasRequiredLabelsCheck))
		})
		Context("When it does not have required labels", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				delete(engine.ImageInspectReport.Images[0].Config.Labels, "description")
			})
			It("should not pass Validate", checkShouldNotPassValidate(&hasRequiredLabelsCheck))
		})
	})
	checkPodmanErrors(&hasRequiredLabelsCheck)()
})
