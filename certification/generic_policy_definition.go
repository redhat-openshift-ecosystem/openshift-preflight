package certification

import "github.com/sirupsen/logrus"

// ValidatorFunc describes a function that, when executed, will check that an
// artifact (e.g. operator bundle) complies with a given policy.
type ValidatorFunc = func(string, *logrus.Logger) (bool, error)

type genericPolicyDefinition struct {
	name        string
	validatorFn ValidatorFunc
	metadata    Metadata
	helpText    HelpText
}

func (pd *genericPolicyDefinition) Name() string {
	return pd.name
}

func (pd *genericPolicyDefinition) Validate(image string, logger *logrus.Logger) (bool, error) {
	return pd.validatorFn(image, logger)
}

func (pd *genericPolicyDefinition) Metadata() Metadata {
	return pd.metadata
}

func (pd *genericPolicyDefinition) Help() HelpText {
	return pd.helpText
}

// NewGenericPolicyDefinition returns a basic policy implementation with the provided
// inputs. This is to enable a quick way to add additional policies to the default
// policies already enforced.
//
// Developers can always define structs with internal keys and methods, and have that
// fulfill the Policy interface. However, if no internal data or methods are needed,
// then this generic policy provides an easier, purely-functional approach.
func NewGenericPolicyDefinition(
	name string,
	validatorFn ValidatorFunc,
	metadata Metadata,
	helptext HelpText) Policy {
	return &genericPolicyDefinition{
		name:        name,
		validatorFn: validatorFn,
		metadata:    metadata,
		helpText:    helptext,
	}
}
