package client

import (
	"context"

	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type operatorGroupClient struct {
	client runtimeclient.Client
}

func (c operatorGroupClient) Create(ctx context.Context, data cli.OperatorGroupData) (*operatorv1.OperatorGroup, error) {
	operatorGroup := &operatorv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: data.Name,
		},
		Spec: operatorv1.OperatorGroupSpec{
			TargetNamespaces: data.TargetNamespaces,
		},
	}
	err := c.client.Create(ctx, operatorGroup)
	return operatorGroup, err
}

func (c operatorGroupClient) Delete(ctx context.Context, name string) error {
	operatorGroup := &operatorv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return c.client.Delete(ctx, operatorGroup)
}

func (c operatorGroupClient) Get(ctx context.Context, name string) (*operatorv1.OperatorGroup, error) {
	operatorGroup := &operatorv1.OperatorGroup{}
	err := c.client.Get(ctx, runtimeclient.ObjectKey{
		Name: name,
	}, operatorGroup)

	return operatorGroup, err
}

func OperatorGroupClient(namespace string) (*operatorGroupClient, error) {
	scheme := runtime.NewScheme()
	operatorv1.AddToScheme(scheme)
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("could not get kubeconfig")
		return nil, err
	}

	client, err := runtimeclient.New(kubeconfig, runtimeclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Error("could not get operator group client")
		return nil, err
	}

	return &operatorGroupClient{
		client: runtimeclient.NewNamespacedClient(client, namespace),
	}, nil
}
