package openshift

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"

	imagestreamv1 "github.com/openshift/api/image/v1"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type openshiftClient struct {
	Client crclient.Client
}

// NewClient provides a wrapper around the passed in client in
// order to present convenience functions for each of the object
// types that are interacted with.
func NewClient(client crclient.Client) Client {
	var osclient Client = &openshiftClient{
		Client: client,
	}
	return osclient
}

func AddSchemes(scheme *apiruntime.Scheme) error {
	if err := operatorsv1.AddToScheme(scheme); err != nil {
		return err
	}
	if err := operatorsv1alpha1.AddToScheme(scheme); err != nil {
		return err
	}
	if err := imagestreamv1.AddToScheme(scheme); err != nil {
		return err
	}
	if err := rbacv1.AddToScheme(scheme); err != nil {
		return err
	}
	return nil
}

// CreateNamespace can return an ErrAlreadyExists
func (oe *openshiftClient) CreateNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("creating namespace", "namespace", name)
	nsSpec := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	err := oe.Client.Create(ctx, &nsSpec, &crclient.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return &nsSpec, fmt.Errorf("could not create namespace: %s: %w: %v", name, ErrAlreadyExists, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not create namespace: %s: %v", name, err)
	}
	return &nsSpec, nil
}

func (oe *openshiftClient) DeleteNamespace(ctx context.Context, name string) error {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("deleting namespace", "namespace", name)
	nsSpec := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	err := oe.Client.Delete(ctx, &nsSpec, &crclient.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("could not delete namespace: %s: %v", name, err)
	}

	return nil
}

// GetNamespace can return am ErrNotFound
func (oe *openshiftClient) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("fetching namespace", "namespace", name)
	nsSpec := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name: name,
	}, &nsSpec)
	if apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("could not retrieve namespace: %s: %w: %v", name, ErrNotFound, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve namespace: %s: %v", name, err)
	}
	return &nsSpec, nil
}

// CreateOperatorGroup can return an ErrAlreadyExists
func (oe *openshiftClient) CreateOperatorGroup(ctx context.Context, data OperatorGroupData, namespace string) (*operatorsv1.OperatorGroup, error) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("creating OperatorGroup", "namespace", namespace, "name", data.Name)
	operatorGroup := &operatorsv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name,
			Namespace: namespace,
		},
		Spec: operatorsv1.OperatorGroupSpec{
			TargetNamespaces: data.TargetNamespaces,
		},
	}
	err := oe.Client.Create(ctx, operatorGroup)
	if apierrors.IsAlreadyExists(err) {
		return operatorGroup, fmt.Errorf("could not create operatorgroup: %s/%s: %w: %v", namespace, data.Name, ErrAlreadyExists, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not create operatorgroup: %s/%s: %v", namespace, data.Name, err)
	}

	return operatorGroup, nil
}

func (oe *openshiftClient) DeleteOperatorGroup(ctx context.Context, name string, namespace string) error {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("deleting OperatorGroup", "namespace", namespace, "name", name)
	operatorGroup := operatorsv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := oe.Client.Delete(ctx, &operatorGroup)
	if err != nil {
		return fmt.Errorf("could not delete operatorgroup: %s/%s: %v", namespace, name, err)
	}

	return nil
}

// GetOperatorGroup can return an ErrNotFound
func (oe *openshiftClient) GetOperatorGroup(ctx context.Context, name string, namespace string) (*operatorsv1.OperatorGroup, error) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("fetching operatorgroup", "namespace", namespace, "name", name)
	operatorGroup := operatorsv1.OperatorGroup{}
	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, &operatorGroup)
	if apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("could not retrieve operatorgroup: %s/%s: %w: %v", namespace, name, ErrNotFound, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve operatorgroup: %s/%s: %v", namespace, name, err)
	}
	return &operatorGroup, nil
}

// CreateSecret can return an ErrAlreadyExists
func (oe openshiftClient) CreateSecret(ctx context.Context, name string, content map[string]string, secretType corev1.SecretType, namespace string) (*corev1.Secret, error) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("creating secret", "namespace", namespace, "name", name)
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: content,
		Type:       secretType,
	}
	err := oe.Client.Create(ctx, &secret, &crclient.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return &secret, fmt.Errorf("could not create secret: %s/%s: %w: %v", namespace, name, ErrAlreadyExists, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not create secret: %s/%s: %v", namespace, name, err)
	}

	return &secret, nil
}

