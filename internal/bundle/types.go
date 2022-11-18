package bundle

import (
	"github.com/operator-framework/api/pkg/manifests"
	validationerrors "github.com/operator-framework/api/pkg/validation/errors"
)

type AnnotationsFile struct {
	Annotations Annotations `json:"annotations" yaml:"annotations"`
}

type Annotations struct {
	manifests.Annotations

	OpenshiftVersions string `json:"com.redhat.openshift.versions,omitempty" yaml:"com.redhat.openshift.versions,omitempty"`
}

type Report struct {
	Results []validationerrors.ManifestResult
	Passed  bool
}
