package client

import (
	"context"
	"log"

	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var csvKind schema.GroupVersionKind = schema.GroupVersionKind{
	Group:   "operators.coreos.com",
	Kind:    "ClusterServiceVersion",
	Version: "v1alpha1",
}

type CSVInterface interface {
	Get(name string, namespace string) operatorv1.OperatorGroup
	convert(u *unstructured.Unstructured) (*operatorv1.OperatorGroup, error)
}

type csvClient struct {
	client client.Client
	ns     string
}

func (c csvClient) Get(name string, namespace string) (*operatorv1alpha1.ClusterServiceVersion, error) {

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(csvKind)

	err := c.client.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, u)
	if err != nil {
		return nil, err
	}

	return c.convert(u)
}

func (c csvClient) convert(u *unstructured.Unstructured) (*operatorv1alpha1.ClusterServiceVersion, error) {
	var obj operatorv1alpha1.ClusterServiceVersion
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(u.UnstructuredContent(), &obj)
	if err != nil {
		return nil, err
	}

	return &obj, nil
}

func CsvClient(namespace string) (*csvClient, error) {
	scheme := runtime.NewScheme()
	operatorv1alpha1.AddToScheme(scheme)
	kubeconfig := ctrl.GetConfigOrDie()
	controllerClient, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &csvClient{
		client: controllerClient,
		ns:     namespace,
	}, nil

}
