package policy

func BasedOnUBI() *Definition {
	return &Definition{
		ValidatorFunc: basedOnUBIValidatorFunc,
		Metadata:      basedOnUBIPolicyMeta,
		HelpText:      basedOnUBIPolicyHelp,
	}
}

var basedOnUBIValidatorFunc = func(image string) (bool, error) {
	return true, nil // placeholder!
}

var basedOnUBIPolicyMeta = Metadata{
	Description:      "Checking if the container's base image is based on UBI",
	Level:            "best",
	KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	PolicyURL:        "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
}

var basedOnUBIPolicyHelp = HelpText{
	Message:    "It is recommened that your image be based upon the Red Hat Universal Base Image (UBI)",
	Suggestion: "Change the FROM directive in your Dockerfile or Containerfile to FROM registry.access.redhat.com/ubi8/ubi",
}
