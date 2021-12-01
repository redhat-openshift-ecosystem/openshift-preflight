package engine

import (
	"context"
	"fmt"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"

	log "github.com/sirupsen/logrus"

	imagestreamv1 "github.com/openshift/api/image/v1"
	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	configv1ClientSet "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	client "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/client"
	preflightRuntime "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
)

type openshiftEngine struct{}

func NewOpenshiftEngine() *cli.OpenshiftEngine {
	var engine cli.OpenshiftEngine = &openshiftEngine{}
	return &engine
}

func (oe *openshiftEngine) CreateNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("could not get kubeconfig")
		return nil, err
	}
	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}

	nsSpec := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	ns, err := k8sClientset.CoreV1().
		Namespaces().
		Create(ctx, nsSpec, metav1.CreateOptions{})

	if err != nil {
		log.Error(fmt.Sprintf("error while creating Namespace %s: ", name), err)
		return nil, err
	}
	log.Debug("Namespace created: ", name)
	return ns, nil
}

func (oe *openshiftEngine) DeleteNamespace(ctx context.Context, name string) error {
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("could not get kubeconfig")
		return err
	}
	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return err
	}
	log.Debug("Deleting namespace: " + name)

	return k8sClientset.CoreV1().
		Namespaces().
		Delete(ctx, name, metav1.DeleteOptions{})
}

func (oe *openshiftEngine) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("could not get kubeconfig")
		return nil, err
	}
	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}
	log.Debug("fetching namespace: " + name)

	return k8sClientset.CoreV1().
		Namespaces().
		Get(ctx, name, metav1.GetOptions{})
}

func (oe *openshiftEngine) CreateOperatorGroup(ctx context.Context, data cli.OperatorGroupData, namespace string) (*operatorv1.OperatorGroup, error) {

	ogClient, err := client.OperatorGroupClient(namespace)
	if err != nil {
		log.Error("unable to create a client for OperatorGroup: ", err)
		return nil, err
	}

	log.Debug(fmt.Sprintf("Creating OperatorGroup %s in namespace %s", data.Name, namespace))
	resp, err := ogClient.Create(ctx, data)

	if err != nil {
		log.Error(fmt.Sprintf("error while creating OperatorGroup %s: ", data.Name), err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("OperatorGroup %s is created successfully in namespace %s", data.Name, namespace))

	return resp, nil
}

func (oe *openshiftEngine) DeleteOperatorGroup(ctx context.Context, name string, namespace string) error {
	ogClient, err := client.OperatorGroupClient(namespace)
	if err != nil {
		log.Error("unable to create a client for OperatorGroup: ", err)
		return err
	}
	log.Debug(fmt.Sprintf("Deleting OperatorGroup %s in namespace %s", name, namespace))

	err = ogClient.Delete(ctx, name)
	if err != nil {
		log.Error(fmt.Sprintf("error while deleting OperatorGroup %s in namespace %s: ", name, namespace), err)
		return err
	}
	log.Debug(fmt.Sprintf("OperatorGroup %s is deleted successfully from namespace %s", name, namespace))

	return nil
}

func (oe *openshiftEngine) GetOperatorGroup(ctx context.Context, name string, namespace string) (*operatorv1.OperatorGroup, error) {
	ogClient, err := client.OperatorGroupClient(namespace)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("fetching operatorgroup %s from namespace %s", name, namespace))
	return ogClient.Get(ctx, name)
}

func (oe openshiftEngine) CreateSecret(ctx context.Context, name string, content map[string]string, secretType corev1.SecretType, namespace string) (*corev1.Secret, error) {
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("could not get kubeconfig")
		return nil, err
	}

	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}

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
	resp, err := k8sClientset.CoreV1().
		Secrets(namespace).
		Create(ctx, &secret, metav1.CreateOptions{})
	if err != nil {
		log.Error(fmt.Sprintf("error while creating secret %s in namespace %s: ", name, namespace), err)
		return nil, err
	}

	log.Debug(fmt.Sprintf("Secret %s created successfully in namespace %s", name, namespace))
	return resp, nil
}

