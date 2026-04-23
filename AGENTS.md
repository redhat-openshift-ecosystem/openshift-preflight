# AGENTS.md

Guidance for agentic coding agents working in this repository.

## Project Overview

OpenShift Preflight is a Go CLI tool for Red Hat Software Certification that validates partner-submitted containers and operator bundles. Module: `github.com/redhat-openshift-ecosystem/openshift-preflight`.

## Build Commands

```bash
make build                    # Build the preflight binary
make build-multi-arch-linux   # Build for amd64, arm64, ppc64le, s390x
make build-multi-arch-mac     # Build for amd64, arm64 (Darwin)
```

All builds use `CGO_ENABLED=0` and `-trimpath`. Version metadata is injected via `-ldflags`.

## Test Commands

```bash
make test          # Run all unit tests (excludes e2e)
make cover         # Run tests with -race and coverage report
make test-e2e      # End-to-end tests (requires live OpenShift cluster)
```

The `-tags testing` build tag is **required** for all unit test runs.

### Running a Single Package's Tests

```bash
go test -v -tags testing \
  -ldflags "-X github.com/redhat-openshift-ecosystem/openshift-preflight/version.commit=bar \
            -X github.com/redhat-openshift-ecosystem/openshift-preflight/version.version=foo" \
  ./internal/policy/container/
```

### Running a Single Ginkgo Spec or Describe Block

Use `--ginkgo.focus` to filter by description string. Run from the package directory:

```bash
go test -v -tags testing \
  -ldflags "-X github.com/redhat-openshift-ecosystem/openshift-preflight/version.commit=bar \
            -X github.com/redhat-openshift-ecosystem/openshift-preflight/version.version=foo" \
  -run TestContainer \
  --ginkgo.focus="HasLicense" \
  ./internal/policy/container/
```

The `-run` value must match the `TestXxx` suite function name defined in the `*_suite_test.go` file for that package. Common mappings:

| `-run` flag            | Package path                        |
|------------------------|-------------------------------------|
| `TestInternalEngine`   | `./internal/engine/`                |
| `TestContainer`        | `./internal/policy/container/`      |
| `TestOperator`         | `./internal/policy/operator/`       |
| `TestRuntime`          | `./internal/runtime/`               |

## Code Quality Commands

```bash
make lint   # golangci-lint run --build-tags testing
make fmt    # gofumpt -l -w . (must produce no diff)
make vet    # go vet -tags testing ./...
make tidy   # go mod tidy (must produce no diff)
```

All PRs must pass lint, fmt, vet, and tidy with no diffs.

## Code Style

### Formatting

- Formatter: **gofumpt** (stricter than `gofmt`). Run `make fmt` before committing.
- Do not manually format; let `gofumpt` handle it.

### Import Organization

Three groups, separated by blank lines, enforced by `goimports` with local prefix:

```go
import (
    // 1. Standard library
    "context"
    "fmt"

    // 2. Third-party dependencies
    "github.com/go-logr/logr"
    "github.com/google/go-containerregistry/pkg/crane"

    // 3. Project-local (github.com/redhat-openshift-ecosystem/openshift-preflight/...)
    "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
    "github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
)
```

### Required Import Aliases

The `importas` linter enforces these aliases — use them exactly:

| Import path                                                      | Alias               |
|------------------------------------------------------------------|---------------------|
| `k8s.io/api/core/v1`                                             | `corev1`            |
| `k8s.io/apimachinery/pkg/apis/meta/v1`                           | `metav1`            |
| `k8s.io/apimachinery/pkg/api/errors`                             | `apierrors`         |
| `github.com/operator-framework/api/pkg/operators/v1alpha1`       | `operatorsv1alpha1` |
| `github.com/operator-framework/api/pkg/operators/v1`             | `operatorsv1`       |
| `github.com/openshift/api/image/v1`                              | `imagestreamv1`     |

Ginkgo and Gomega use dot imports in test files (allowed by lint config):
```go
import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)
```

### Naming Conventions

