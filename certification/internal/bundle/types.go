package bundle

import "github.com/operator-framework/api/pkg/manifests"

type AnnotationsFile struct {
	Annotations Annotations `json:"annotations" yaml:"annotations"`
}

type Annotations struct {
	manifests.Annotations

	OpenshiftVersions string `json:"com.redhat.openshift.versions,omitempty" yaml:"com.redhat.openshift.versions,omitempty"`
}