func (oe openshiftClient) DeleteSecret(ctx context.Context, name string, namespace string) error {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("deleting secret", "namespace", namespace, "name", name)
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := oe.Client.Delete(ctx, &secret, &crclient.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("could not delete secret: %s/%s: %v", namespace, name, err)
	}

	return nil
}

// GetSecret can return an ErrNotFound
func (oe openshiftClient) GetSecret(ctx context.Context, name string, namespace string) (*corev1.Secret, error) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("fetching secret", "namespace", namespace, "name", name)
	secret := corev1.Secret{}
	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, &secret)
	if apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("could not retrieve secret %s/%s: %w: %v", namespace, name, ErrNotFound, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve secret %s/%s: %v", namespace, name, err)
	}
	return &secret, nil
}

// CreateCatalogSource can return an ErrAlreadyExists
func (oe openshiftClient) CreateCatalogSource(ctx context.Context, data CatalogSourceData, namespace string) (*operatorsv1alpha1.CatalogSource, error) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("creating CatalogSource", "namespace", namespace, "name", data.Name)
	catalogSource := &operatorsv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name,
			Namespace: namespace,
		},
		Spec: operatorsv1alpha1.CatalogSourceSpec{
			SourceType:  operatorsv1alpha1.SourceTypeGrpc,
			Image:       data.Image,
			DisplayName: data.Name,
			Secrets:     data.Secrets,
		},
	}
	err := oe.Client.Create(ctx, catalogSource)
	if apierrors.IsAlreadyExists(err) {
		return catalogSource, fmt.Errorf("could not create catalogsource: %s/%s: %w: %v", namespace, data.Name, ErrAlreadyExists, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not create catalogsource: %s/%s: %v", namespace, data.Name, err)
	}
	return catalogSource, nil
}

func (oe *openshiftClient) DeleteCatalogSource(ctx context.Context, name string, namespace string) error {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("deleting CatalogSource", "namespace", namespace, "name", name)
	catalogSource := operatorsv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := oe.Client.Delete(ctx, &catalogSource)
	if err != nil {
		return fmt.Errorf("could not delete catalogsource: %s/%s: %v", namespace, name, err)
	}
	return nil
}

// GetCatalogSource cat return an ErrNotFound
func (oe *openshiftClient) GetCatalogSource(ctx context.Context, name string, namespace string) (*operatorsv1alpha1.CatalogSource, error) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("fetching catalogsource", "name", name)
	catalogSource := &operatorsv1alpha1.CatalogSource{}
	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, catalogSource)
	if apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("could not retrieve catalogsource: %s/%s: %w: %v", namespace, name, ErrNotFound, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve catalogsource: %s/%s: %v", namespace, name, err)
	}
	return catalogSource, nil
}

// CreateSubscription can return an ErrAlreadyExists
func (oe openshiftClient) CreateSubscription(ctx context.Context, data SubscriptionData, namespace string) (*operatorsv1alpha1.Subscription, error) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("creating Subscription", "namespace", namespace, "name", data.Name)
	subscription := &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name,
			Namespace: namespace,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			CatalogSource:          data.CatalogSource,
			CatalogSourceNamespace: data.CatalogSourceNamespace,
			Channel:                data.Channel,
			Package:                data.Package,
		},
	}
	err := oe.Client.Create(ctx, subscription)
	if apierrors.IsAlreadyExists(err) {
		return subscription, fmt.Errorf("could not create subscription: %s/%s: %w: %v", namespace, data.Name, ErrAlreadyExists, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not create subscription: %s/%s: %v", namespace, data.Name, err)
	}
	return subscription, nil
}

// GetSubscription can return an ErrNotFound
func (oe *openshiftClient) GetSubscription(ctx context.Context, name string, namespace string) (*operatorsv1alpha1.Subscription, error) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("fetching subscription", "namespace", namespace, "name", name)
	subscription := &operatorsv1alpha1.Subscription{}
	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, subscription)
	if apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("could not retrieve subscription: %s/%s: %w: %v", namespace, name, ErrNotFound, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve subscription: %s/%s: %v", namespace, name, err)
	}
	return subscription, nil
}

