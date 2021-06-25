# Developer Documentation

The Preflight project intends to offer tooling that can be used to evaluate your
operator projects to see if they pass Red Hat operator certification
requirements.

The project will include a commandline interface that will accept your operator
bundle image as an input, and will run validate that your operator bundle
complies with a configurable set of checks.

The project also has a goal of providing a library that can be leveraged to
check your operator bundle's certification compliance in your own testing use cases,
including writing and ensuring your bundle complies with your own custom tests.

## Design

The current design leverages a series of interfaces for handling the following
tasks related to check enforcement:

* managing container image assets (e.g. pulling from an external registry,
  managing an image tarball on disk, etc.)

* enforcing checks against the assets on disk, and storing the results of the
  checks

* formatting the results into various output formats to fit various known use
  cases.

The interface definitions for managing each of these tasks should allow for
developers to:

* define their own approach for managing container assets in their own test
  environments if the included approach is not preferred.

* define their own checks and implementation details in addition to the
  built-in checks

* define custom output formats other than those built-in to the tooling.

The included CLI will leverage these interfaces to provide built-in checks,
built-in formatters, and built-in container asset managers. 

## Libraries

The `certification` library contains the built-in check definitions that are
used to validate an operator bundle is in compliance with Red Hat's operator
bundle certification requirements. These are the built-in tests that can be
enabled when using the compiled binary.

The `certification/formatters` library includes the necessary constructs to
build out your own custom formatters. Developers can leverage the included
`certification.formatters.GenericFormatter` struct to build out custom
formatters by simply passing in a `certification/formatters.FormatterFunc` and
additional metadata, or they can build out their own by implementing the
`certification/formatters.ResponseFormatter` interface.

The `certification/inputmanager` library includes the necessary constructs to
build out your own custom input managers. Input management just refers to the
managing of container image assets on disk, and to/from remote registries if
needed.
*TODO: complete implementation and documentation*

The `certification/runtime` library includes the necessary constructs to build
out your own check runner. A check runner just refers to the interface that
registers what checks to execute, and to which asset.
*TODO: complete the reusable generic implementation and documentation*

## CLI Implementation

The Preflight CLI utilizes the libraries and design mentioned above. The bulk
of the CLI code is found in the `cmd` package.

Currently, we assume that the user must provide a single positional argument:
the container image which will be checked for compliance with our checks.

The user can then further tailor their execution by specifying configurables
such as the exact checks to enforce, and the output format. The CLI currently
allows for the definition of those configurables via flags (e.g.
`--output-format`), but environment variables are also parsed, with flag values
taking precedence. See `cmd/constants.go` for the existing supported environment
variables. 

The CLI will take all user input and derive a `certification/runtime.Config`
instance with the appropriate values filled in. Then, an inputmanager,
formatter, and checkrunner is derived based on that configuration (commonly
seen as `NewForConfig(...)` functions available in each package).

The inputmanager manager is then used to gather the required assets. The
runtime, will execute checks and store results, and finally the formatter will
prepare that output. The CLI will end execution by writing that output to
whatever output has been specified (`os.Stdout` today, but this will eventually
support additional locations).

## Input Manager Implementation

*TODO Currently Unimplemented*

## CheckRunner Implementation

The built-in checkrunner is referred to as `podmanexec` (this name may change
as we make calls to other tools as well). It issues calls out to `podman` and
other related tools directly in order to determine if the provided asset is in
compliance.

## Formatter Implementation

The built-in formatters allow for formatting output as JSON or XML. The final
user-facing response is still being defined.