func (oe openshiftEngine) DeleteSecret(ctx context.Context, name string, namespace string) error {
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("could not get kubeconfig")
		return err
	}
	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return err
	}
	log.Debug(fmt.Sprintf("Deleting secret %s from namespace %s", name, namespace))
	return k8sClientset.CoreV1().
		Secrets(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
}

func (oe openshiftEngine) GetSecret(ctx context.Context, name string, namespace string) (*corev1.Secret, error) {
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("could not get kubeconfig")
		return nil, err
	}
	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("fetching secrets %s from namespace %s", name, namespace))
	return k8sClientset.CoreV1().
		Secrets(namespace).
		Get(ctx, name, metav1.GetOptions{})
}

func (oe openshiftEngine) CreateCatalogSource(ctx context.Context, data cli.CatalogSourceData, namespace string) (*operatorv1alpha1.CatalogSource, error) {
	csClient, err := client.CatalogSourceClient(namespace)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("Creating CatalogSource %s in namespace %s", data.Name, namespace))
	resp, err := csClient.Create(ctx, data)
	if err != nil {
		log.Error(fmt.Sprintf("error while creating CatalogSource %s: ", data.Name), err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("CatalogSource %s is created successfully in namespace %s", data.Name, namespace))
	return resp, nil
}

func (oe *openshiftEngine) DeleteCatalogSource(ctx context.Context, name string, namespace string) error {
	csClient, err := client.CatalogSourceClient(namespace)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return err
	}
	log.Debug(fmt.Sprintf("Deleting CatalogSource %s in namespace %s", name, namespace))
	err = csClient.Delete(ctx, name)
	if err != nil {
		log.Error(fmt.Sprintf("error while deleting CatalogSource %s in namespace %s: ", name, namespace), err)
		return err
	}
	log.Debug(fmt.Sprintf("CatalogSource %s is deleted successfully from namespace %s", name, namespace))
	return nil
}

func (oe *openshiftEngine) GetCatalogSource(ctx context.Context, name string, namespace string) (*operatorv1alpha1.CatalogSource, error) {
	csClient, err := client.CatalogSourceClient(namespace)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return nil, err
	}
	log.Debug("fetching catalogsource: " + name)
	return csClient.Get(ctx, name)
}

func (oe openshiftEngine) CreateSubscription(ctx context.Context, data cli.SubscriptionData, namespace string) (*operatorv1alpha1.Subscription, error) {

	subsClient, err := client.SubscriptionClient(namespace)
	if err != nil {
		log.Error("unable to create a client for Subscription: ", err)
		return nil, err
	}

	log.Debug(fmt.Sprintf("Creating Subscription %s in namespace %s", data.Name, namespace))
	resp, err := subsClient.Create(ctx, data)

	if err != nil {
		log.Error(fmt.Sprintf("error while creating Subscription %s: ", data.Name), err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("Subscription %s is created successfully in namespace %s", data.Name, namespace))

	return resp, nil
}

func (oe *openshiftEngine) GetSubscription(ctx context.Context, name string, namespace string) (*operatorv1alpha1.Subscription, error) {
	subsClient, err := client.SubscriptionClient(namespace)
	if err != nil {
		log.Error("unable to create a client for Subscription: ", err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("fetching subscription %s from namespace %s ", name, namespace))
	return subsClient.Get(ctx, name)
}

func (oe openshiftEngine) DeleteSubscription(ctx context.Context, name string, namespace string) error {
	subsClient, err := client.SubscriptionClient(namespace)
	if err != nil {
		log.Error("unable to create a client for Subscription: ", err)
		return err
	}
	log.Debug(fmt.Sprintf("Deleting Subscription %s in namespace %s", name, namespace))

	err = subsClient.Delete(ctx, name)
	if err != nil {
		log.Error(fmt.Sprintf("error while deleting Subscription %s in namespace %s: ", name, namespace), err)
		return err
	}
	log.Debug(fmt.Sprintf("Subscription %s is deleted successfully from namespace %s", name, namespace))
	return nil
}

func (oe *openshiftEngine) GetCSV(ctx context.Context, name string, namespace string) (*operatorv1alpha1.ClusterServiceVersion, error) {
	csvClient, err := client.CsvClient(namespace)
	if err != nil {
		log.Error("unable to create a client for csv: ", err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("fetching csv %s from namespace %s", name, namespace))
	return csvClient.Get(ctx, name)
}

func (oe *openshiftEngine) GetImages(ctx context.Context) (map[string]struct{}, error) {
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("could not get kubeconfig")
		return nil, err
	}
	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}
	pods, err := k8sClientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
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

	scheme := runtime.NewScheme()
	imagestreamv1.AddToScheme(scheme)
	isClient, err := crclient.New(kubeconfig, crclient.Options{Scheme: scheme})
	if err != nil {
		log.Error("could not create isClient: ", err)
		return nil, err
	}
	var imageStreamList imagestreamv1.ImageStreamList
	if err := isClient.List(ctx, &imageStreamList, &crclient.ListOptions{}); err != nil {
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
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("unable to create a rest client: ", err)
		return nil, err
	}

	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}

	log.Debug(fmt.Sprintf("Creating RoleBinding %s in namespace %s", data.Name, namespace))
	subjectsObj := make([]rbacv1.Subject, 1)
	for i := range data.Subjects {
		subjectsObj[i] = rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      data.Subjects[i],
			Namespace: data.Namespace,
		}
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
	resp, err := k8sClientset.RbacV1().
		RoleBindings(namespace).
		Create(ctx, &roleBindingObj, metav1.CreateOptions{})
	if err != nil {
		log.Error(fmt.Sprintf("error while creating rolebinding %s in namespace %s: ", data.Name, namespace), err)
		return nil, err
	}

	log.Debug(fmt.Sprintf("RoleBinding %s created in namespace %s", data.Name, namespace))
	return resp, nil
}

