package engine

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	client "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/client"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

type OpenshiftEngine struct {
	KubeConfig *rest.Config
}

func (pe OpenshiftEngine) CreateNamespace(name string, opts cli.OpenshiftOptions) (*cli.OpenshiftReport, error) {

	k8sClientset, err := kubernetes.NewForConfig(pe.KubeConfig)

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
		log.Error(fmt.Sprintf("error while creating Namespace %s:", name), err)
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}

	log.Debug("Namespace created: ", name)
	log.Trace("Received Namespace object from API server: ", resp)

	return &cli.OpenshiftReport{
		Stdout: nsSpec.String(),
		Stderr: "",
	}, nil
}

func (pe OpenshiftEngine) DeleteNamespace(name string, opts cli.OpenshiftOptions) error {
	k8sClientset, err := kubernetes.NewForConfig(pe.KubeConfig)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return err
	}
	log.Debug("Deleting namespace: " + name)
	return k8sClientset.CoreV1().
		Namespaces().
		Delete(context.Background(), name, metav1.DeleteOptions{})
}

func (pe OpenshiftEngine) GetNamespace(name string) (*corev1.Namespace, error) {
	k8sClientset, err := kubernetes.NewForConfig(pe.KubeConfig)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}
	log.Debug("fetching namespace: " + name)
	return k8sClientset.CoreV1().
		Namespaces().
		Get(context.Background(), name, metav1.GetOptions{})
}

func (pe OpenshiftEngine) CreateOperatorGroup(data cli.OperatorGroupData, opts cli.OpenshiftOptions) (*cli.OpenshiftReport, error) {

	crdClient, err := client.OperatorGroupClient(pe.KubeConfig, opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for OperatorGroup: ", err)
		return nil, err
	}

	log.Debug(fmt.Sprintf("Creating OperatorGroup %s in namespace %s", data.Name, opts.Namespace))
	resp, err := crdClient.Create(data, opts)

	if err != nil {
		log.Error(fmt.Sprintf("error while creating OperatorGroup: %s", data.Name), err)
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}
	log.Debug(fmt.Sprintf("OperatorGroup %s is created successfully in namespace %s", data.Name, opts.Namespace))

	return &cli.OpenshiftReport{
		Stdout: fmt.Sprintf("%#v", resp),
		Stderr: "",
	}, nil
}

func (pe OpenshiftEngine) DeleteOperatorGroup(name string, opts cli.OpenshiftOptions) (*cli.OpenshiftReport, error) {
	crdClient, err := client.OperatorGroupClient(pe.KubeConfig, opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for OperatorGroup: ", err)
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}
	log.Debug(fmt.Sprintf("Deleting OperatorGroup %s in namespace %s", name, opts.Namespace))

	err = crdClient.Delete(name, opts)
	if err != nil {
		log.Error(fmt.Sprintf("error while deleting OperatorGroup %s in namespace %s: ", name, opts.Namespace), err)
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}
	log.Debug(fmt.Sprintf("OperatorGroup %s is deleted successfully from namespace %s", name, opts.Namespace))

	cs, err := pe.GetOperatorGroup(name, opts)
	if err != nil {
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}

	return &cli.OpenshiftReport{
		Stdout: fmt.Sprintf("%#v", cs),
		Stderr: "",
	}, nil

}

func (pe OpenshiftEngine) GetOperatorGroup(name string, opts cli.OpenshiftOptions) (*operatorv1.OperatorGroup, error) {
	crdClient, err := client.OperatorGroupClient(pe.KubeConfig, opts.Namespace)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("fetching operatorgroup %s from namespace %s ", name, opts.Namespace))
	return crdClient.Get(name, opts.Namespace)
}

func (pe OpenshiftEngine) CreateCatalogSource(data cli.CatalogSourceData, opts cli.OpenshiftOptions) (*cli.OpenshiftReport, error) {

	crdClient, err := client.CatalogSourceClient(pe.KubeConfig, opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return nil, err
	}

	log.Debug(fmt.Sprintf("Creating CatalogSource %s in namespace %s", data.Name, opts.Namespace))
	resp, err := crdClient.Create(data, opts)

	if err != nil {
		log.Error(fmt.Sprintf("error while creating CatalogSource %s: ", data.Name), err)
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}
	log.Debug(fmt.Sprintf("CatalogSource %s is created successfully in namespace %s", data.Name, opts.Namespace))

	return &cli.OpenshiftReport{
		Stdout: fmt.Sprintf("%#v", resp),
		Stderr: "",
	}, nil
}

