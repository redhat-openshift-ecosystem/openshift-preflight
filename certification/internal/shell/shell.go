// package shell contains check implementations that rely on utilizing
// shell commands directly through the use of cmd.Exec. This implies that the
// various shell tools are installed.
package shell

import "github.com/redhat-openshift-ecosystem/openshift-preflight/cli"

// Create a package-level podmanEngine variable, that can be overridden
// at the test level.
var (
	podmanEngine      cli.PodmanEngine
	skopeoEngine      cli.SkopeoEngine
	operatorSdkEngine cli.OperatorSdkEngine
)

func init() {
	podmanEngine = PodmanCLIEngine{}
	skopeoEngine = SkopeoCLIEngine{}
	operatorSdkEngine = OperatorSdkCLIEngine{}
}
