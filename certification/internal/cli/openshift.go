package cli

import (
	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type OpenshiftOptions struct {
	Namespace string
	Labels    map[string]string
}

type SubscriptionData struct {
	Name                   string
	Channel                string
	CatalogSource          string
	CatalogSourceNamespace string
	Package                string
}

type CatalogSourceData struct {
	Name    string
	Image   string
	Secrets []string
}

type OperatorGroupData struct {
	Name             string
	TargetNamespaces []string
}

type OpenshiftEngine interface {
	CreateNamespace(name string, opts OpenshiftOptions) (*corev1.Namespace, error)
	DeleteNamespace(name string, opts OpenshiftOptions) error
	GetNamespace(name string) (*corev1.Namespace, error)

	CreateSecret(name string, content map[string]string, secretType corev1.SecretType, opts OpenshiftOptions) (*corev1.Secret, error)
	DeleteSecret(name string, opts OpenshiftOptions) error
	GetSecret(name string, opts OpenshiftOptions) (*corev1.Secret, error)

	CreateOperatorGroup(data OperatorGroupData, opts OpenshiftOptions) (*operatorv1.OperatorGroup, error)
	DeleteOperatorGroup(name string, opts OpenshiftOptions) error
	GetOperatorGroup(name string, opts OpenshiftOptions) (*operatorv1.OperatorGroup, error)

	CreateCatalogSource(data CatalogSourceData, opts OpenshiftOptions) (*operatorv1alpha1.CatalogSource, error)
	DeleteCatalogSource(name string, opts OpenshiftOptions) error
	GetCatalogSource(name string, opts OpenshiftOptions) (*operatorv1alpha1.CatalogSource, error)

	CreateSubscription(data SubscriptionData, opts OpenshiftOptions) (*operatorv1alpha1.Subscription, error)
	DeleteSubscription(name string, opts OpenshiftOptions) error
	GetSubscription(name string, opts OpenshiftOptions) (*operatorv1alpha1.Subscription, error)

	GetCSV(name string, opts OpenshiftOptions) (*operatorv1alpha1.ClusterServiceVersion, error)

	GetImages() (map[string]struct{}, error)
}
