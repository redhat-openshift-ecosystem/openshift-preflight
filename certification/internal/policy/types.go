package policy

type Definition struct {
	ValidatorFunc func(string) (bool, error)
	Metadata      Metadata
	HelpText      HelpText
}

func (pd *Definition) Validate(image string) (bool, error) {
	return pd.ValidatorFunc(image)
}

func (pd *Definition) Meta() Metadata {
	return pd.Metadata
}

func (pd *Definition) Help() HelpText {
	return pd.HelpText
}

type PolicyInfo struct {
	Metadata `json:"metadata" xml:"metadata"`
	HelpText `json:"helptext"`
}

// Metadata contains useful information regarding the policy
// being enforced
type Metadata struct {
	Description      string `json:"description" xml:"description"`
	Level            string `json:"level" xml:"level"`
	KnowledgeBaseURL string `json:"knowledge_base_url,omitempty" xml:"knowledgeBaseURL"`
	PolicyURL        string `json:"policy_url,omitempty" xml:"policyURL"`
}

// HelpText is the help message associated with any given policy.
type HelpText struct {
	Message    string `json:"message" xml:"message"`
	Suggestion string `json:"suggestion" xml:"suggestion"`
}
