package csv

import (
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("CSV Functions", func() {
	DescribeTable(
		"Validating infrastructure features list disconnected",
		func(featurelist string, expected bool) {
			actual := SupportsDisconnected(featurelist)
			Expect(actual).To(Equal(expected))
		},
		Entry("disconnected in all caps", `["DISCONNECTED"]`, true),
		Entry("disconnected in all lower case", `["disconnected"]`, true),
		Entry("disconnected not found in list", `["foo"]`, false),
		Entry("list is completely empty", `[]`, false),
		Entry("data is not a list", `disconnected`, false),
	)

	DescribeTable(
		"Determining the presence of related images",
		func(imgs []operatorsv1alpha1.RelatedImage, expected bool) {
			csv := operatorsv1alpha1.ClusterServiceVersion{
				Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
					RelatedImages: imgs,
				},
			}
			Expect(HasRelatedImages(&csv)).To(Equal(expected))
		},
		Entry("one related image", []operatorsv1alpha1.RelatedImage{
			{
				Name:  "memcached",
				Image: "docker.io/library/memcached@sha256:00b68b00139155817a8b1d69d74865563def06b3af1e6fc79ac541a1b2f6b961",
			},
		}, true),
		Entry("multiple related images", []operatorsv1alpha1.RelatedImage{
			{
				Name:  "memcached",
				Image: "docker.io/library/memcached@sha256:00b68b00139155817a8b1d69d74865563def06b3af1e6fc79ac541a1b2f6b961",
			},
			{
				Name:  "redis",
				Image: "mirror.gcr.io/library/redis@sha256:4970a4bbd34f9072b56389e85185204dd07dc86ba74a1be441931d551f74b472",
			},
		}, true),
		Entry("no related image", []operatorsv1alpha1.RelatedImage{}, false),
	)

	When("Checking CSVs for the infrastructure-features annotation", func() {
		var csv operatorsv1alpha1.ClusterServiceVersion
		It("Should return false if the annotation is missing", func() {
			annotations := map[string]string{}
			csv.Annotations = annotations
			Expect(HasInfrastructureFeaturesAnnotation(&csv)).To(BeFalse())
		})

		It("Should return true if the annotation is present, regardless of its value", func() {
			annotations := map[string]string{InfrastructureFeaturesAnnotation: "foo"}
			csv.Annotations = annotations
			Expect(HasInfrastructureFeaturesAnnotation(&csv)).To(BeTrue())
		})
	})

	DescribeTable(
		"Determining the image values in related images are pinned",
		func(imgs []operatorsv1alpha1.RelatedImage, expected bool) {
			Expect(RelatedImagesArePinned(imgs)).To(Equal(expected))
		},
		Entry("a pinned image", []operatorsv1alpha1.RelatedImage{
			{
				Name:  "memcached",
				Image: "docker.io/library/memcached@sha256:00b68b00139155817a8b1d69d74865563def06b3af1e6fc79ac541a1b2f6b961",
			},
		}, true),
		Entry("a tagged image", []operatorsv1alpha1.RelatedImage{
			{
				Name:  "redis",
				Image: "mirror.gcr.io/library/redis:1.0.0",
			},
		}, false),
		Entry("no related image", []operatorsv1alpha1.RelatedImage{}, false),
		Entry("an image with no tag or digest", []operatorsv1alpha1.RelatedImage{
			{
				Name:  "redis",
				Image: "mirror.gcr.io/library/redis",
			},
		}, false),
	)

	DescribeTable("Extracting related images from the container spec environment",
		func(deps []appsv1.DeploymentSpec, expected []string) {
			actual := RelatedImageReferencesInEnvironment(deps...)
			sort.Strings(actual)
			sort.Strings(expected)
			Expect(actual).To(Equal(expected))
		},
		Entry(
			"deployment with 1 container with two refs", []appsv1.DeploymentSpec{
				{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "RELATED_IMAGE_FOO",
											Value: "foovalue",
										},
										{
											Name:  "RELATED_IMAGE_BAR",
											Value: "barvalue",
										},
									},
								},
							},
						},
					},
				},
			},
			[]string{"RELATED_IMAGE_FOO", "RELATED_IMAGE_BAR"},
		),
		Entry(
			"deployment with 1 container with two refs", []appsv1.DeploymentSpec{
				{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "RELATED_IMAGE_FOO",
											Value: "foovalue",
										},
										{
											Name:  "RELATED_IMAGE_BAR",
											Value: "barvalue",
										},
									},
								},
							},
						},
					},
				},
			},
			[]string{"RELATED_IMAGE_FOO", "RELATED_IMAGE_BAR"},
		),
		Entry(
			"deployment with 1 initContainer with two refs", []appsv1.DeploymentSpec{
				{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "RELATED_IMAGE_FOO",
											Value: "foovalue",
										},
										{
											Name:  "RELATED_IMAGE_BAR",
											Value: "barvalue",
										},
									},
								},
							},
						},
					},
				},
			},
			[]string{"RELATED_IMAGE_FOO", "RELATED_IMAGE_BAR"},
		),
		Entry(
			"deployment with 1 initContainer with two refs", []appsv1.DeploymentSpec{
				{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "RELATED_IMAGE_FOO",
											Value: "foovalue",
										},
										{
											Name:  "RELATED_IMAGE_BAR",
											Value: "barvalue",
										},
									},
								},
							},
						},
					},
				},
			},
			[]string{"RELATED_IMAGE_FOO", "RELATED_IMAGE_BAR"},
		),
	)
})
