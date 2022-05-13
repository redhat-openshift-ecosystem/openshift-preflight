package engine

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	imagestreamv1 "github.com/openshift/api/image/v1"
	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
)

type openshiftEngine struct {
	Client crclient.Client
}

func NewOpenshiftEngine(client crclient.Client) *cli.OpenshiftEngine {
	var engine cli.OpenshiftEngine = &openshiftEngine{
		Client: client,
	}
	return &engine
}

func (oe *openshiftEngine) CreateNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	nsSpec := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	err := oe.Client.Create(ctx, &nsSpec, &crclient.CreateOptions{})
	if err != nil {
		log.Error(fmt.Errorf("%w: error while creating Namespace: %s", err, name))
		return nil, err
	}
	log.Debug("Namespace created: ", name)
	return &nsSpec, nil
}

func (oe *openshiftEngine) DeleteNamespace(ctx context.Context, name string) error {
	log.Debugf("Deleting namespace: %s", name)
	nsSpec := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return oe.Client.Delete(ctx, &nsSpec, &crclient.DeleteOptions{})
}

func (oe *openshiftEngine) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	log.Debugf("fetching namespace %s", name)
	nsSpec := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name: name,
	}, &nsSpec)
	if err != nil {
		log.Error(fmt.Errorf("%w: could not retrieve namespace: %s", err, name))
		return nil, err
	}
	return &nsSpec, nil
}

func (oe *openshiftEngine) CreateOperatorGroup(ctx context.Context, data cli.OperatorGroupData, namespace string) (*operatorv1.OperatorGroup, error) {
	log.Debugf("Creating OperatorGroup %s in namespace %s", data.Name, namespace)
	operatorGroup := &operatorv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name,
			Namespace: namespace,
		},
		Spec: operatorv1.OperatorGroupSpec{
			TargetNamespaces: data.TargetNamespaces,
		},
	}
	err := oe.Client.Create(ctx, operatorGroup)
	if err != nil {
		log.Error(fmt.Errorf("%w: error while creating OperatorGroup: %s", err, data.Name))
		return nil, err
	}

	log.Debugf("OperatorGroup %s is created successfully in namespace %s", data.Name, namespace)
	return operatorGroup, nil
}

func (oe *openshiftEngine) DeleteOperatorGroup(ctx context.Context, name string, namespace string) error {
	log.Debugf("Deleting OperatorGroup %s in namespace %s", name, namespace)
	operatorGroup := operatorv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := oe.Client.Delete(ctx, &operatorGroup)
	if err != nil {
		log.Error(fmt.Errorf("%w: error while deleting OperatorGroup: %s in namespace: %s", err, name, namespace))
		return err
	}

	log.Debugf("OperatorGroup %s is deleted successfully from namespace %s", name, namespace)
	return nil
}

func (oe *openshiftEngine) GetOperatorGroup(ctx context.Context, name string, namespace string) (*operatorv1.OperatorGroup, error) {
	log.Debugf("fetching operatorgroup %s from namespace %s", name, namespace)
	operatorGroup := operatorv1.OperatorGroup{}
	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, &operatorGroup)
	if err != nil {
		log.Error(fmt.Errorf("%w: error while retrieving OperatorGroup %s in namespace %s", err, name, namespace))
		return nil, err
	}
	return &operatorGroup, nil
}

func (oe openshiftEngine) CreateSecret(ctx context.Context, name string, content map[string]string, secretType corev1.SecretType, namespace string) (*corev1.Secret, error) {
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
	if err != nil {
		log.Error(fmt.Errorf("%w: error while creating secret: %s in namespace: %s", err, name, namespace))
		return nil, err
	}

	log.Debugf("Secret %s created successfully in namespace %s", name, namespace)
	return &secret, nil
}

func (oe openshiftEngine) DeleteSecret(ctx context.Context, name string, namespace string) error {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	log.Debugf("Deleting secret %s from namespace %s", name, namespace)
	return oe.Client.Delete(ctx, &secret, &crclient.DeleteOptions{})
}

func (oe openshiftEngine) GetSecret(ctx context.Context, name string, namespace string) (*corev1.Secret, error) {
	log.Debugf("fetching secrets %s from namespace %s", name, namespace)
	secret := corev1.Secret{}
	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, &secret)
	if err != nil {
		log.Error(fmt.Errorf("%w: error while retrieving secret: %s in namespace: %s", err, name, namespace))
		return nil, err
	}
	return &secret, nil
}

func (oe openshiftEngine) CreateCatalogSource(ctx context.Context, data cli.CatalogSourceData, namespace string) (*operatorv1alpha1.CatalogSource, error) {
	log.Debugf("Creating CatalogSource %s in namespace %s", data.Name, namespace)
	catalogSource := &operatorv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name,
			Namespace: namespace,
		},
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType:  operatorv1alpha1.SourceTypeGrpc,
			Image:       data.Image,
			DisplayName: data.Name,
			Secrets:     data.Secrets,
		},
	}
	err := oe.Client.Create(ctx, catalogSource)
	if err != nil {
		log.Error(fmt.Errorf("%w: error while creating CatalogSource: %s", err, data.Name))
		return nil, err
	}
	log.Debugf("CatalogSource %s is created successfully in namespace %s", data.Name, namespace)
	return catalogSource, nil
}

