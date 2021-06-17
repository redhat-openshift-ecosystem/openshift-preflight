package policy

import "github.com/komish/preflight/certification/errors"

func RunsAsNonRootUser() *Definition {
	return &Definition{
		ValidatorFunc: nonRootUserValidatorFunc,
		Metadata:      nonRootUserPolicyMeta,
		HelpText:      nonRootUserPolicyHelp,
	}
}

var nonRootUserValidatorFunc = func(image string) (bool, error) {
	return false, errors.ErrFeatureNotImplemented
}

var nonRootUserPolicyMeta = Metadata{
	Description:      "Checking if container runs as the root user",
	Level:            "best",
	KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	PolicyURL:        "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
}

var nonRootUserPolicyHelp = HelpText{
	Message:    "A container that does not specify a non-root user will fail the automatic certification, and will be subject to a manual review before the container can be approved for publication",
	Suggestion: "Indicate a specific USER in the dockerfile or containerfile",
}
