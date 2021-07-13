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

type OperatorSdkEngine interface {
	Scorecard(image string, opts OperatorSdkScorecardOptions) (*OperatorSdkScorecardReport, error)
}
