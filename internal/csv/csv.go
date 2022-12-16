package csv

import (
	"encoding/json"
	"strings"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-manifest-tools/pkg/imagename"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const InfrastructureFeaturesAnnotation = "operators.openshift.io/infrastructure-features"

// SupportsDisconnected accepts a stringified list of supported features
// and returns true if "disconnected" is listed as a supported feature.
//
// E.g. '["disconnected"]'.
//
// This is case insensitive, as each infrastructure
// is normalized before checking. A failure to unmarshal this structure
// returns false.
func SupportsDisconnected(infrastructureFeatures string) bool {
	var features []string

	err := json.Unmarshal([]byte(infrastructureFeatures), &features)
	if err != nil {
		return false
	}

	for _, feature := range features {
		if strings.ToLower(feature) == "disconnected" {
			return true
		}
	}

	return false
}

// HasInfrastructureFeaturesAnnotation returns true if the infrastructure-features annotation
// exists in the .metadata.annotations block of csv.
func HasInfrastructureFeaturesAnnotation(csv *operatorsv1alpha1.ClusterServiceVersion) bool {
	_, ok := csv.GetAnnotations()[InfrastructureFeaturesAnnotation]
	return ok
}

// HasRelatedImages returns true if the length of the .spec.relatedImages section of
// csv is greater than 0.
func HasRelatedImages(csv *operatorsv1alpha1.ClusterServiceVersion) bool {
	return len(csv.Spec.RelatedImages) > 0
}

// RelatedImagesArePinned returns true if all related images are digest-references.
// If a reference cannot be parsed, this returns false. If relatedImage is empty,
// this returns false.
func RelatedImagesArePinned(relatedImages []operatorsv1alpha1.RelatedImage) bool {
	if len(relatedImages) == 0 {
		return false
	}

	for _, ri := range relatedImages {
		img := imagename.Parse(ri.Image)
		if !img.HasDigest() {
			return false
		}
	}

	return true
}

// RelatedImageReferencesInEnvironment returns a slice of environment variables that are
// prefixed with RELATED_IMAGE_, which is the typical way to push these values into
// the controller's environment.
func RelatedImageReferencesInEnvironment(deployments ...appsv1.DeploymentSpec) []string {
	values := []string{}

	for _, depl := range deployments {
		values = append(values, relatedImageReferencesInContainerEnvironment(depl.Template.Spec.Containers)...)
		values = append(values, relatedImageReferencesInContainerEnvironment(depl.Template.Spec.InitContainers)...)
	}

	return values
}

func relatedImageReferencesInContainerEnvironment(containers []corev1.Container) []string {
	values := []string{}

	for _, container := range containers {
		envs := container.Env
		for _, env := range envs {
			if strings.HasPrefix(env.Name, "RELATED_IMAGE_") {
				values = append(values, env.Name)
			}
		}
	}

	return values
}
