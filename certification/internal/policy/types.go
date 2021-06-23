package policy

import "github.com/komish/preflight/certification"

type Definition struct {
	ValidatorFunc func(string) (bool, error)
	Metadata      certification.Metadata
	HelpText      certification.HelpText
}

func (pd *Definition) Validate(image string) (bool, error) {
	return pd.ValidatorFunc(image)
}

func (pd *Definition) Meta() certification.Metadata {
	return pd.Metadata
}

func (pd *Definition) Help() certification.HelpText {
	return pd.HelpText
}
