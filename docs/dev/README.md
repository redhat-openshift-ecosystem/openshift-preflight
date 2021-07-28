# Developer Documentation

The Preflight project intends to offer tooling that can be used to evaluate your
operator projects to see if they pass Red Hat operator certification
requirements.

The project will include a commandline interface that will accept your operator
bundle or container image as an input, and will run validate that your container
image or operator bundle complies with a series of validations.

## Checks

The term "check" refers to a single validation executed against the given asset.
See our docs on [check implementation](../IMPLEMENT_A_CHECK.md) to find out more
about how checks are implemented.

## Policies

The Preflight utility validates a given Operator or Container by applying a
series of validations (or "checks") against the asset. The term "policy" is used
to describe a collection of checks.

In order for a given asset (container or operator) to pass certification, it
must pass all checks defined in the corresponding policy.

The project has an [Operator
policy](https://github.com/redhat-openshift-ecosystem/openshift-preflight/blob/main/certification/engine/engine.go#L101)
and a [Container
policy](https://github.com/redhat-openshift-ecosystem/openshift-preflight/blob/main/certification/engine/engine.go#L101),
corresponding with the validations `preflight check` implements. Each
implemented policy has its own checks.
