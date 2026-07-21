# Developer Documentation

The Preflight project intends to offer tooling that can be used to evaluate your
operator projects to see if they pass Red Hat operator certification
requirements.

The project will include a commandline interface that will accept your operator
bundle or container image as an input, and will run validate that your container
image or operator bundle complies with a series of validations.

## Requirements

Development and testing preflight requires that you have the following tools installed,
functional, and in your path.

| Name             | Tool cli          | Minimum version |
|----------------- |:-----------------:|----------------:|
| OperatorSDK      | `operator-sdk`    |         v1.42.3 |
| OpenShift Client | `oc`              |         v4.10.0 |
| Podman           | `podman`          |            v3.0 |

## Checks

The term "check" refers to a single validation executed against the given asset.
See our docs on [check implementation](IMPLEMENT_A_CHECK.md) to find out more
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

## Running with Podman

Preflight can be run with Podman. However, it requires use of '--privileged'

Steps to build and run the container:

1. `IMAGE_REPO=quay.io/myuser IMAGE_BUILDER=podman make image-build`
2. Run the image

`podman run --privileged -v /path/to/local/artifacts:/artifacts quay.io/myuser/preflight:<sha of commit> check container <container to be checked>`

or

`podman run --privileged -v ${KUBECONFIG}:/kubeconfig -e KUBECONFIG=/kubeconfig -v /path/to/local/artifacts:/artifacts quay.io/myuser/preflight:<sha of commit> check operator <bundle to be checked>`

## Updating the Operator SDK GPG Signing Key

The Operator SDK release signing key is vendored at `keys/operator-sdk-release.asc`
and used during the Docker build to verify the downloaded `operator-sdk` binary.
If the Operator SDK project rotates their signing key, update this file using the
Makefile target:

```bash
make update-operator-sdk-key
```

This fetches the key from `keyserver.ubuntu.com` using the fingerprint
`3B2F1481D146238080B346BB052996E2A20B5C7E` and exports it to `keys/operator-sdk-release.asc`.

If the key has been rotated to a new fingerprint:

```bash
make update-operator-sdk-key OPERATOR_SDK_GPG_FINGERPRINT=<new-fingerprint>
```

After updating the key, also update the fingerprint check in the `Dockerfile`
and the `OPERATOR_SDK_GPG_FINGERPRINT` default in the `Makefile` to match.

See the [Operator SDK installation docs](https://sdk.operatorframework.io/docs/installation/#2-verify-the-downloaded-binary)
for the upstream verification reference.
