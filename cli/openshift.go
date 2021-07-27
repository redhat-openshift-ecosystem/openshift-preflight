package cli

import (
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/client-go/rest"
)

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
	CreateNamespace(name string, opts OpenShiftCliOptions, config *rest.Config) (*OpenshiftCreateReport, error)
	DeleteNamespace(name string, opts OpenShiftCliOptions, config *rest.Config) error
	CreateOperatorGroup(data OperatorGroupData, opts OpenShiftCliOptions, config *rest.Config) (*OpenshiftCreateReport, error)
	DeleteOperatorGroup(name string, opts OpenShiftCliOptions, config *rest.Config) error
	CreateCatalogSource(data CatalogSourceData, opts OpenShiftCliOptions, config *rest.Config) (*OpenshiftCreateReport, error)
	DeleteCatalogSource(name string, opts OpenShiftCliOptions, config *rest.Config) error
	CreateSubscription(data SubscriptionData, opts OpenShiftCliOptions, config *rest.Config) (*OpenshiftCreateReport, error)
	DeleteSubscription(name string, opts OpenShiftCliOptions, config *rest.Config) error
	GetSubscription(name string, opts OpenShiftCliOptions, config *rest.Config) (*operatorv1alpha1.Subscription, error)
	GetCSV(name string, opts OpenShiftCliOptions, config *rest.Config) (*operatorv1alpha1.ClusterServiceVersion, error)
}
