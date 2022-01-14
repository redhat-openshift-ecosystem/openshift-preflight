# Testing

All pull requests will be subject to the tests described below.

## Unit

Unit tests are executed using [GitHub
Actions](https://github.com/redhat-openshift-ecosystem/openshift-preflight/actions).

Our workflows can be found [here](https://github.com/redhat-openshift-ecosystem/openshift-preflight/tree/main/.github/workflows)

## End-to-End

E2E testing will be executed against all pull requests made against the main
branch before it is merged.

E2E testing will perform the following series of tasks against the code base
containing the code contributions:

- Build Preflight from source containing the contributed changes.

- Run preflight `check operator` against a known good
  [asset](https://github.com/opdev/simple-demo-operator) against supported OCP
  versions, and expect a passing result.

_TODO: add/test additional subcommands_

### Where are these tests defined?

These tests are executed using [OpenShift CI](https://docs.ci.openshift.org/),
and are defined in the
[openshift/release](https://github.com/openshift/release/) repository.

OpenShift CI utilizes
[Prow](https://github.com/kubernetes/test-infra/tree/master/prow) in addition to
the [CI-Operator](https://github.com/openshift/ci-tools) to provide access to
OCP clusters on-demand.

Our presubmit definitions can be found
[here](https://github.com/openshift/release/tree/master/ci-operator/config/redhat-openshift-ecosystem/openshift-preflight).
These encompass all of our test definitions that require the use of an OCP
cluster, such as our E2E test suite.

The test definitions utilize our testing scripts and tooling found in
[/test/e2e](../../test/e2e/).

### What version of OCP are used for testing?

There is a corresponding presubmit definition for each version of OCP that is
currently used as a cluster-target for testing. As those may change over time,
it's best to refer to the [presubmit
definitions](https://github.com/openshift/release/tree/master/ci-operator/config/redhat-openshift-ecosystem/openshift-preflight).

### High-Level Workflow

This is a succinct version of
[this](https://github.com/kubernetes/community/blob/master/contributors/guide/owners.md#the-code-review-process)
process.

- Contributor submits a pull request against the `main` branch.
- An [OWNER](../../OWNERS) will add an `/ok-to-test` comment to
  start
- Tests will execute against the pull request. If they fail, contributors may
  need to push additional changes.
- If tests pass, OWNERS can do any further review and then add a
  `/lgtm` comment when they're happy with the changes.
- If enough maintainers have approved the issue, the patch can be merged.

### Additional Information

- If needed, a maintainer can `/override` a test case to bypass it. This should
  be used sparingly.
- The "Tide" plugin may be enabled at a future date, turning on auto-merging
  when enough maintainers have approved a request, and it has passed tests. This
  was not enabled out of the gate so that maintainers can get used to the bot's
  workflow. See [this PR](https://github.com/openshift/release/pull/25043/files)
  for notes on how to enable this.
