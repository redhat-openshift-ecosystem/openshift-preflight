package shell

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

var _ = Describe("LessThanMaxLayers", func() {
	var (
		lessThanMaxLayers UnderLayerMaxCheck
		fakeEngine        cli.PodmanEngine
	)

	BeforeEach(func() {
		layers := make([]string, 5)
		for i := range layers {
			layers[i] = fmt.Sprintf("layer%d", i)
		}
		fakeEngine = FakePodmanEngine{
			ImageInspectReport: cli.ImageInspectReport{
				Images: []cli.PodmanImage{
					{
						Id: "imageid",
						RootFS: cli.PodmanRootFS{
							Type:   "layers",
							Layers: layers,
						},
					},
				},
			},
		}
		podmanEngine = fakeEngine
	})

	Describe("Checking for less than max layers", func() {
		Context("When it has fewer layers than max", func() {
			It("should pass Validate", func() {
				ok, err := lessThanMaxLayers.Validate("dummy/image")
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		Context("When has more layers than max", func() {
			BeforeEach(func() {
				engine := fakeEngine.(FakePodmanEngine)
				layers := make([]string, 50)
				for i := range layers {
					layers[i] = fmt.Sprintf("layer%d", i)
				}
				engine.ImageInspectReport.Images[0].RootFS.Layers = layers
			})
			It("should not succeed the check", func() {
				ok, err := lessThanMaxLayers.Validate("dummy/image")
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
				ok, err := lessThanMaxLayers.Validate("dummy/image")
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
})
