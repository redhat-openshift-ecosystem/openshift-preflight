package engine

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	imagestreamv1 "github.com/openshift/api/image/v1"
	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	client "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/client"
)

type openshiftEngine struct{}

func NewOpenshiftEngine() *cli.OpenshiftEngine {
	var engine cli.OpenshiftEngine = &openshiftEngine{}
	return &engine
}

func (oe *openshiftEngine) CreateNamespace(name string, opts cli.OpenshiftOptions) (*corev1.Namespace, error) {

	kubeconfig := ctrl.GetConfigOrDie()
	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}

	nsSpec := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: opts.Labels,
		},
	}

	resp, err := k8sClientset.CoreV1().
		Namespaces().
		Create(context.Background(), nsSpec, metav1.CreateOptions{})

	if err != nil {
		log.Error(fmt.Sprintf("error while creating Namespace %s: ", name), err)
		return nil, err
	}

	log.Debug("Namespace created: ", name)
	log.Trace("Received Namespace object from API server: ", resp)

	return resp, nil
}

func (oe *openshiftEngine) DeleteNamespace(name string, opts cli.OpenshiftOptions) error {
	kubeconfig := ctrl.GetConfigOrDie()
	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return err
	}
	log.Debug("Deleting namespace: " + name)
	return k8sClientset.CoreV1().
		Namespaces().
		Delete(context.Background(), name, metav1.DeleteOptions{})
}

func (oe *openshiftEngine) GetNamespace(name string) (*corev1.Namespace, error) {

	kubeconfig := ctrl.GetConfigOrDie()
	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}
	log.Debug("fetching namespace: " + name)
	return k8sClientset.CoreV1().
		Namespaces().
		Get(context.Background(), name, metav1.GetOptions{})
}

func (oe *openshiftEngine) CreateOperatorGroup(data cli.OperatorGroupData, opts cli.OpenshiftOptions) (*operatorv1.OperatorGroup, error) {

	ogClient, err := client.OperatorGroupClient(opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for OperatorGroup: ", err)
		return nil, err
	}

	log.Debug(fmt.Sprintf("Creating OperatorGroup %s in namespace %s", data.Name, opts.Namespace))
	resp, err := ogClient.Create(data, opts)

	if err != nil {
		log.Error(fmt.Sprintf("error while creating OperatorGroup %s: ", data.Name), err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("OperatorGroup %s is created successfully in namespace %s", data.Name, opts.Namespace))

	return resp, nil
}

func (oe *openshiftEngine) DeleteOperatorGroup(name string, opts cli.OpenshiftOptions) error {
	ogClient, err := client.OperatorGroupClient(opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for OperatorGroup: ", err)
		return err
	}
	log.Debug(fmt.Sprintf("Deleting OperatorGroup %s in namespace %s", name, opts.Namespace))

	err = ogClient.Delete(name, opts)
	if err != nil {
		log.Error(fmt.Sprintf("error while deleting OperatorGroup %s in namespace %s: ", name, opts.Namespace), err)
		return err
	}
	log.Debug(fmt.Sprintf("OperatorGroup %s is deleted successfully from namespace %s", name, opts.Namespace))

	return nil

}

func (oe *openshiftEngine) GetOperatorGroup(name string, opts cli.OpenshiftOptions) (*operatorv1.OperatorGroup, error) {
	ogClient, err := client.OperatorGroupClient(opts.Namespace)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("fetching operatorgroup %s from namespace %s", name, opts.Namespace))
	return ogClient.Get(name, opts.Namespace)
}

func (oe openshiftEngine) CreateSecret(name string, content map[string]string, secretType corev1.SecretType, opts cli.OpenshiftOptions) (*corev1.Secret, error) {
	kubeconfig := ctrl.GetConfigOrDie()
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
			Namespace: opts.Namespace,
		},
		StringData: content,
		Type:       secretType,
	}
	resp, err := k8sClientset.CoreV1().
		Secrets(opts.Namespace).
		Create(context.Background(), &secret, metav1.CreateOptions{})

	if err != nil {
		log.Error(fmt.Sprintf("error while creating secret %s in namespace %s", name, opts.Namespace), err)
		return nil, err
	}

	log.Debug("Secret created: ", name)
	log.Trace("Received Secret object from API server: ", resp)

	return resp, nil
}

func (oe openshiftEngine) DeleteSecret(name string, opts cli.OpenshiftOptions) error {
	kubeconfig := ctrl.GetConfigOrDie()
	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return err
	}
	log.Debug(fmt.Sprintf("Deleting secret %s from namespace %s", name, opts.Namespace))
	return k8sClientset.CoreV1().
		Secrets(opts.Namespace).
		Delete(context.Background(), name, metav1.DeleteOptions{})
}

