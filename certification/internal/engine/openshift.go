package engine

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

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
	log.Debug(fmt.Sprintf("fetching operatorgroup %s from namespace %s ", name, opts.Namespace))
	return ogClient.Get(name, opts.Namespace)
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