func (oe *openshiftEngine) GetRoleBinding(ctx context.Context, name string, namespace string) (*rbacv1.RoleBinding, error) {
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("unable to create a rest client: ", err)
		return nil, err
	}

	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}

	log.Debug(fmt.Sprintf("fetching RoleBinding %s from namespace %s: ", name, namespace))
	return k8sClientset.RbacV1().
		RoleBindings(namespace).
		Get(ctx, name, metav1.GetOptions{})
}

func (oe *openshiftEngine) DeleteRoleBinding(ctx context.Context, name string, namespace string) error {
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("unable to create a rest client: ", err)
		return err
	}

	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return err
	}

	log.Debug(fmt.Sprintf("Deleting RoleBinding %s in namespace %s", name, namespace))

	err = k8sClientset.RbacV1().
		RoleBindings(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})

	if err != nil {
		log.Error(fmt.Sprintf("error while deleting RoleBiding %s in namespace %s: ", name, namespace), err)
		return err
	}
	log.Debug(fmt.Sprintf("RoleBinding %s is deleted successfully from namespace %s", name, namespace))
	return nil
}

func GetOpenshiftClusterVersion() (preflightRuntime.OpenshiftClusterVersion, error) {

	if _, ok := os.LookupEnv("KUBECONFIG"); !ok {
		return preflightRuntime.UnknownOpenshiftClusterVersion(), errors.ErrNoKubeconfig
	}
	kubeConfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("unable to load the config, check if KUBECONFIG is set correctly: ", err)
		return preflightRuntime.UnknownOpenshiftClusterVersion(), err
	}
	configV1Client, err := configv1ClientSet.NewForConfig(kubeConfig)
	if err != nil {
		log.Error("unable to create a client with the provided kubeconfig: ", err)
		return preflightRuntime.UnknownOpenshiftClusterVersion(), err
	}
	openshiftApiServer, err := configV1Client.ClusterOperators().Get(context.Background(), "openshift-apiserver", metav1.GetOptions{})
	if err != nil {
		log.Error("unable to get openshift-apiserver cluster operator: ", err)
		return preflightRuntime.UnknownOpenshiftClusterVersion(), err
	}

	log.Debug(fmt.Sprintf("fetching operator version and openshift-apiserver version %s from %s", openshiftApiServer.Status.Versions, kubeConfig.Host))
	return preflightRuntime.OpenshiftClusterVersion{
		Name:    "OpenShift",
		Version: openshiftApiServer.Status.Versions[1].Version,
	}, nil
}
