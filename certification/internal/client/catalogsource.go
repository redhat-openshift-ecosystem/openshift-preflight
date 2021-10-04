package client

import (
	"context"
	"log"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var catalogSourceKind schema.GroupVersionKind = schema.GroupVersionKind{
	Group:   "operators.coreos.com",
	Kind:    "CatalogSource",
	Version: "v1alpha1",
}

type CatalogSourceInterface interface {
	Create(data cli.CatalogSourceData, opts cli.OpenshiftOptions) (*operatorv1alpha1.CatalogSource, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Get(name string, namespace string) operatorv1alpha1.CatalogSource
	convert(u *unstructured.Unstructured) (*operatorv1alpha1.CatalogSource, error)
}

type catalogSourceClient struct {
	client client.Client
	ns     string
}

func (c catalogSourceClient) Create(data cli.CatalogSourceData, opts cli.OpenshiftOptions) (*operatorv1alpha1.CatalogSource, error) {

	u := &unstructured.Unstructured{}
	u.Object = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      data.Name,
			"namespace": opts.Namespace,
			"labels":    opts.Labels,
		},
		"spec": map[string]interface{}{
			"sourceType":  operatorv1alpha1.SourceTypeGrpc,
			"image":       data.Image,
			"displayName": data.Name,
			"secrets":     data.Secrets,
		},
	}
	u.SetGroupVersionKind(catalogSourceKind)

	err := c.client.Create(context.Background(), u)

	if err != nil {
		return nil, err
	}

	return c.convert(u)
}

func (c catalogSourceClient) Delete(name string, opts cli.OpenshiftOptions) error {

	u := &unstructured.Unstructured{}

	u.SetName(name)
	u.SetNamespace(opts.Namespace)
	u.SetGroupVersionKind(catalogSourceKind)

	return c.client.Delete(context.Background(), u)
}

func (c catalogSourceClient) Get(name string, namespace string) (*operatorv1alpha1.CatalogSource, error) {

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(catalogSourceKind)

	err := c.client.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, u)
	if err != nil {
		return nil, err
	}

	return c.convert(u)
}

func (c catalogSourceClient) convert(u *unstructured.Unstructured) (*operatorv1alpha1.CatalogSource, error) {
	var obj operatorv1alpha1.CatalogSource
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(u.UnstructuredContent(), &obj)
	if err != nil {
		return nil, err
	}

	return &obj, nil
}

func CatalogSourceClient(namespace string) (*catalogSourceClient, error) {
	scheme := runtime.NewScheme()
	operatorv1alpha1.AddToScheme(scheme)
	kubeconfig := ctrl.GetConfigOrDie()
	controllerClient, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &catalogSourceClient{
		client: controllerClient,
		ns:     namespace,
	}, nil

}