func (pe OpenshiftEngine) DeleteCatalogSource(name string, opts cli.OpenshiftOptions) (*cli.OpenshiftReport, error) {
	crdClient, err := client.CatalogSourceClient(pe.KubeConfig, opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}
	log.Debug(fmt.Sprintf("Deleting CatalogSource %s in namespace %s", name, opts.Namespace))

	err = crdClient.Delete(name, opts)
	if err != nil {
		log.Error(fmt.Sprintf("error while deleting CatalogSource %s in namespace %s: ", name, opts.Namespace), err)
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}
	log.Debug(fmt.Sprintf("CatalogSource %s is deleted successfully from namespace %s", name, opts.Namespace))

	cs, err := pe.GetCatalogSource(name, opts)
	if err != nil {
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}

	return &cli.OpenshiftReport{
		Stdout: fmt.Sprintf("%#v", cs),
		Stderr: "",
	}, nil
}

func (pe OpenshiftEngine) GetCatalogSource(name string, opts cli.OpenshiftOptions) (*operatorv1alpha1.CatalogSource, error) {

	crdClient, err := client.CatalogSourceClient(pe.KubeConfig, opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return nil, err
	}
	log.Debug("fetching catalogsource: " + name)
	return crdClient.Get(name, opts.Namespace)
}

func (pe OpenshiftEngine) CreateSubscription(data cli.SubscriptionData, opts cli.OpenshiftOptions) (*cli.OpenshiftReport, error) {

	crdClient, err := client.SubscriptionClient(pe.KubeConfig, opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for Subscription: ", err)
		return nil, err
	}

	log.Debug(fmt.Sprintf("Creating Subscription %s in namespace %s", data.Name, opts.Namespace))
	resp, err := crdClient.Create(data, opts)

	if err != nil {
		log.Error(fmt.Sprintf("error while creating Subscription %s: ", data.Name), err)
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}
	log.Debug(fmt.Sprintf("Subscription %s is created successfully in namespace %s", data.Name, opts.Namespace))

	return &cli.OpenshiftReport{
		Stdout: fmt.Sprintf("%#v", resp),
		Stderr: "",
	}, nil
}

func (pe OpenshiftEngine) GetSubscription(name string, opts cli.OpenshiftOptions) (*operatorv1alpha1.Subscription, error) {
	crdClient, err := client.SubscriptionClient(pe.KubeConfig, opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for Subscription: ", err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("fetching subscription %s from namespace %s ", name, opts.Namespace))
	return crdClient.Get(name, opts.Namespace)
}

func (pe OpenshiftEngine) DeleteSubscription(name string, opts cli.OpenshiftOptions) (*cli.OpenshiftReport, error) {

	crdClient, err := client.SubscriptionClient(pe.KubeConfig, opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for Subscription: ", err)
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}
	log.Debug(fmt.Sprintf("Deleting Subscription %s in namespace %s", name, opts.Namespace))

	err = crdClient.Delete(name, opts)
	if err != nil {
		log.Error(fmt.Sprintf("error while deleting Subscription %s in namespace %s: ", name, opts.Namespace), err)
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}
	log.Debug(fmt.Sprintf("Subscription %s is deleted successfully from namespace %s", name, opts.Namespace))

	cs, err := pe.GetSubscription(name, opts)
	if err != nil {
		return &cli.OpenshiftReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}

	return &cli.OpenshiftReport{
		Stdout: fmt.Sprintf("%#v", cs),
		Stderr: "",
	}, nil
}

func (pe OpenshiftEngine) GetCSV(name string, opts cli.OpenshiftOptions) (*operatorv1alpha1.ClusterServiceVersion, error) {

	crdClient, err := client.CsvClient(pe.KubeConfig, opts.Namespace)
	if err != nil {
		log.Error("unable to create a client for csv: ", err)
		return nil, err
	}
	log.Debug(fmt.Sprintf("fetching csv %s from namespace %s ", name, opts.Namespace))
	return crdClient.Get(name, opts.Namespace)
}
