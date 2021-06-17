package certification

import "github.com/komish/preflight/certification/internal/policy"

// PolicyFunc is a function that returns a policy.
type PolicyFunc = func() Policy

// Policy as an interface containing all methods necessary
// to use and identify a given policy.
type Policy interface {
	// Validate identify whether the asset enforces
	// the policy.
	Validate(image string) (result bool, err error)
	Meta() policy.Metadata
	// Help returns the policy's help information.
	Help() policy.HelpText
}

// PolicyMap maps policy string names to policy functions.
var PolicyMap = map[string]PolicyFunc{
	"under_40_layers":     PolicyHasLessThan40Layers,
	"is_ubi_based":        PolicyIsUBIBased,
	"nonroot":             PolicyRunsAsNonRootUser,
	"has_required_labels": PolicyImageHasRequiredLabels,
}

// AllPolicies returns all policies made available by this library.
func AllPolicies() []string {
	all := make([]string, len(PolicyMap))
	i := 0

	for k := range PolicyMap {
		all[i] = k
		i++
	}

	return all
}

// PolicyHasLessThan40Layers checks the container image and ensures
// it has less than 40 layers.
func PolicyHasLessThan40Layers() Policy {
	return policy.UnderLayerMax()
}

// PolicyIsUBIBased checks the container image and ensures
// it is based on the Red Hat Universal Base Image.
func PolicyIsUBIBased() Policy {
	return policy.BasedOnUBI()
}

// PolicyRunsAsNonRootUser checks that the container image is not
// configured to run as the root user
func PolicyRunsAsNonRootUser() Policy {
	return policy.RunsAsNonRootUser()
}

// PolicyImageHasRequiredLabels checks that the container image has
// the required labels
func PolicyImageHasRequiredLabels() Policy {
	return policy.HasRequiredLabels()
}
