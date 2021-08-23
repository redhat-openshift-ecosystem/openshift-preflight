package certification

// ValidatorFunc describes a function that, when executed, will check that an
// artifact (e.g. operator bundle) complies with a given check.
type ValidatorFunc = func(ImageReference) (bool, error)

type genericCheckDefinition struct {
	name        string
	validatorFn ValidatorFunc
	metadata    Metadata
	helpText    HelpText
}

func (pd *genericCheckDefinition) Name() string {
	return pd.name
}

func (pd *genericCheckDefinition) Validate(imgRef ImageReference) (bool, error) {
	return pd.validatorFn(imgRef)
}

func (pd *genericCheckDefinition) Metadata() Metadata {
	return pd.metadata
}

func (pd *genericCheckDefinition) Help() HelpText {
	return pd.helpText
}

// NewGenericCheck returns a basic check implementation with the provided
// inputs. This is to enable a quick way to add additional checks to the default
// checks already enforced.
//
// Developers can always define structs with internal keys and methods, and have that
// fulfill the Check interface. However, if no internal data or methods are needed,
// then this generic check provides an easier, purely-functional approach.
func NewGenericCheck(
	name string,
	validatorFn ValidatorFunc,
	metadata Metadata,
	helptext HelpText) Check {
	return &genericCheckDefinition{
		name:        name,
		validatorFn: validatorFn,
		metadata:    metadata,
		helpText:    helptext,
	}
}
