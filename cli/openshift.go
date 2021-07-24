package cli

type OpenShiftCliOptions struct {
	Namespace string
	Labels    map[string]string
}

type OpenshiftCreateReport struct {
	Stdout string
	Stderr string
	Items  []OperatorSdkScorecardItem `json:"items"`
}

type SubscriptionData struct {
	Name                   string
	Channel                string
	CatalogSource          string
	CatalogSourceNamespace string
	Package                string
}

type CatalogSourceData struct {
	Name  string
	Image string
}

type OperatorGroupData struct {
	Name             string
	TargetNamespaces []string
}

type OpenshiftEngine interface {
	CreateNamespace(name string, opts OpenShiftCliOptions) (*OpenshiftCreateReport, error)
	DeleteNamespace(name string, opts OpenShiftCliOptions) error
	CreateOperatorGroup(data OperatorGroupData, opts OpenShiftCliOptions) (*OpenshiftCreateReport, error)
	DeleteOperatorGroup(name string, opts OpenShiftCliOptions) error
	CreateCatalogSource(data CatalogSourceData, opts OpenShiftCliOptions) (*OpenshiftCreateReport, error)
	DeleteCatalogSource(name string, opts OpenShiftCliOptions) error
	CreateSubscription(data SubscriptionData, opts OpenShiftCliOptions) (*OpenshiftCreateReport, error)
	DeleteSubscription(name string, opts OpenShiftCliOptions) error
}
