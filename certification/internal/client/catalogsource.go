package client

import (
	"context"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
)

type CatalogSourceV1Alpha1Client struct {
	restClient rest.Interface
}

type CatalogSourceInterface interface {
	Create(obj *operatorv1alpha1.CatalogSource) (*operatorv1alpha1.CatalogSource, error)
	Update(obj *operatorv1alpha1.CatalogSource) (*operatorv1alpha1.CatalogSource, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Get(name string) (*operatorv1alpha1.CatalogSource, error)
}

type catalogSourceClient struct {
	client rest.Interface
	ns     string
}

func (c *CatalogSourceV1Alpha1Client) CatalogSource(namespace string) CatalogSourceInterface {
	return catalogSourceClient{
		client: c.restClient,
		ns:     namespace,
	}
}

func (c catalogSourceClient) Create(obj *operatorv1alpha1.CatalogSource) (*operatorv1alpha1.CatalogSource, error) {
	// TODO not to fail the creation if the resource already exists
	result := &operatorv1alpha1.CatalogSource{}
	err := c.client.Post().
		Namespace(c.ns).
		Resource("catalogsources").
		Body(obj).
		Do(context.Background()).
		Into(result)

	return result, err
}

func (c catalogSourceClient) Update(obj *operatorv1alpha1.CatalogSource) (*operatorv1alpha1.CatalogSource, error) {
	result := &operatorv1alpha1.CatalogSource{}

	err := c.client.Put().
		Namespace(c.ns).
		Resource("catalogsources").
		Body(obj).
		Do(context.Background()).
		Into(result)

	return result, err
}

func (c catalogSourceClient) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.
		Delete().
		Namespace(c.ns).
		Resource("catalogsources").
		Name(name).
		Body(options).
		Do(context.Background()).
		Error()
}

func (c catalogSourceClient) Get(name string) (*operatorv1alpha1.CatalogSource, error) {
	result := &operatorv1alpha1.CatalogSource{}

	err := c.client.Get().
		Namespace(c.ns).
		Resource("catalogsources").
		Name(name).
		Do(context.Background()).
		Into(result)

	return result, err
}

func CatalogSourceClient(cfg *rest.Config) (*CatalogSourceV1Alpha1Client, error) {
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
	return &CatalogSourceV1Alpha1Client{restClient: client}, nil
}
