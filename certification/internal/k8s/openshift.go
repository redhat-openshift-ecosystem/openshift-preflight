package k8s

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

type OpenshiftCLIEngine struct{}

func (pe OpenshiftCLIEngine) CreateNamespace(name string, opts cli.OpenShiftCliOptions, config *rest.Config) (*cli.OpenshiftCreateReport, error) {

	k8sClientset, err := kubernetes.NewForConfig(config)

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

	createOptions := metav1.CreateOptions{
		metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		[]string{},
		"",
	}
	log.Debug("Namespace created: ", name)

	resp, err := k8sClientset.CoreV1().
		Namespaces().
		Create(context.Background(), nsSpec, createOptions)

	if err != nil {
		log.Error(fmt.Sprintf("error while creating Namespace: %s", name), err)
		return &cli.OpenshiftCreateReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}

	log.Debug("Namespace created: ", name)
	log.Trace("Received Namespace object from API server: ", resp)

	return &cli.OpenshiftCreateReport{
		Stdout: nsSpec.String(),
		Stderr: "",
	}, nil
}

func (pe OpenshiftCLIEngine) DeleteNamespace(name string, opts cli.OpenShiftCliOptions, config *rest.Config) error {
	k8sClientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return err
	}
	log.Debug("Deleting namespace: " + name)
	return k8sClientset.CoreV1().
		Namespaces().
		Delete(context.Background(), name, metav1.DeleteOptions{})
}

func (pe OpenshiftCLIEngine) CreateOperatorGroup(data cli.OperatorGroupData, opts cli.OpenShiftCliOptions, config *rest.Config) (*cli.OpenshiftCreateReport, error) {

	crdClient, err := client.OperatorGroupClient(config)
	if err != nil {
		log.Error("unable to create a client for OperatorGroup: ", err)
		return nil, err
	}
	// https://github.com/operator-framework/api/blob/master/pkg/operators/v1/operatorgroup_types.go
	operatorGroup := &operatorv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:   data.Name,
			Labels: opts.Labels,
		},
		Spec: operatorv1.OperatorGroupSpec{
			TargetNamespaces: data.TargetNamespaces,
		},
		Status: operatorv1.OperatorGroupStatus{
			Namespaces: data.TargetNamespaces,
		},
	}

	log.Debug(fmt.Sprintf("Creating OperatorGroup %s in namespace %s", data.Name, opts.Namespace))
	resp, err := crdClient.OperatorGroup(opts.Namespace).Create(operatorGroup)

	if err != nil {
		log.Error(fmt.Sprintf("error while creating OperatorGroup: %s", data.Name), err)
		return &cli.OpenshiftCreateReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}
	log.Debug(fmt.Sprintf("OperatorGroup %s created successfully in namespace %s", data.Name, opts.Namespace))
	log.Trace("Reveived OperatorGroup object from API server: ", resp)

	return &cli.OpenshiftCreateReport{
		Stdout: fmt.Sprintf("%#v", resp),
		Stderr: "",
	}, nil
}

func (pe OpenshiftCLIEngine) DeleteOperatorGroup(name string, opts cli.OpenShiftCliOptions, config *rest.Config) error {
	crdClient, err := client.OperatorGroupClient(config)
	if err != nil {
		log.Error("unable to create a client for OperatorGroup: ", err)
		return err
	}
	log.Debug(fmt.Sprintf("Deleting OperatorGroup %s in namespace %s", name, opts.Namespace))

	return crdClient.OperatorGroup(opts.Namespace).Delete(name, &metav1.DeleteOptions{})
}

