
# Preflight

[![Build Status](https://github.com/redhat-openshift-ecosystem/openshift-preflight/actions/workflows/go.yml/badge.svg)](https://github.com/redhat-openshift-ecosystem/openshift-preflight/actions?workflow=go)
[![Coverage Status](https://coveralls.io/repos/github/redhat-openshift-ecosystem/openshift-preflight/badge.svg?branch=main)](https://coveralls.io/github/redhat-openshift-ecosystem/openshift-preflight?branch=main)

**Preflight** is a command line (CLI) tool to verify that partner-submitted containers meet minimum requirements for Red Hat Software Certification. These include:

- [OpenShift](https://www.openshift.com) Operator Bundles
- OpenShift containers
- [OpenStack](https://www.redhat.com/en/technologies/linux-platforms/openstack-platform) containers
- [Red Hat Enterprise Linux](https://connect.redhat.com/partner-with-us/red-hat-enterprise-linux-certification) application containers

This project is in active and rapid development! Many facets of this project are
subject to change, and some features are not fully implemented.

## Certification Workflow Guide

For the complete container and operator bundle certification workflow instructions, please reference our 
[official certification documentation](https://access.redhat.com/documentation/en-us/red_hat_software_certification/2024/html/red_hat_software_certification_workflow_guide/index).

## Requirements

For running the Preflight binary, the host or VM must have at least RHEL 8.5, CentOS 8.5 or Fedora 35 installed.

The Preflight binary currently requires that you have the following tools installed,
functional, and in your path.

| Name             | Tool cli          | Minimum version |
|----------------- |:-----------------:|----------------:|
| OperatorSDK      | `operator-sdk`    |         v1.40.0 |

See our [Vagrantfile](Vagrantfile) for more information on setting up a
development environment. Some checks may also require access to an OpenShift
cluster. which is not provided by the Vagrantfile

## Usage

The `preflight` utility allows one to confirm that container and Operator projects
comply with container and Operator certification policies.

A brief summary of the available sub-commands is as follows:

```text
A utility that allows you to pre-test your bundles, operators, and container before submitting for Red Hat Certification.

Usage:
  preflight [command]

Available Commands:
  check          Run checks for an operator or container
  completion     Generate the autocompletion script for the specified shell
  help           Help about any command
  runtime-assets Returns information about assets used at runtime.
  support        Submits a support request

Flags:
  -h, --help      help for preflight
  -v, --version   version for preflight

Use "preflight [command] --help" for more information about a command.
```

To check a container, utilize the `check container` sub-command:

```text
preflight check container quay.io/example-namespace/example-container:0.0.1 \
--pyxis-api-token=abcdefghijklmnopqrstuvwxyz123456 \
--certification-component-id=1234567890a987654321bcde 
```

To check an Operator bundle, utilize the `check Operator` sub-command:

```text
preflight check operator quay.io/example-namespace/example-operator:0.0.1
```

For more detailed usage examples, see [Recipes](docs/RECIPES.md).

For more information on how to configure the execution of `preflight`, see
[CONFIG](docs/CONFIG.md)

### Authenticating to Registries

If a registry requires authentication, one must set the environment variable
`PFTL_DOCKERCONFIG` or pass the `--docker-config` parameter on the command line.
This should be the full path to a properly formatted Docker config.json.

#### Remote Checks

In some cases (e.g. *DeployableByOLM*), `preflight` will also pass credentials
to the cluster used for testing (i.e. the cluster that is accessible through the
current-context of the provided `KUBECONFIG`).

We anticipate that the credentials in `${DOCKER_CONFIG}/config.json` or 
`${XDG_RUNTIME_DIR}/containers/auth.json` may contain more access than what is
needed for `preflight` execution. It is recommended to generate a dockerconfigjson
with only the credentials necessary to retrieve the image under test to avoid 
passing more credentials than needed into a cluster for those checks. `preflight`
accepts a full path to a dockerconfigjson that would be passed through to a remote
cluster via the `PFLT_DOCKERCONFIG` environment variable or the `--docker-config`
command line parameter.

If this variable is unset, `preflight` will assume that the images in scope
(e.g. PFLT_INDEXIMAGE value, and the test target itself) are located in a public
registry and already accessible from the cluster used for testing.

## Installation

Before installing `preflight`, ensure that the [required dependencies](#requirements) have been installed on the local machine.

### Install Prebuilt Release

One of the prebuilt [release binaries][releases_link] for the supported
architectures can be downloaded and installed to the local machine.

### Install From Source

Once the repository has been cloned locally, the `preflight` binary can be built
from source by using the provided target from within the root of the project directory.

```bash
make build
```

The `preflight` binary will be created in the root of the project directory. The
binary can then be copied manually to a location in the local `$PATH`.

```bash
sudo mv preflight /usr/local/bin/
```

Verify that the `preflight` binary can run successfully.

```bash
preflight --version
```

The version information should be displayed.

```bash
preflight version 0.0.0 <commit: 2d3bb671bff8a95d385621382f31215234877d44>
```

[releases_link]:https://github.com/redhat-openshift-ecosystem/openshift-preflight/releases

## Unit Tests
Execute the below command if you want to run all the unit tests in this repository.

Run all the unit tests:
```bash
make test
```

## End-to-end Tests
Execute the below command if you want to run all the end-to-end tests in this repository. **Note:** Running these test
requires a running OpenShift cluster.

Run all the unit tests:
```bash
make test-e2e
```

## How to Contribute

Check out the [contributor documentation][contribution_docs].

## Signed Images

This repository uses the [Cosign](https://github.com/sigstore/cosign) project to
sign release images stored in `quay.io/opdev/preflight` using [Keyless
Signatures](https://docs.sigstore.dev/cosign/keyless/). The verification
workflow is documented in the `verify-image` Make target for those that want to
confirm the images locally

```shell
RELEASE_TAG=1.5.4 make verify-image
```

## License

Operator SDK is under Apache 2.0 license. See the [LICENSE][license_file] file
for details.

[contribution_docs]: ./CONTRIBUTING.md
[license_file]:./LICENSE