func (oe openshiftEngine) GetSecret(name string, opts cli.OpenshiftOptions) (*corev1.Secret, error) {
	kubeconfig := ctrl.GetConfigOrDie()
	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("fetching secrets %s from namespace %s", name, opts.Namespace))
	return k8sClientset.CoreV1().
		Secrets(opts.Namespace).
		Get(context.Background(), name, metav1.GetOptions{})
}

func (oe openshiftEngine) CreateCatalogSource(data cli.CatalogSourceData, opts cli.OpenshiftOptions) (*operatorv1alpha1.CatalogSource, error) {

	csClient, err := client.CatalogSourceClient(opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return nil, err
	}

	log.Debug(fmt.Sprintf("Creating CatalogSource %s in namespace %s", data.Name, opts.Namespace))
	resp, err := csClient.Create(data, opts)

	if err != nil {
		log.Error(fmt.Sprintf("error while creating CatalogSource %s: ", data.Name), err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("CatalogSource %s is created successfully in namespace %s", data.Name, opts.Namespace))

	return resp, nil
}

func (oe *openshiftEngine) DeleteCatalogSource(name string, opts cli.OpenshiftOptions) error {
	csClient, err := client.CatalogSourceClient(opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return err
	}
	log.Debug(fmt.Sprintf("Deleting CatalogSource %s in namespace %s", name, opts.Namespace))

	err = csClient.Delete(name, opts)
	if err != nil {
		log.Error(fmt.Sprintf("error while deleting CatalogSource %s in namespace %s: ", name, opts.Namespace), err)
		return err
	}
	log.Debug(fmt.Sprintf("CatalogSource %s is deleted successfully from namespace %s", name, opts.Namespace))

	return nil
}

func (oe *openshiftEngine) GetCatalogSource(name string, opts cli.OpenshiftOptions) (*operatorv1alpha1.CatalogSource, error) {

	csClient, err := client.CatalogSourceClient(opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return nil, err
	}
	log.Debug("fetching catalogsource: " + name)
	return csClient.Get(name, opts.Namespace)
}

func (oe openshiftEngine) CreateSubscription(data cli.SubscriptionData, opts cli.OpenshiftOptions) (*operatorv1alpha1.Subscription, error) {

	subsClient, err := client.SubscriptionClient(opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for Subscription: ", err)
		return nil, err
	}

	log.Debug(fmt.Sprintf("Creating Subscription %s in namespace %s", data.Name, opts.Namespace))
	resp, err := subsClient.Create(data, opts)

	if err != nil {
		log.Error(fmt.Sprintf("error while creating Subscription %s: ", data.Name), err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("Subscription %s is created successfully in namespace %s", data.Name, opts.Namespace))

	return resp, nil
}

func (oe *openshiftEngine) GetSubscription(name string, opts cli.OpenshiftOptions) (*operatorv1alpha1.Subscription, error) {
	subsClient, err := client.SubscriptionClient(opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for Subscription: ", err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("fetching subscription %s from namespace %s ", name, opts.Namespace))
	return subsClient.Get(name, opts.Namespace)
}

func (oe openshiftEngine) DeleteSubscription(name string, opts cli.OpenshiftOptions) error {

	subsClient, err := client.SubscriptionClient(opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for Subscription: ", err)
		return err
	}
	log.Debug(fmt.Sprintf("Deleting Subscription %s in namespace %s", name, opts.Namespace))

	err = subsClient.Delete(name, opts)
	if err != nil {
		log.Error(fmt.Sprintf("error while deleting Subscription %s in namespace %s: ", name, opts.Namespace), err)
		return err
	}
	log.Debug(fmt.Sprintf("Subscription %s is deleted successfully from namespace %s", name, opts.Namespace))

	return nil
}

func (oe *openshiftEngine) GetCSV(name string, opts cli.OpenshiftOptions) (*operatorv1alpha1.ClusterServiceVersion, error) {

	csvClient, err := client.CsvClient(opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for csv: ", err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("fetching csv %s from namespace %s ", name, opts.Namespace))
	return csvClient.Get(name, opts.Namespace)
}

func (oe *openshiftEngine) GetImages() (map[string]struct{}, error) {
	kubeconfig := ctrl.GetConfigOrDie()
	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}
	pods, err := k8sClientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
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
	if err := isClient.List(context.Background(), &imageStreamList, &crclient.ListOptions{}); err != nil {
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
