package client

import (
	"context"
	"log"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var subscriptionKind schema.GroupVersionKind = schema.GroupVersionKind{
	Group:   "operators.coreos.com",
	Kind:    "Subscription",
	Version: "v1alpha1",
}

type SubscriptionInterface interface {
	Create(data cli.SubscriptionData, opts cli.OpenshiftOptions) (*operatorv1alpha1.Subscription, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Get(name string, namespace string) operatorv1alpha1.Subscription
	convert(u *unstructured.Unstructured) (*operatorv1alpha1.Subscription, error)
}

type subscriptionClient struct {
	client client.Client
	ns     string
}

func (c subscriptionClient) Create(data cli.SubscriptionData, opts cli.OpenshiftOptions) (*operatorv1alpha1.Subscription, error) {

	u := &unstructured.Unstructured{}
	u.Object = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      data.Name,
			"namespace": opts.Namespace,
			"labels":    opts.Labels,
		},
		"spec": map[string]interface{}{
			"name":            data.Package,
			"source":          data.CatalogSource,
			"channel":         data.Channel,
			"sourceNamespace": data.CatalogSourceNamespace,
		},
	}

	u.SetGroupVersionKind(subscriptionKind)

	err := c.client.Create(context.Background(), u)

	if err != nil {
		return nil, err
	}

	return c.convert(u)
}

func (c subscriptionClient) Delete(name string, opts cli.OpenshiftOptions) error {

	u := &unstructured.Unstructured{}

	u.SetName(name)
	u.SetNamespace(opts.Namespace)
	u.SetGroupVersionKind(subscriptionKind)

	return c.client.Delete(context.Background(), u)
}

func (c subscriptionClient) Get(name string, namespace string) (*operatorv1alpha1.Subscription, error) {

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(subscriptionKind)

	err := c.client.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, u)
	if err != nil {
		return nil, err
	}

	return c.convert(u)
}

func (c subscriptionClient) convert(u *unstructured.Unstructured) (*operatorv1alpha1.Subscription, error) {
	var obj operatorv1alpha1.Subscription
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(u.UnstructuredContent(), &obj)
	if err != nil {
		return nil, err
	}

	return &obj, nil
}

func SubscriptionClient(namespace string) (*subscriptionClient, error) {
	scheme := runtime.NewScheme()
	operatorv1alpha1.AddToScheme(scheme)
	kubeconfig := ctrl.GetConfigOrDie()
	controllerClient, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &subscriptionClient{
		client: controllerClient,
		ns:     namespace,
	}, nil

}