func (pe OpenshiftCLIEngine) CreateCatalogSource(data cli.CatalogSourceData, opts cli.OpenShiftCliOptions, config *rest.Config) (*cli.OpenshiftCreateReport, error) {

	crdClient, err := client.CatalogSourceClient(config)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return nil, err
	}
	// https://github.com/operator-framework/api/blob/master/pkg/operators/v1alpha1/catalogsource_types.go
	catalogSource := &operatorv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:   data.Name,
			Labels: opts.Labels,
		},
		Spec: operatorv1alpha1.CatalogSourceSpec{
			DisplayName: data.Name,
			SourceType:  operatorv1alpha1.SourceTypeGrpc,
			Image:       data.Image,
		},
		Status: operatorv1alpha1.CatalogSourceStatus{},
	}
	log.Debug(fmt.Sprintf("Creating CatalogSource %s in namespace %s", data.Name, opts.Namespace))
	resp, err := crdClient.CatalogSource(opts.Namespace).Create(catalogSource)

	if err != nil {
		log.Error(fmt.Sprintf("error while creating CatalogSource: %s", data.Name), err)
		return &cli.OpenshiftCreateReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}
	log.Debug(fmt.Sprintf("CatalogSource %s is created succeesully in namespace %s", data.Name, opts.Namespace))

	return &cli.OpenshiftCreateReport{
		Stdout: fmt.Sprintf("%#v", resp),
		Stderr: "",
	}, nil
}

func (pe OpenshiftCLIEngine) DeleteCatalogSource(name string, opts cli.OpenShiftCliOptions, config *rest.Config) error {
	crdClient, err := client.CatalogSourceClient(config)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return err
	}

	log.Debug(fmt.Sprintf("Deleting CatalogSource %s in namespace %s", name, opts.Namespace))

	return crdClient.CatalogSource(opts.Namespace).Delete(name, &metav1.DeleteOptions{})
}

func (pe OpenshiftCLIEngine) CreateSubscription(data cli.SubscriptionData, opts cli.OpenShiftCliOptions, config *rest.Config) (*cli.OpenshiftCreateReport, error) {

	crdClient, err := client.SubscriptionClient(config)
	if err != nil {
		log.Error("unable to create a client for Subscription: ", err)
		return nil, err
	}
	// https://github.com/operator-framework/api/blob/master/pkg/operators/v1alpha1/subscription_types.go
	subscription := &operatorv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:   data.Name,
			Labels: opts.Labels,
		},
		Spec: &operatorv1alpha1.SubscriptionSpec{
			CatalogSource:          data.CatalogSource,
			CatalogSourceNamespace: data.CatalogSourceNamespace,
			Channel:                data.Channel,
			Package:                data.Package,
		},
		Status: operatorv1alpha1.SubscriptionStatus{},
	}
	log.Debug(fmt.Sprintf("Creating Subscription %s in namespace %s", data.Name, opts.Namespace))

	resp, err := crdClient.Subscription(opts.Namespace).Create(subscription)

	if err != nil {
		log.Error(fmt.Sprintf("error while creating Subscription: %s", data.Name), err)
		return &cli.OpenshiftCreateReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}
	log.Debug(fmt.Sprintf("Subscription %s is created successfully in namespace %s", data.Name, opts.Namespace))
	log.Trace("Received Subscription object from API server: ", resp)

	return &cli.OpenshiftCreateReport{
		Stdout: fmt.Sprintf("%#v", resp),
		Stderr: "",
	}, nil
}

func (pe OpenshiftCLIEngine) GetSubscription(name string, opts cli.OpenShiftCliOptions, config *rest.Config) (*operatorv1alpha1.Subscription, error) {
	crdClient, err := client.SubscriptionClient(config)
	if err != nil {
		log.Error("unable to create a client for subscription: ", err)
		return nil, err
	}

	return crdClient.Subscription(opts.Namespace).Get(name)

}

func (pe OpenshiftCLIEngine) DeleteSubscription(name string, opts cli.OpenShiftCliOptions, config *rest.Config) error {
	crdClient, err := client.SubscriptionClient(config)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return err
	}
	log.Debug(fmt.Sprintf("Deleting Subscription %s in namespace %s", name, opts.Namespace))

	return crdClient.Subscription(opts.Namespace).Delete(name, &metav1.DeleteOptions{})
}

func (pe OpenshiftCLIEngine) GetCSV(name string, opts cli.OpenShiftCliOptions, config *rest.Config) (*operatorv1alpha1.ClusterServiceVersion, error) {

	// https://github.com/operator-framework/api/blob/master/pkg/operators/v1alpha1/clusterserviceversion_types.go
	crdClient, err := client.ClusterServiceVersionClient(config)
	if err != nil {
		log.Error("unable to create a client for csv: ", err)
		return nil, err
	}

	return crdClient.ClusterServiceVersion(opts.Namespace).Get(name)
}