func (oe *openshiftEngine) DeleteCatalogSource(ctx context.Context, name string, namespace string) error {
	log.Debugf("Deleting CatalogSource %s in namespace %s", name, namespace)
	catalogSource := operatorv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := oe.Client.Delete(ctx, &catalogSource)
	if err != nil {
		log.Error(fmt.Errorf("%w: error while deleting CatalogSource: %s in namespace: %s", err, name, namespace))
		return err
	}
	log.Debugf("CatalogSource %s is deleted successfully from namespace %s", name, namespace)
	return nil
}

func (oe *openshiftEngine) GetCatalogSource(ctx context.Context, name string, namespace string) (*operatorv1alpha1.CatalogSource, error) {
	log.Debug("fetching catalogsource: " + name)
	catalogSource := &operatorv1alpha1.CatalogSource{}
	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, catalogSource)
	if err != nil {
		log.Error(fmt.Errorf("%w: error while retieving CatalogSource: %s in namespace: %s", err, name, namespace))
		return nil, err
	}
	return catalogSource, nil
}

func (oe openshiftEngine) CreateSubscription(ctx context.Context, data cli.SubscriptionData, namespace string) (*operatorv1alpha1.Subscription, error) {
	log.Debugf("Creating Subscription %s in namespace %s", data.Name, namespace)
	subscription := &operatorv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name,
			Namespace: namespace,
		},
		Spec: &operatorv1alpha1.SubscriptionSpec{
			CatalogSource:          data.CatalogSource,
			CatalogSourceNamespace: data.CatalogSourceNamespace,
			Channel:                data.Channel,
			Package:                data.Package,
		},
	}
	err := oe.Client.Create(ctx, subscription)
	if err != nil {
		log.Error(fmt.Errorf("%w: error while creating Subscription: %s", err, data.Name))
		return nil, err
	}
	log.Debugf("Subscription %s is created successfully in namespace %s", data.Name, namespace)

	return subscription, nil
}

func (oe *openshiftEngine) GetSubscription(ctx context.Context, name string, namespace string) (*operatorv1alpha1.Subscription, error) {
	log.Debugf("fetching subscription %s from namespace %s ", name, namespace)
	subscription := &operatorv1alpha1.Subscription{}
	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, subscription)
	if err != nil {
		log.Error(fmt.Errorf("%w: error while retrieving Subscription: %s in namespace: %s", err, name, namespace))
		return nil, err
	}
	return subscription, nil
}

func (oe openshiftEngine) DeleteSubscription(ctx context.Context, name string, namespace string) error {
	log.Debugf("Deleting Subscription %s in namespace %s", name, namespace)

	subscription := &operatorv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := oe.Client.Delete(ctx, subscription)
	if err != nil {
		log.Error(fmt.Errorf("%w: error while deleting Subscription: %s in namespace: %s", err, name, namespace))
		return err
	}
	log.Debugf("Subscription %s is deleted successfully from namespace %s", name, namespace)
	return nil
}

func (oe *openshiftEngine) GetCSV(ctx context.Context, name string, namespace string) (*operatorv1alpha1.ClusterServiceVersion, error) {
	log.Debugf("fetching csv %s from namespace %s", name, namespace)
	csv := &operatorv1alpha1.ClusterServiceVersion{}
	err := oe.Client.Get(ctx, crclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, csv)

	return csv, err
}

func (oe *openshiftEngine) GetImages(ctx context.Context) (map[string]struct{}, error) {
	var pods corev1.PodList
	err := oe.Client.List(ctx, &pods, &crclient.ListOptions{})
	if err != nil {
		log.Error("could not retrieve pod list: ", err)
		return nil, err
	}

	imageList := make(map[string]struct{})
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			imageList[container.Image] = struct{}{}
		}
	}

	var imageStreamList imagestreamv1.ImageStreamList
	if err := oe.Client.List(ctx, &imageStreamList, &crclient.ListOptions{}); err != nil {
		log.Error("could not list image stream: ", err)
		return nil, err
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

func (oe *openshiftEngine) CreateRoleBinding(ctx context.Context, data cli.RoleBindingData, namespace string) (*rbacv1.RoleBinding, error) {
	log.Debugf("Creating RoleBinding %s in namespace %s", data.Name, namespace)
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
	if err != nil {
		log.Error(fmt.Errorf("%w: error while creating rolebinding: %s in namespace: %s", err, data.Name, namespace))
		return nil, err
	}

	log.Debugf("RoleBinding %s created in namespace %s", data.Name, namespace)
	return &roleBindingObj, nil
}

func (oe *openshiftEngine) GetRoleBinding(ctx context.Context, name string, namespace string) (*rbacv1.RoleBinding, error) {
	log.Debugf("fetching RoleBinding %s from namespace %s: ", name, namespace)
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
	if err != nil {
		log.Error(fmt.Errorf("%w: error while retrieving rolebinding: %s in namespace: %s", err, name, namespace))
		return nil, err
	}
	return &roleBinding, nil
}

func (oe *openshiftEngine) DeleteRoleBinding(ctx context.Context, name string, namespace string) error {
	log.Debugf("Deleting RoleBinding %s in namespace %s", name, namespace)

	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := oe.Client.Delete(ctx, &roleBinding, &crclient.DeleteOptions{}); err != nil {
		log.Error(fmt.Errorf("%w: error while deleting RoleBiding: %s in namespace: %s", err, name, namespace))
		return err
	}
	log.Debugf("RoleBinding %s is deleted successfully from namespace %s", name, namespace)
	return nil
}
