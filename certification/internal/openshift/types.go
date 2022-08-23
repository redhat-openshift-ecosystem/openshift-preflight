package openshift

import (
	"context"

	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

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

type RoleBindingData struct {
	Name      string
	Subjects  []string
	Role      string
	Namespace string
}

type Client interface {
	CreateNamespace(ctx context.Context, name string) (*corev1.Namespace, error)
	DeleteNamespace(ctx context.Context, name string) error
	GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error)
	CreateSecret(ctx context.Context, name string, content map[string]string, secretType corev1.SecretType, namespace string) (*corev1.Secret, error)
	DeleteSecret(ctx context.Context, name string, namespace string) error
	GetSecret(ctx context.Context, name string, namespace string) (*corev1.Secret, error)
	CreateOperatorGroup(ctx context.Context, data OperatorGroupData, namespace string) (*operatorsv1.OperatorGroup, error)
	DeleteOperatorGroup(ctx context.Context, name string, namespace string) error
	GetOperatorGroup(ctx context.Context, name string, namespace string) (*operatorsv1.OperatorGroup, error)
	CreateCatalogSource(ctx context.Context, data CatalogSourceData, namespace string) (*operatorsv1alpha1.CatalogSource, error)
	DeleteCatalogSource(ctx context.Context, name string, namespace string) error
	GetCatalogSource(ctx context.Context, name string, namespace string) (*operatorsv1alpha1.CatalogSource, error)
	CreateSubscription(ctx context.Context, data SubscriptionData, namespace string) (*operatorsv1alpha1.Subscription, error)
	DeleteSubscription(ctx context.Context, name string, namespace string) error
	GetSubscription(ctx context.Context, name string, namespace string) (*operatorsv1alpha1.Subscription, error)
	GetCSV(ctx context.Context, name string, namespace string) (*operatorsv1alpha1.ClusterServiceVersion, error)
	GetImages(ctx context.Context) (map[string]struct{}, error)
	CreateRoleBinding(ctx context.Context, data RoleBindingData, namespace string) (*rbacv1.RoleBinding, error)
	GetRoleBinding(ctx context.Context, name string, namespace string) (*rbacv1.RoleBinding, error)
	DeleteRoleBinding(ctx context.Context, name string, namespace string) error
}
