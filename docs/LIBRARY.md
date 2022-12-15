# Using the Preflight Library

## Operator and Container Policy Execution

You can now apply the Operator and Container policy checks programmatically, in
the same way that the `preflight` cli executes tests.

Here is an annotated example execting the container policy. The operator policy
is very similar to this, but has a few additional required parameters.

```go
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/container"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/formatters"
)

func main() {
	myImage := "quay.io/opdev/simple-demo-operator:latest"
	imageAuthFilePath := "/path/to/your/dockerconfig.json"

	// [1] Configuring an Artifacts Writer to receive any files written by checks
	artifactsWriter, err := artifacts.NewMapWriter()
	logAndExitIfError(err)
	ctx := artifacts.ContextWithWriter(context.Background(), artifactsWriter)

	// [2] Configure a logr.Logger compliant logger of your choice, and add it to the context
	logbytes := bytes.NewBuffer([]byte{})
	checklogger := log.Default()
	checklogger.SetOutput(logbytes)
	logger := stdr.New(checklogger)
	ctx = logr.NewContext(ctx, logger)

	// [3] Creating an instance of a container check execution with a single option.
	containerCheck := container.NewCheck(myImage, container.WithDockerConfigJSONFromFile(imageAuthFilePath))
	results, err := containerCheck.Run(ctx)
	logAndExitIfError(err)

	// [4] Accessing logs
	fmt.Println("The preflight log contained:")
	fmt.Println(logbytes.String())

	fmt.Println()
	// [5] Accessing Artifacts
	fmt.Println("The cert-image.json artifact contains:")
	certimagereader := artifactsWriter.Files()["cert-image.json"]
	certimagebytes, err := io.ReadAll(certimagereader)
	logAndExitIfError(err)
	fmt.Println(string(certimagebytes))

	// [6] Formatting Results
	fr, err := formatAsText(ctx, results)
	logAndExitIfError(err)
	fmt.Println("The results:")
	fmt.Println(string(fr))

	fmt.Println("The final result was:", results.PassedOverall)
}

// logAndExitIfError emits an error to stderr and exits if it's not nil.
func logAndExitIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// formatAsText is a basic example of a FormatterFunc, which takes results and represents it as desired. This
// function simply shows passed, failed, and erroring checks with an equivalent prefix, and how long it took.
// Assigning this function to a variable here just serves to demonstrate that the formatters.FormatterFunc type
// can be used as a point of reference.
var formatAsText formatters.FormatterFunc = func(ctx context.Context, r certification.Results) (response []byte, formattingError error) {
	b := []byte{}
	for _, v := range r.Passed {
		t := v.ElapsedTime.Milliseconds()
		s := fmt.Sprintf("PASSED  %s in %dms\n", v.Name(), t)
		b = append(b, []byte(s)...)
	}
	for _, v := range r.Failed {
		t := v.ElapsedTime.Milliseconds()
		s := fmt.Sprintf("FAILED  %s in %dms\n", v.Name(), t)
		b = append(b, []byte(s)...)
	}
	for _, v := range r.Errors {
		t := v.ElapsedTime.Milliseconds()
		s := fmt.Sprintf("ERRORED %s in %dms\n", v.Name(), t)
		b = append(b, []byte(s)...)
	}

	return b, nil
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

[2] Preflight Logs

Preflight logic will reach into the context and extract a logr.Logger using
methods predefined by the same library. If not provided, the preflight logic
will discard log messages. Add a compliant logger to the context for your use
case.

[3]: Executing checks

The container and operator checks are instantiated by accepting required
parameters as well as configurable options. For a full reference of options,
review the container and operator package source files.

[4]: Accessing Logs

Logs are accessed using whatever method you provided in the [2] section. This
example used a byte buffer, and so logs that are written through execution are
retrieved through the same buffer. Modify this to fit your use case.a

[5]: Accessing Artifacts

Artifacts, as mentioned previously, are written by checks by reaching into the
`context` and extracting the provided `ArtifactWriter`. No functionality outside
of this interface will be used by checks.

Callers should rely on their instances of the `ArtifactWriter` to access
additional functionality in their implementations directly after checks have
been executed. For example, the `MapWriter` used in the example stores written
artifacts in a map of filenames to `io.Reader`.

[4]: Formatting results

Once checks have been executed, raw `runtime.Results` are returned. It is up to
the caller to format these results by whatever means necessary for their use
case. For reference, the `formatters` defines a FormattersFunc as a guide on how
a formatter function might be written. This definition is utilized for
formatters consumed internally by preflight as well.