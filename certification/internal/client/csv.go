package client

import (
	"context"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
)

type ClusterServiceVersionV1Alpha1Client struct {
	restClient rest.Interface
}

type ClusterServiceVersionInterface interface {
	Get(name string) (*operatorv1alpha1.ClusterServiceVersion, error)
}

type clusterServiceVersionClient struct {
	client rest.Interface
	ns     string
}

func (c *ClusterServiceVersionV1Alpha1Client) ClusterServiceVersion(namespace string) ClusterServiceVersionInterface {
	return clusterServiceVersionClient{
		client: c.restClient,
		ns:     namespace,
	}
}

func (c clusterServiceVersionClient) Get(name string) (*operatorv1alpha1.ClusterServiceVersion, error) {
	result := &operatorv1alpha1.ClusterServiceVersion{}

	err := c.client.Get().
		Namespace(c.ns).
		Resource("clusterserviceversions").
		Name(name).
		Do(context.Background()).
		Into(result)

	return result, err
}

func ClusterServiceVersionClient(cfg *rest.Config) (*ClusterServiceVersionV1Alpha1Client, error) {
	scheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	config := *cfg
	config.GroupVersion = &OperatorV1Alpha1SchemeGV
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &ClusterServiceVersionV1Alpha1Client{restClient: client}, nil
}
