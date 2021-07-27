package client

import (
	"context"

	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
)

type OperatorGroupV1Client struct {
	restClient rest.Interface
}

type OperatorGroupInterface interface {
	Create(obj *operatorv1.OperatorGroup) (*operatorv1.OperatorGroup, error)
	Update(obj *operatorv1.OperatorGroup) (*operatorv1.OperatorGroup, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Get(name string) (*operatorv1.OperatorGroup, error)
}

type operatorGroupClient struct {
	client rest.Interface
	ns     string
}

func (c *OperatorGroupV1Client) OperatorGroup(namespace string) OperatorGroupInterface {
	return operatorGroupClient{
		client: c.restClient,
		ns:     namespace,
	}
}

func (c operatorGroupClient) Create(obj *operatorv1.OperatorGroup) (*operatorv1.OperatorGroup, error) {
	result := &operatorv1.OperatorGroup{}
	err := c.client.Post().
		Namespace(c.ns).
		Resource("operatorgroups").
		Body(obj).
		Do(context.Background()).
		Into(result)

	return result, err
}

func (c operatorGroupClient) Update(obj *operatorv1.OperatorGroup) (*operatorv1.OperatorGroup, error) {
	result := &operatorv1.OperatorGroup{}

	err := c.client.Put().
		Namespace(c.ns).
		Resource("operatorgroups").
		Body(obj).
		Do(context.Background()).
		Into(result)

	return result, err
}

func (c operatorGroupClient) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.
		Delete().
		Namespace(c.ns).
		Resource("operatorgroups").
		Name(name).
		Body(options).
		Do(context.Background()).
		Error()
}

func (c operatorGroupClient) Get(name string) (*operatorv1.OperatorGroup, error) {
	result := &operatorv1.OperatorGroup{}
	err := c.client.Get().
		Namespace(c.ns).
		Resource("operatorgroups").
		Name(name).
		Do(context.Background()).
		Into(result)
	return result, err
}

func OperatorGroupClient(cfg *rest.Config) (*OperatorGroupV1Client, error) {
	scheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	config := *cfg
	config.GroupVersion = &operatorV1SchemeGV
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &OperatorGroupV1Client{restClient: client}, nil
}
