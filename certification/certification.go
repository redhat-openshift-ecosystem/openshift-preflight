package certification

// Check as an interface containing all methods necessary
// to use and identify a given check.
type Check interface {
	// Validate will test the provided image and determine whether the
	// image complies with the check's requirements.
	Validate(imageReference ImageReference) (result bool, err error)
	// Name returns the name of the check.
	Name() string
	// Metadata returns the check's metadata.
	Metadata() Metadata
	// Help return the check's help information
	Help() HelpText
}

// Metadata contains useful information regarding the check.
type Metadata struct {
	// Description contains a brief text detailing the overall goal of the check.
	Description string `json:"description" xml:"description"`
	// Level describes the certification level associated with the given check.
	//
	// TODO: define this more explicitly when requirements surrounding this metadata
	// text.
	Level string `json:"level" xml:"level"`
	// KnowledgeBaseURL is a URL detailing how to resolve a check failure.
	KnowledgeBaseURL string `json:"knowledge_base_url,omitempty" xml:"knowledgeBaseURL"`
	// CheckURL is a URL pointing to the official policy documentation from Red Hat, containing
	// information on exactly what is being tested and why.
	CheckURL string `json:"check_url,omitempty" xml:"checkURL"`
}

// HelpText is the help message associated with any given check
type HelpText struct {
	// Message is text provided to the user indicating where they should look
	// to find out why they failed or encountered an error in validation.
	Message string `json:"message" xml:"message"`
	// Suggestion is text provided to the user indicating what might need to
	// change in order to pass a check.
	Suggestion string `json:"suggestion" xml:"suggestion"`
}
