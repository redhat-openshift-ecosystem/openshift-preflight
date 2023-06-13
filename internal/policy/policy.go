package policy

import "context"

type Policy = string

const (
	PolicyOperator  Policy = "operator"
	PolicyContainer Policy = "container"
	PolicyScratch   Policy = "scratch"
	PolicyRoot      Policy = "root"
)

// NewContext adds Policy p to the context ctx.
func NewContext(ctx context.Context, p Policy) context.Context {
	return context.WithValue(ctx, policyContextKey, p)
}

// FromContext returns the policy from the context, or empty string.
func FromContext(ctx context.Context) Policy {
	p := ctx.Value(policyContextKey)
	if policy, ok := p.(Policy); ok {
		return policy
	}

	return ""
}

// contextKey is a key used to store/retrieve Policy in/from context.Context.
type contextKey string

const policyContextKey contextKey = "Policy"
