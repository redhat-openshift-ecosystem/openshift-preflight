package certification

// Check as an interface containing all methods necessary
// to use and identify a given check.
type Check interface {
	// Validate checks whether the asset enforces the check.
	Validate(image string) (result bool, err error)
	// Name returns the name of the check.
	Name() string
	// Metadata returns the check's metadata.
	Metadata() Metadata
	// Help return the check's help text.
	Help() HelpText
}

// Metadata contains useful information regarding the check
type Metadata struct {
	Description      string `json:"description" xml:"description"`
	Level            string `json:"level" xml:"level"`
	KnowledgeBaseURL string `json:"knowledge_base_url,omitempty" xml:"knowledgeBaseURL"`
	CheckURL         string `json:"check_url,omitempty" xml:"checkURL"`
}

// HelpText is the help message associated with any given check
type HelpText struct {
	Message    string `json:"message" xml:"message"`
	Suggestion string `json:"suggestion" xml:"suggestion"`
}

type CheckInfo struct {
	Metadata `json:"metadata" xml:"metadata"`
	HelpText `json:"helptext"`
}
