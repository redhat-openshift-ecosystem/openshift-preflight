package policy

import "errors"

func HasRequiredLabels() *Definition {
	return &Definition{
		ValidatorFunc: hasRequiredLabelsValidatorFunc,
		Metadata:      hasRequiredLabelsPolicyMeta,
		HelpText:      hasRequiredLabelsPolicyHelp,
	}
}

var hasRequiredLabelsValidatorFunc = func(image string) (bool, error) {
	return false, errors.New("placeholder error of some kinds") // placeholder demonstrating error handling
}

var hasRequiredLabelsPolicyMeta = Metadata{
	Description:      "Checking if the container's base image is based on UBI",
	Level:            "best",
	KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	PolicyURL:        "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
}

var hasRequiredLabelsPolicyHelp = HelpText{
	Message:    "It is recommened that your image be based upon the Red Hat Universal Base Image (UBI)",
	Suggestion: "Change the FROM directive in your Dockerfile or Containerfile to FROM registry.access.redhat.com/ubi8/ubi",
}
