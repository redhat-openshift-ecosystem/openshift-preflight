package operator

const (
	// packageKey is the packageKey in annotations.yaml that contains the package name.
	packageKey = "operators.operatorframework.io.bundle.package.v1"

	// channelKey is the channel in annotations.yaml that contains the channel name.
	channelKey = "operators.operatorframework.io.bundle.channel.default.v1"

	// IndexImageKey is the key in viper that contains the index (catalog) image URI
	indexImageKey = "indexImage"

	// apiEndpoint is the endpoint used to query for package uniqueness.
	apiEndpoint = "https://catalog.redhat.com/api/containers/v1/operators/packages"

	// secretName is the K8s secret name which stores the auth keys for the private registry access
	secretName = "registry-auth-keys"

	// versionsKey is the OpenShift versions in annotations.yaml that lists the versions allowed for an operator
	versionsKey = "com.redhat.openshift.versions"
)
