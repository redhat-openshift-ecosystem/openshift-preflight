package cli

type OperatorSdkScorecardOptions struct {
	LogLevel     string
	OutputFormat string
	Selector     []string
	ResultFile   string
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
	LogLevel         string
	ContainerEnginer string
	Selector         []string
	OutputFormat     string
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

type OperatorSdkEngine interface {
	Scorecard(image string, opts OperatorSdkScorecardOptions) (*OperatorSdkScorecardReport, error)
	BundleValidate(image string, opts OperatorSdkBundleValidateOptions) (*OperatorSdkBundleValidateReport, error)
}
