package client

import (
	"context"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
)

type SubscriptionV1Alpha1Client struct {
	restClient rest.Interface
}

type SubscriptionInterface interface {
	Create(obj *operatorv1alpha1.Subscription) (*operatorv1alpha1.Subscription, error)
	Update(obj *operatorv1alpha1.Subscription) (*operatorv1alpha1.Subscription, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Get(name string) (*operatorv1alpha1.Subscription, error)
}

type subscriptionClient struct {
	client rest.Interface
	ns     string
}

func (c *SubscriptionV1Alpha1Client) Subscription(namespace string) SubscriptionInterface {
	return subscriptionClient{
		client: c.restClient,
		ns:     namespace,
	}
}

func (c subscriptionClient) Create(obj *operatorv1alpha1.Subscription) (*operatorv1alpha1.Subscription, error) {
	// TODO not to fail the creation if the resource already exists
	result := &operatorv1alpha1.Subscription{}
	err := c.client.Post().
		Namespace(c.ns).
		Resource("subscriptions").
		Body(obj).
		Do(context.Background()).
		Into(result)

	return result, err
}

func (c subscriptionClient) Update(obj *operatorv1alpha1.Subscription) (*operatorv1alpha1.Subscription, error) {
	result := &operatorv1alpha1.Subscription{}

	err := c.client.Put().
		Namespace(c.ns).
		Resource("subscriptions").
		Body(obj).
		Do(context.Background()).
		Into(result)

	return result, err
}

func (c subscriptionClient) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.
		Delete().
		Namespace(c.ns).
		Resource("subscriptions").
		Name(name).
		Body(options).
		Do(context.Background()).
		Error()
}

func (c subscriptionClient) Get(name string) (*operatorv1alpha1.Subscription, error) {
	result := &operatorv1alpha1.Subscription{}
	err := c.client.Get().
		Namespace(c.ns).
		Resource("subscriptions").
		Name(name).
		Do(context.Background()).
		Into(result)
	return result, err
}

func SubscriptionClient(cfg *rest.Config) (*SubscriptionV1Alpha1Client, error) {
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
	return &SubscriptionV1Alpha1Client{restClient: client}, nil
}
