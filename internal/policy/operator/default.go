package operator

import (
	"time"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
)

const (
	// secretName is the K8s secret name which stores the auth keys for the private registry access
	secretName = "registry-auth-keys"

	// pipelineServiceAccount is the name of the service account which is used by the pipeline
	pipelineServiceAccount = "pipeline"

	// registryViewerRole is the name of the ClusterRole which is used by SA to view the image registry
	registryViewerRole = "registry-viewer"

	// imagePullerRole is the name of the ClusterRole which is used by SA to pull images from the image registry
	imagePullerRole = "system:image-puller"

	// openshiftMarketplaceNamespace is the project name for the default openshift marketplace
	openshiftMarketplaceNamespace = "openshift-marketplace"

	// errorPrefix is the prefix used by goroutines to send error messages on the same channel as the data
	errorPrefix = "error:"

	// imageRegistryService is the service name of the image registry
	imageRegistryService = "image-registry.openshift-image-registry.svc"
)

var (
	subscriptionTimeout time.Duration = 180 * time.Second

	csvTimeout time.Duration = 180 * time.Second

	approvedRegistries = map[string]struct{}{
		"registry.connect.dev.redhat.com":   {},
		"registry.connect.qa.redhat.com":    {},
		"registry.connect.stage.redhat.com": {},
		"registry.connect.redhat.com":       {},
		"registry.redhat.io":                {},
		"registry.access.redhat.com":        {},
	}

	prioritizedInstallModes = []operatorsv1alpha1.InstallModeType{
		operatorsv1alpha1.InstallModeTypeOwnNamespace,
		operatorsv1alpha1.InstallModeTypeSingleNamespace,
		operatorsv1alpha1.InstallModeTypeMultiNamespace,
		operatorsv1alpha1.InstallModeTypeAllNamespaces,
	}
)
