package client

import (
	"context"
	"log"

	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var operatorGroupKind schema.GroupVersionKind = schema.GroupVersionKind{
	Group:   "operators.coreos.com",
	Kind:    "OperatorGroup",
	Version: "v1",
}

type OperatorGroupInterface interface {
	Create(data cli.OperatorGroupData, opts cli.OpenshiftOptions) (operatorv1.OperatorGroup, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Get(name string, namespace string) operatorv1.OperatorGroup
	convert(u *unstructured.Unstructured) (*operatorv1.OperatorGroup, error)
}

type operatorGroupClient struct {
	client client.Client
	ns     string
}

func (c operatorGroupClient) Create(data cli.OperatorGroupData, opts cli.OpenshiftOptions) (*operatorv1.OperatorGroup, error) {

	u := &unstructured.Unstructured{}
	u.Object = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      data.Name,
			"namespace": opts.Namespace,
			"labels":    opts.Labels,
		},
		"spec": map[string]interface{}{
			"targetNamespaces": data.TargetNamespaces,
		},
	}
	u.SetGroupVersionKind(operatorGroupKind)

	err := c.client.Create(context.Background(), u)

	if err != nil {
		return nil, err
	}

	return c.convert(u)
}

func (c operatorGroupClient) Delete(name string, opts cli.OpenshiftOptions) error {

	u := &unstructured.Unstructured{}

	u.SetName(name)
	u.SetNamespace(opts.Namespace)
	u.SetGroupVersionKind(operatorGroupKind)

	return c.client.Delete(context.Background(), u)
}

func (c operatorGroupClient) Get(name string, namespace string) (*operatorv1.OperatorGroup, error) {

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(operatorGroupKind)

	err := c.client.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, u)
	if err != nil {
		return nil, err
	}

	return c.convert(u)
}

func (c operatorGroupClient) convert(u *unstructured.Unstructured) (*operatorv1.OperatorGroup, error) {
	var obj operatorv1.OperatorGroup
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(u.UnstructuredContent(), &obj)
	if err != nil {
		return nil, err
	}

	return &obj, nil
}

func OperatorGroupClient(namespace string) (*operatorGroupClient, error) {
	scheme := runtime.NewScheme()
	operatorv1alpha1.AddToScheme(scheme)
	kubeconfig := ctrl.GetConfigOrDie()
	controllerClient, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &operatorGroupClient{
		client: controllerClient,
		ns:     namespace,
	}, nil

}
