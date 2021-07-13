# Preflight

**Preflight** is a commandline interface for validating if
[OpenShift](https://www.openshift.com/) operator bundles and containers meet minimum
reqiurements for [Red Hat OpenShift
Certification](https://connect.redhat.com/en/partner-with-us/red-hat-openshift-certification).

This project is in active and rapid development! Many facets of this project are
subject to change, and some features are not fully implemented.

## Requirements

The preflight binary currently requires that you have the following tools installed,
functional, and in your path.

- OperatorSDK `operator-sdk`
- OpenShift Client `oc`
- Podman `podman`
- OpenSCAP `openscap-podman`
- Skopeo `skopeo`

See our [Vagrantfile](Vagrantfile) for more information on setting up a
development environment. Some checks may also requires access to an OpenShift
cluster. which is not provided by the Vagrantfile

## Usage

The `preflight` allows you to confirm that your container and Operator projects
comply with container and Operator certification policies.

A brief summary of the available subcommands is as follows:

```text
A utility that allows you to pre-test your bundles, operators, and container before submitting for Red Hat Certification.

Usage:
  preflight [command]

Available Commands:
  certify     Submits check results to Red Hat
  check       Run checks for an operator or container
  help        Help about any command
  support     Submits a support request
```

(note that `certify` and `support` subcommands are pending implementation)

To check a container, utilize the `check container` subcommand:

```text
preflight check container quay.io/example-namespace/example-container:0.0.1
```

The check an Operator bundle, utilize the `check Operator` subcommand:

```text
preflight check operator quay.io/example-namespace/example-operator:0.0.1
```
