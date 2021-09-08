package cli

import (
	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

type OpenshiftOptions struct {
	Namespace string
	Labels    map[string]string
}

type OpenshiftReport struct {
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
	CreateNamespace(name string, opts OpenshiftOptions, config *rest.Config) (*OpenshiftReport, error)
	DeleteNamespace(name string, opts OpenshiftOptions, config *rest.Config) error
	GetNamespace(name string, config *rest.Config) (*corev1.Namespace, error)

	CreateOperatorGroup(data OperatorGroupData, opts OpenshiftOptions, config *rest.Config) (*OpenshiftReport, error)
	DeleteOperatorGroup(name string, opts OpenshiftOptions, config *rest.Config) (*OpenshiftReport, error)
	GetOperatorGroup(name string, opts OpenshiftOptions, config *rest.Config) (*operatorv1.OperatorGroup, error)

	CreateCatalogSource(data CatalogSourceData, opts OpenshiftOptions, config *rest.Config) (*OpenshiftReport, error)
	DeleteCatalogSource(name string, opts OpenshiftOptions, config *rest.Config) (*OpenshiftReport, error)
	GetCatalogSource(name string, opts OpenshiftOptions, config *rest.Config) (*operatorv1alpha1.CatalogSource, error)

	CreateSubscription(data SubscriptionData, opts OpenshiftOptions, config *rest.Config) (*OpenshiftReport, error)
	DeleteSubscription(name string, opts OpenshiftOptions, config *rest.Config) (*OpenshiftReport, error)
	GetSubscription(name string, opts OpenshiftOptions, config *rest.Config) (*operatorv1alpha1.Subscription, error)

	GetCSV(name string, opts OpenshiftOptions, config *rest.Config) (*operatorv1alpha1.ClusterServiceVersion, error)
}
