# Preflight

**Preflight** is a commandline interface for validating if
[OpenShift](https://www.openshift.com/) operator bundles meet minimum
reqiurements for [Red Hat OpenShift
Certification](https://connect.redhat.com/en/partner-with-us/red-hat-openshift-certification).

This project is in active and rapid development! Many facets of this project are
subject to change.

## Usage

```shell
A utility that allows you to pre-test your bundles, operators, and container before submitting for Red Hat Certification.
Choose from any of the following checks:
        HasLicense, HasUniqueTag, RunAsNonRoot, LayerCountAcceptable, HasMinimalVulnerabilities, HasNoProhibitedPackages, ValidateOperatorBundle, HasRequiredLabel, BasedOnUbi
Choose from any of the following output formats:
        json, xml, junitxml

Usage:
  preflight <container-image> [flags]

Flags:
  -c, --enabled-checks string   Which checks to apply to the image to ensure compliance.
                                (Env) PREFLIGHT_ENABLED_CHECKS
  -h, --help                    help for preflight
  -o, --output-format string    The format for the check test results.
                                (Env) PREFLIGHT_OUTPUT_FORMAT (Default) json
  -v, --version                 version for preflight
```

