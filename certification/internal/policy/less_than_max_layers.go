package policy

import "github.com/komish/preflight/certification/errors"

func UnderLayerMax() *Definition {
	return &Definition{
		ValidatorFunc: isUnderLayerMaxValidatorFunc,
		Metadata:      isUnderLayerMaxPolicyMeta,
		HelpText:      isUnderLayerMaxPolicyHelp,
	}
}

var isUnderLayerMaxValidatorFunc = func(image string) (bool, error) {
	return false, errors.ErrFeatureNotImplemented
}

var isUnderLayerMaxPolicyMeta = Metadata{
	Description:      "Checking if container has less than 40 layers",
	Level:            "better",
	KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	PolicyURL:        "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
}

var isUnderLayerMaxPolicyHelp = HelpText{
	Message:    "Uncompressed container images should have less than 40 layers. Too many layers within the container images can degrade container performance.",
	Suggestion: "Optimize your Dockerfile to consolidate and minimize the number of layers. Each RUN command will produce a new layer. Try combining RUN commands using && where possible.",
}
