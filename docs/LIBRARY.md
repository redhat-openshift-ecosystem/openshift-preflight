# Using the Preflight Library

## Operator and Container Policy Execution

You can now apply the Operator and Container policy checks programmatically, in
the same way that the `preflight` cli executes tests.

Here is an annotated example execting the container policy. The operator policy
is very similar to this, but has a few additional required parameters.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/container"
)

func main() {
	myImage := "quay.io/opdev/simple-demo-operator:latest"
	imageAuthFilePath := "/path/to/your/dockerconfig.json"

	// [1] Configuring an Artifacts Writer to receive any files written by checks, the preflight log, etc.
	artifactsWriter, err := artifacts.NewMapWriter()
	logAndExitIfError(err)
	ctx := artifacts.ContextWithWriter(context.Background(), artifactsWriter)

	// [2] Creating an instance of a container check execution with a single option.
	containerCheck := container.NewCheck(myImage, container.WithDockerConfigJSONFromFile(imageAuthFilePath))
	results, err := containerCheck.Run(ctx)
	logAndExitIfError(err)

	// [3] Accessing logs, artifacts, results, etc.
	fmt.Println("The preflight log contained:")
	fmt.Println(artifactsWriter.Files()["preflight.log"])
	fmt.Println("The final result was:", results.PassedOverall)

	// [4] Using a Formatter to print results.
	fmtter, err := formatters.NewByName("json")
	logAndExitIfError(err)
	fr, err := fmtter.Format(ctx, results)
	logAndExitIfError(err)
	fmt.Println("The results in JSON format:")
	fmt.Println(string(fr))
}

// logAndExitIfError emits an error to stderr and exits if it's not nil.
func logAndExitIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

[1]: Writing Artifacts

Checks executed by preflight may want to write an "artifact", or some file that
contains data relevant to the execution of that specific check. These checks are
allowed to reach into the `context` and extract an artifact writer, and so
callers of the library should provide a relevant artifact writer into the
context when executing checks.

A custom Artifact Writer can be written so long as it implements the
`ArtifactWriter` interface definition.

[2]: Executing checks

The container and operator checks are instantiated by accepting required
parameters as well as configurable options. For a full reference of options,
review the container and operator package source files.

[3]: Accessing Artifacts and Logs

Artifacts, as mentioned previously, are written by checks by reaching into the
`context` and extracting the provided `ArtifactWriter`. No functionality outside
of this interface will be used by checks.

Callers should rely on their instances of the `ArtifactWriter` to access
additional functionality in their implementations directly after checks have
been executed. For example, the `MapWriter` used in the example stores written
artifacts in a map of filenames to `io.Reader`. The `preflight.log` is written
to the ArtifactWriter for library callers, and can be accessed once checks have
been executed.

A logger of choice should be added to the context with logr.NewContext. Otherwise,
no logs will be available from the library.

[4]: Formatting results

Once checks have been executed, raw `runtime.Results` are returned. It is up to
the caller to format these results by whatever means necessary for their use
case. The formatters used by Preflight's CLI (e.g. `"json"`) are available in
the `certification/formatters` package.

The `formatters.ResponseFormatter` interface is a common point of reference for
how a formatter should look. The `formatters.New` function can act as a bridge
to build a custom formatter quickly.