- Types and exported symbols: `PascalCase`
- Unexported implementation types: `camelCase` (e.g., `craneEngine`)
- Interfaces are exported; implementations may be unexported
- Check structs: name matches the `Name()` return (e.g., `HasLicenseCheck` → `"HasLicense"`)
- Constructors: `New<Type>(...)` returning the interface or concrete type
- Method receivers: short abbreviation of the receiver type (e.g., `p *HasLicenseCheck`, `c *craneEngine`)
- Package-level sentinel errors: `var errSomething = errors.New("...")`

### Error Handling

- Wrap errors with `fmt.Errorf("context: %w", err)` to preserve the chain
- Use `%v` for non-wrapped formatting, `%w` when callers may use `errors.Is`/`errors.As`
- Use `errors.Is(err, target)` for sentinel comparisons — never compare error strings
- Non-fatal errors (informational): log with `logger.Error(err, "message")` and continue
- Fatal errors: wrap and return up the call stack
- Validation failures (check did not pass, no code error): return `false, nil`

```go
// Compile-time interface satisfaction check — place near the type definition
var _ check.Check = &HasLicenseCheck{}
```

### Check Implementation Pattern

```go
var _ check.Check = &MyCheck{}

type MyCheck struct{}

func (p *MyCheck) Validate(ctx context.Context, imgRef image.ImageReference) (bool, error) { ... }
func (p *MyCheck) Name() string                  { return "MyCheck" }
func (p *MyCheck) Metadata() check.Metadata      { return check.Metadata{ ... } }
func (p *MyCheck) Help() check.HelpText          { return check.HelpText{ ... } }
func (p *MyCheck) RequiredFilePatterns() []string { return []string{...} }
```

## Testing Patterns (Ginkgo/Gomega)

Every package has a suite bootstrap file `*_suite_test.go`:

```go
package mypkg

import (
    "testing"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestMyPkg(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "MyPkg Suite")
}
```

Individual spec files follow this structure:

```go
var _ = Describe("MyCheck", func() {
    var check MyCheck

    BeforeEach(func() {
        check = MyCheck{}
    })

    Context("when the condition is met", func() {
        It("should pass validation", func() {
            ok, err := check.Validate(context.TODO(), imgRef)
            Expect(err).ToNot(HaveOccurred())
            Expect(ok).To(BeTrue())
        })
    })
})
```

- Use `Describe` / `Context` / `When` for nesting (semantically equivalent in Ginkgo)
- `It` for leaf test cases
- `DeferCleanup(...)` preferred over `AfterEach` for resource cleanup
- `DescribeTable` + `Entry` for table-driven tests
- Fakes: unexported structs implementing the target interface, defined in suite or spec file
- Kubernetes fakes: `sigs.k8s.io/controller-runtime/pkg/client/fake` builder
- Container image fakes: `github.com/google/go-containerregistry/pkg/v1/fake.FakeImage`

## Commit Conventions

Use conventional commits:

```
<subsystem>: <what changed>

Body explaining why the change was made.

Refs: #<issue-number>
```

Examples: `engine: add layer extraction span attributes`, `check: fix nil pointer in HasLicense`

## Git Hygiene

**Never use `git add .`** — always stage individual files explicitly:

```bash
# Good
git add internal/engine/engine.go internal/engine/engine_test.go

# Bad — may stage unintended artifacts
git add .
```

### GitHub Merge Strategy

This repository uses **rebase** for merging PRs. Squash and merge commits are disabled.

```bash
# Merge a PR
gh pr merge <number> --rebase
```

## Key Architecture Notes

- **Policies** (`internal/policy/`) define collections of checks for containers and operators
- **Engine** (`internal/engine/`) orchestrates check execution using `crane` for image operations
- **Checks** implement `internal/check.Check` interface: `Validate`, `Name`, `Metadata`, `Help`
- **Configuration** uses `PFLT_` env var prefix via Viper; see `internal/config/` and `internal/viper/`
- **Artifacts** output to `artifacts/` directory by default (`PFLT_ARTIFACTS` overrides)
- **Logging**: use the `logr` logger from context (`logr.FromContextOrDiscard(ctx)`); log levels are warn/info/debug/trace
