// package k8s contains check implementations that rely on interacting with
// Kubernetes API server.
package k8s

import "github.com/redhat-openshift-ecosystem/openshift-preflight/cli"

// Create a package-level openshiftEngine variable, that can be overridden
// at the test level.
var (
	openshiftEngine cli.OpenshiftEngine
)

func init() {
	openshiftEngine = OpenshiftEngine{}
}
