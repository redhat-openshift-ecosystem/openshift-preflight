package shell

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	client "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/client"
	fileutil "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/file"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

type OpenshiftCLIEngine struct{}

var config *rest.Config
var err error

func init() {
	var kubeconfig *string

	if home := fileutil.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "")
	}

	flag.Parse()

	config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Error("unable to load config from kubeconfig file: ", err)
	}
}

func (pe OpenshiftCLIEngine) CreateNamespace(name string, opts cli.OpenShiftCliOptions) (*cli.OpenshiftCreateReport, error) {

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

	resp, err := k8sClientset.CoreV1().
		Namespaces().
		Create(context.Background(), nsSpec, createOptions)

	if err != nil {
		log.Error("error while creating Namespace: %v", err)
		return &cli.OpenshiftCreateReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}

	log.Debug("Namespace created: %v", resp)

	return &cli.OpenshiftCreateReport{
		Stdout: nsSpec.String(),
		Stderr: "",
	}, nil
}

func (pe OpenshiftCLIEngine) DeleteNamespace(name string, opts cli.OpenShiftCliOptions) error {
	k8sClientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		log.Error("unable to obtain k8s client: ", err)
		return err
	}

	return k8sClientset.CoreV1().
		Namespaces().
		Delete(context.Background(), name, metav1.DeleteOptions{})
}

func (pe OpenshiftCLIEngine) CreateOperatorGroup(data cli.OperatorGroupData, opts cli.OpenShiftCliOptions) (*cli.OpenshiftCreateReport, error) {

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

	resp, err := crdClient.OperatorGroup(opts.Namespace).Create(operatorGroup)

	if err != nil {
		log.Error("error while creating OperatorGroup: %v", err)
		return &cli.OpenshiftCreateReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}

	log.Debug("OperatorGroup created: %v", resp)

	return &cli.OpenshiftCreateReport{
		Stdout: fmt.Sprintf("%#v", resp),
		Stderr: "",
	}, nil
}

// opts could be used to pass Delete options to Delete method. It is not being used at the moment.
func (pe OpenshiftCLIEngine) DeleteOperatorGroup(name string, opts cli.OpenShiftCliOptions) error {
	crdClient, err := client.OperatorGroupClient(config)
	if err != nil {
		log.Error("unable to create a client for OperatorGroup: ", err)
		return err
	}
	return crdClient.OperatorGroup(opts.Namespace).Delete(name, &metav1.DeleteOptions{})
}

func (pe OpenshiftCLIEngine) CreateCatalogSource(data cli.CatalogSourceData, opts cli.OpenShiftCliOptions) (*cli.OpenshiftCreateReport, error) {

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

	resp, err := crdClient.CatalogSource(opts.Namespace).Create(catalogSource)

	if err != nil {
		log.Error("error while creating CatalogSource: %v", err)
		return &cli.OpenshiftCreateReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}

	log.Debug("CatalogSource created: %v", resp)

	return &cli.OpenshiftCreateReport{
		Stdout: fmt.Sprintf("%#v", resp),
		Stderr: "",
	}, nil
}

func (pe OpenshiftCLIEngine) DeleteCatalogSource(name string, opts cli.OpenShiftCliOptions) error {
	crdClient, err := client.CatalogSourceClient(config)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return err
	}
	return crdClient.CatalogSource(opts.Namespace).Delete(name, &metav1.DeleteOptions{})
}

func (pe OpenshiftCLIEngine) CreateSubscription(data cli.SubscriptionData, opts cli.OpenShiftCliOptions) (*cli.OpenshiftCreateReport, error) {

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
		Status: operatorv1alpha1.SubscriptionStatus{}, //TODO
	}

	resp, err := crdClient.Subscription(opts.Namespace).Create(subscription)

	if err != nil {
		log.Error("error while creating Subscription: %v", err)
		return &cli.OpenshiftCreateReport{
			Stdout: "",
			Stderr: err.Error(),
		}, err
	}

	log.Debug("Subscription created: %v", resp)

	return &cli.OpenshiftCreateReport{
		Stdout: fmt.Sprintf("%#v", resp),
		Stderr: "",
	}, nil
}

func (pe OpenshiftCLIEngine) DeleteSubscription(name string, opts cli.OpenShiftCliOptions) error {
	crdClient, err := client.SubscriptionClient(config)
	if err != nil {
		log.Error("unable to create a client for CatalogSource: ", err)
		return err
	}
	return crdClient.Subscription(opts.Namespace).Delete(name, &metav1.DeleteOptions{})
}
