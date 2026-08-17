# How to Contribute

OpenShift Preflight is Apache 2.0 licensed and part of the [Red Hat Operator Ecosystem][operator_ecosystem_org]. Contributions are accepted via GitHub pull requests. This document outlines some of the conventions on commit message formatting, contact points for developers, and other resources to help get contributions into openshift-preflight.

## Contact

- Contact: [Red Hat Operator Ecosystem Google Group][operator_ecosystem_contact]  

## Getting Started

Preflight is a CLI (and Go library) that validates partner-submitted containers
and Operator bundles against Red Hat Software Certification requirements, via
a Container policy (`preflight check container <image>`) and an Operator
policy (`preflight check operator <bundle>`). A **check** is a single
validation; a **policy** is the set of checks an asset must pass. See the
[developer documentation][developer_docs] for more.

- Fork the repository on GitHub.
- Install the CLI tools listed under [Dev Setup][dev_setup_docs] and confirm
  they're on your `PATH`. None of them are dependencies of the compiled
  `preflight` binary itself — they're only needed to build test images, stand
  up test clusters, and exercise checks end-to-end. The [Vagrantfile](Vagrantfile)
  provides a ready-made dev VM.

## High-Level Codebase Map

- **Entry point**: `cmd/preflight/main.go` calls into the [Cobra](https://github.com/spf13/cobra)
  command tree rooted at `cmd/preflight/cmd`, which invokes the CLI runner in
  `internal/cli/cli.go`.
- **Public library API**: `container/check_container.go` and
  `operator/check_operator.go` expose `NewCheck(...).Run(ctx)` — used by both
  the CLI and external callers (see [Library Usage][library_docs]).
- **Engine**: `internal/engine/engine.go`'s `craneEngine` pulls the
  image/bundle with `crane`, extracts the filesystem content checks need, runs
  each check, and assembles results.
- **Checks**: implement the `check.Check` interface in
  `internal/check/certification.go` (`Validate`, `Name`, `Metadata`, `Help`,
  `RequiredFilePatterns`).
- **Policies**: named check sets keyed by the `Policy` constants in
  `internal/policy/policy.go`. Concrete checks live in
  `internal/policy/container` and `internal/policy/operator`.
- **Results & output**: raw results are `certification.Results`
  (`certification/results.go`), rendered via `formatters.FormatterFunc`
  (`formatters/formatters.go`); check-produced files go through the
  `ArtifactWriter` in `artifacts/artifacts.go`.

## Building and Testing Your Change

```make
make build                    # build the binary
make test                     # unit tests (requires -tags testing)
make lint fmt vet tidy        # must be clean before a PR is accepted
```

Run your change end-to-end against a real image or bundle before opening a
PR — see [Recipes][recipes_docs] for examples. Operator checks need a
`KUBECONFIG` for an OCP 4.10+ cluster with OLM (a local `crc` cluster works
well) and an index image containing your bundle (see
[Building an Index Image][index_image_docs]).

Once a PR is opened, unit tests run via GitHub Actions and E2E tests run
against a real OCP cluster via OpenShift CI/Prow. See [Testing][testing_docs]
for how these are wired up and what to expect.

## Adding a New Check

New checks are only merged when they map to an explicit certification
requirement — open an issue first to discuss before implementing one. See
[Implementing a Check][implement_check_docs] for the `Check` interface,
`NewGenericCheck` for simple validations, and how to register a check with
its policy.

## Reporting Bugs and Creating Issues

Reporting bugs is one of the best ways to contribute. However, a good bug report has some very specific qualities, so please read over the information below (and [these general guidelines][reporting_issues]) before submitting a bug report.

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

Error handling and logging follow a couple of project-specific rules:

- Use a sentinel `var Err... = errors.New(...)` only when callers need
  `errors.Is`; use `errors.As` for typed errors.
- Wrap with `fmt.Errorf("context: %w", err)` — don't wrap if it would leak
  implementation details.
- Log errors only at the top-most layer that handles them, to avoid duplicate
  log noise from wrapped errors.

See [Errors & Logging][errors_logging_docs] for more detail.

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
[library_docs]: https://github.com/redhat-openshift-ecosystem/openshift-preflight/tree/main/docs/LIBRARY.md
[dev_setup_docs]: https://github.com/redhat-openshift-ecosystem/openshift-preflight/tree/main/docs/dev/README.md#dev-setup
[recipes_docs]: https://github.com/redhat-openshift-ecosystem/openshift-preflight/tree/main/docs/RECIPES.md
[index_image_docs]: https://github.com/redhat-openshift-ecosystem/openshift-preflight/tree/main/docs/BUILDING_AN_INDEX.md
[testing_docs]: https://github.com/redhat-openshift-ecosystem/openshift-preflight/tree/main/docs/dev/TESTING.md
[implement_check_docs]: https://github.com/redhat-openshift-ecosystem/openshift-preflight/tree/main/docs/dev/IMPLEMENT_A_CHECK.md
[errors_logging_docs]: https://github.com/redhat-openshift-ecosystem/openshift-preflight/tree/main/docs/dev/ERRORS_AND_LOGGING.md
[reporting_issues]: https://sdk.operatorframework.io/docs/contribution-guidelines/reporting-issues/
[golang_style_doc]: https://github.com/golang/go/wiki/CodeReviewComments