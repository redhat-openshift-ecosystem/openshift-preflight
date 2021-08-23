package migration

import "github.com/redhat-openshift-ecosystem/openshift-preflight/certification"

// ImageToImageReference converts an image string, used in early Check.Validate
// interface definitions and replaces it with an ImageReference used in later
// implementations.
func ImageToImageReference(image string) certification.ImageReference {
	return certification.ImageReference{ImageURI: image}
}