func (oe openshiftClient) DeleteSubscription(ctx context.Context, name string, namespace string) error {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("deleting Subscription", "namespace", namespace, "name", name)

	subscription := &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := oe.Client.Delete(ctx, subscription)
	if err != nil {
		return fmt.Errorf("could not delete subscription: %s/%s: %v", namespace, name, err)
	}
	return nil
}

// GetCSV can return an ErrNotFound
func (oe *openshiftClient) GetCSV(ctx context.Context, name string, namespace string) (*operatorsv1alpha1.ClusterServiceVersion, error) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.DBG).Info("fetching csv", "csvName", name, "namespace", namespace)
	csv := &operatorsv1alpha1.ClusterServiceVersion{}
	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, csv)
	if apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("could not retrieve csv: %s/%s: %w: %v", namespace, name, ErrNotFound, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve csv: %s/%s: %v", namespace, name, err)
	}
	return csv, nil
}

func (oe *openshiftClient) GetImages(ctx context.Context) (map[string]struct{}, error) {
	var pods corev1.PodList
	err := oe.Client.List(ctx, &pods, &crclient.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve pod list: %v", err)
	}

	imageList := make(map[string]struct{})
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			imageList[container.Image] = struct{}{}
		}
	}

	var imageStreamList imagestreamv1.ImageStreamList
	if err := oe.Client.List(ctx, &imageStreamList, &crclient.ListOptions{}); err != nil {
		return nil, fmt.Errorf("could not list image streams: %v", err)
	}
	for _, imageStream := range imageStreamList.Items {
		for _, tag := range imageStream.Spec.Tags {
			if tag.From.Kind == "DockerImage" {
				imageList[tag.From.Name] = struct{}{}
			}
		}
	}

	return imageList, nil
}

// CreateRoleBinding can return an ErrAlreadyExists
func (oe *openshiftClient) CreateRoleBinding(ctx context.Context, data RoleBindingData, namespace string) (*rbacv1.RoleBinding, error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.V(log.TRC).Info("creating RoleBinding", "name", data.Name, "namespace", namespace)
	subjectsObj := make([]rbacv1.Subject, 0, len(data.Subjects))
	for _, subject := range data.Subjects {
		subjectsObj = append(subjectsObj, rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      subject,
			Namespace: data.Namespace,
		})
	}
	roleObj := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		APIGroup: "rbac.authorization.k8s.io",
		Name:     data.Role,
	}
	roleBindingObj := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name,
			Namespace: namespace,
		},
		Subjects: subjectsObj,
		RoleRef:  roleObj,
	}
	err := oe.Client.Create(ctx, &roleBindingObj, &crclient.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return &roleBindingObj, fmt.Errorf("could not create rolebinding: %s/%s: %w: %v", namespace, data.Name, ErrAlreadyExists, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not create rolebinding: %s/%s: %v", namespace, data.Name, err)
	}

	logger.V(log.DBG).Info("created RoleBinding", "name", data.Name, "namespace", namespace)
	return &roleBindingObj, nil
}

// GetRoleBinding can return an ErrNotFound
func (oe *openshiftClient) GetRoleBinding(ctx context.Context, name string, namespace string) (*rbacv1.RoleBinding, error) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("fetching RoleBinding", "namespace", namespace, "name", name)
	roleBinding := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, &roleBinding)
	if apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("could not retrieve rolebinding: %s/%s: %w: %v", namespace, name, ErrNotFound, err)
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve rolebinding: %s/%s: %v", namespace, name, err)
	}
	return &roleBinding, nil
}

func (oe *openshiftClient) DeleteRoleBinding(ctx context.Context, name string, namespace string) error {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.TRC).Info("deleting RoleBinding", "namespace", namespace, "name", name)

	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := oe.Client.Delete(ctx, &roleBinding, &crclient.DeleteOptions{}); err != nil {
		return fmt.Errorf("could not delete rolebinding: %s/%s: %v", namespace, name, err)
	}
	return nil
}
