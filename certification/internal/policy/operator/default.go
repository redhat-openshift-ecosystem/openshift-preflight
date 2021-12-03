package operator

import (
	"time"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
)

const (
	// packageKey is the packageKey in annotations.yaml that contains the package name.
	packageKey = "operators.operatorframework.io.bundle.package.v1"

	// channelKeyInBundle is the channel in annotations.yaml that contains the channel name.
	channelKeyInBundle = "operators.operatorframework.io.bundle.channel.default.v1"

	// IndexImageKey is the key in viper that contains the index (catalog) image URI
	indexImageKey = "indexImage"

	// channelKey is the key in viper that indicates the operator channel under test
	channelKey = "channel"

	// apiEndpoint is the endpoint used to query for package uniqueness.
	apiEndpoint = "https://catalog.redhat.com/api/containers/v1/operators/packages"

	// secretName is the K8s secret name which stores the auth keys for the private registry access
	secretName = "registry-auth-keys"

	// versionsKey is the OpenShift versions in annotations.yaml that lists the versions allowed for an operator
	versionsKey = "com.redhat.openshift.versions"

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

	prioritizedInstallModes = []string{
		string(operatorv1alpha1.InstallModeTypeOwnNamespace),
		string(operatorv1alpha1.InstallModeTypeSingleNamespace),
		string(operatorv1alpha1.InstallModeTypeMultiNamespace),
		string(operatorv1alpha1.InstallModeTypeAllNamespaces),
	}
)
