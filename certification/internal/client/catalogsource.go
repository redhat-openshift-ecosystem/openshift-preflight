package client

import (
	"context"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	log "github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type catalogSourceClient struct {
	client runtimeclient.Client
}

func (c catalogSourceClient) Create(ctx context.Context, data cli.CatalogSourceData) (*operatorv1alpha1.CatalogSource, error) {
	catalogSource := &operatorv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name: data.Name,
		},
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType:  operatorv1alpha1.SourceTypeGrpc,
			Image:       data.Image,
			DisplayName: data.Name,
			Secrets:     data.Secrets,
		},
	}
	err := c.client.Create(ctx, catalogSource)
	return catalogSource, err
}

func (c catalogSourceClient) Delete(ctx context.Context, name string) error {
	catalogSource := &operatorv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return c.client.Delete(ctx, catalogSource)
}

func (c catalogSourceClient) Get(ctx context.Context, name string) (*operatorv1alpha1.CatalogSource, error) {
	catalogSource := &operatorv1alpha1.CatalogSource{}
	err := c.client.Get(ctx, runtimeclient.ObjectKey{
		Name: name,
	}, catalogSource)

	return catalogSource, err
}

func CatalogSourceClient(namespace string) (*catalogSourceClient, error) {
	scheme := runtime.NewScheme()
	operatorv1alpha1.AddToScheme(scheme)
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("could not read kubeconfig")
		return nil, err
	}

	client, err := runtimeclient.New(kubeconfig, runtimeclient.Options{Scheme: scheme})
	if err != nil {
		log.Error("could not get catalog source client")
		return nil, err
	}

	return &catalogSourceClient{
		client: runtimeclient.NewNamespacedClient(client, namespace),
	}, nil
}
