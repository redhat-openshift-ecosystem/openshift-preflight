package client

import (
	"context"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var csvKind schema.GroupVersionKind = schema.GroupVersionKind{
	Group:   "operators.coreos.com",
	Kind:    "ClusterServiceVersion",
	Version: "v1alpha1",
}

type csvClient struct {
	client runtimeclient.Client
}

func (c csvClient) Get(ctx context.Context, name string) (*operatorv1alpha1.ClusterServiceVersion, error) {
	csv := &operatorv1alpha1.ClusterServiceVersion{}
	err := c.client.Get(ctx, runtimeclient.ObjectKey{
		Name: name,
	}, csv)

	return csv, err
}

func CsvClient(namespace string) (*csvClient, error) {
	scheme := runtime.NewScheme()
	operatorv1alpha1.AddToScheme(scheme)
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("could not get kubeconfig")
		return nil, err
	}
	client, err := runtimeclient.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		log.Error("could not get csv client")
		return nil, err
	}

	return &csvClient{
		client: runtimeclient.NewNamespacedClient(client, namespace),
	}, nil
}
