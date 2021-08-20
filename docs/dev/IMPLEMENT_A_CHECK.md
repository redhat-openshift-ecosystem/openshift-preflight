# Check Implementation

Checks make up the core validation logic used in Preflight. Operators and
containers must pass all checks defined by the corresponding policy in order to
be considered certifiable by Red Hat.

All checks must fulfill the `Check` interface to be executed against a given
bundle or container image. For full documentation on what each method that must
be implemented to fulfill the check interface, please refer to the [package
documentation](https://pkg.go.dev/github.com/redhat-openshift-ecosystem/openshift-preflight/certification).

## Defining a new check

The Preflight project focuses on implementing all the checks necessary for a
given container or Operator to pass Red Hat Certification. As a result, the
project may not merge pull requests containing checks that are added to the
mentioned policies if those checks are not explicit requirements of the
certification pipeline.

We recommend opening an issue for guidance on a validation that you would like
to see implemented.

### Using `NewGenericCheck`

Simple checks can be written by simply defining a `ValidatorFunc` and all
surrounding metadata, and then feeding that information to `NewGenericCheck`.
What's returned is a `genericCheckDefinition` which implements the `Check`
interface.

This check can then be added to a policy and passed to a CheckEngine
implementation.

### Custom Structs

Defining a custom struct will provide you more flexibility in the execution of
your validation.

All checks built into the Preflight project are defined in the
`certification/internal` package as they are implementation details specific to
the project.

A new policy file might look something like this:

```go
package shell

type ValidateThingCheck struct{
    // your struct fields here
}

func (p *ValidateThingCheck) Validate(image certification.ImageReference) (bool, error) {
    // your logic here ...
}

func (p *ValidateThingCheck) Name() string {
	return "ValidateThing"
}

func (p *ValidateThingCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "A brief description of what your check is doing.",
		Level:            "best",
		KnowledgeBaseURL: "https://example.com/path/to/your/knowlebase/url", 
		CheckURL:         "https://example.com/path/to/your/check/url",
	}
}

func (p *ValidateThingCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Information about where to look when a user fails this check",
		Suggestion: "A quick tidbit on exactly how one might have passed this check",
	}
}
```

The container or bundle that is being validated is passed to each check in the
event that the information is necessary in performing the validation. All checks
are expected to standalone in their execution, and may not assume that they are
executed in a particular order, or with a particular shared context with other
checks.

With a new check defined, the check will then need to be registered with the
appropriate policy. A map of enabled checks for all policies exists in the
`engine` package.

Once the check is added to the appropriate package, the Preflight utility will
automatically executed the check when the associated policy is called by users.