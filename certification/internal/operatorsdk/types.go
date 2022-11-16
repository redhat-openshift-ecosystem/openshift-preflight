package operatorsdk

type OperatorSdkScorecardOptions struct {
	OutputFormat   string
	Selector       []string
	ResultFile     string
	Kubeconfig     []byte
	Namespace      string
	ServiceAccount string
	Verbose        bool
	WaitTime       string
}

type OperatorSdkScorecardReport struct {
	Stdout string
	Stderr string
	Items  []OperatorSdkScorecardItem `json:"items"`
}

type OperatorSdkScorecardItem struct {
	Status OperatorSdkScorecardStatus `json:"status"`
}

type OperatorSdkScorecardStatus struct {
	Results []OperatorSdkScorecardResult `json:"results"`
}

type OperatorSdkScorecardResult struct {
	Name  string `json:"name"`
	Log   string `json:"log"`
	State string `json:"state"`
}

type OperatorSdkBundleValidateOptions struct {
	LogLevel        string
	ContainerEngine string
	Selector        []string
	OptionalValues  map[string]string
	OutputFormat    string
	Verbose         bool
	WaitTime        string
}

type OperatorSdkBundleValidateReport struct {
	Stdout  string
	Stderr  string
	Passed  bool                              `json:"passed"`
	Outputs []OperatorSdkBundleValidateOutput `json:"outputs"`
}

type OperatorSdkBundleValidateOutput struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
