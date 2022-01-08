# How to Contribute

OpenShift Preflight is Apache 2.0 licensed and part of the [Red Hat Operator Ecosystem][operator_ecosystem_org]. Contributions are accepted via GitHub pull requests. This document outlines some of the conventions on commit message formatting, contact points for developers, and other resources to help get contributions into openshift-preflight.

## Contact

- Contact: [Red Hat Operator Ecosystem Google Group][operator_ecosystem_contact]  

## Getting Started

- Fork the repository on GitHub
- See the [developer documentation][developer_docs] for the project overview and build instructions.

## Reporting Bugs and Creating Issues

Reporting bugs is one of the best ways to contribute. However, a good bug report has some very specific qualities, so please read over the information below before submitting a bug report.

If any part of the openshift-preflight project has bugs or documentation mistakes, please let us know by opening an issue. We treat bugs and mistakes very seriously and believe no issue is too small. Before creating a bug report, please check that an issue reporting the same problem does not already exist.

To make the bug report accurate and easy to understand, please try to create bug reports that are:

- Specific. Include as many details as possible: which version, what environment, what configuration, etc.
- Reproducible. Include the steps to reproduce the problem. We understand some issues might be hard to reproduce, please include the steps that might lead to the problem.
- Isolated. Please try to isolate and reproduce the bug with minimum dependencies. It would significantly slow down the speed to fix a bug if too many dependencies are involved in a bug report. Debugging external systems that rely on openshift-preflight is out of scope, but we are happy to provide guidance in the right direction or help with using openshift-preflight itself.
- Unique. Do not duplicate the existing bug report.
- Scoped. One bug per report. Do not follow up with another bug inside one report.

It may be worthwhile to read Elika Etemad’s article on filing good bug reports before creating a bug report.

We might ask for further information to locate a bug. A duplicated bug report will be closed.

## Contribution Flow

This is a rough outline of what a contributor's workflow looks like:

- Create a topic branch from where to base the contribution. This is usually main.
- Make commits of logical units.
- Make sure commit messages are in the proper format (see below).
- Push changes in a topic branch to a personal fork of the repository.
- Submit a pull request to redhat-openshift-ecosystem/openshift-preflight.
- The PR must receive an LGTM from two maintainers found in the MAINTAINERS file.

Thanks for contributing!

### Code Style

The coding style suggested by the Go community is used in openshift-preflight. See the [style doc][golang_style_doc] for details.

Please follow this style to make openshift-preflight easy to review, maintain and develop.

### Commit Message Format

We follow a rough convention for commit messages that is designed to answer two
questions: what changed and why. The subject line should feature the what and
the body of the commit should describe the why.

```
cmd: add the certify sub-command

this adds the certify sub-command to submit test results to Red Hat for certification.

Fixes #61
```

The format can be described more formally as follows:

```
<subsystem>: <what changed>
<BLANK LINE>
<why this change was made>
<BLANK LINE>
<footer>
```

The first line is the subject and should be no longer than 70 characters, the second line is always blank, and other lines should be wrapped at 80 characters. This allows the message to be easier to read on GitHub as well as in various git tools.

[operator_ecosystem_contact]: https://groups.google.com/g/red-hat-operator-ecosystem
[operator_ecosystem_org]: https://github.com/redhat-openshift-ecosystem
[developer_docs]: https://github.com/redhat-openshift-ecosystem/openshift-preflight/tree/main/docs/dev
[reporting_issues]: https://sdk.operatorframework.io/docs/contribution-guidelines/reporting-issues/
[golang_style_doc]: https://github.com/golang/go/wiki/CodeReviewComments
