package client

import (
	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	config "sigs.k8s.io/controller-runtime/pkg/client/config"

	log "github.com/sirupsen/logrus"
)

func New() *client.Client {
	scheme := runtime.NewScheme()
	operatorv1alpha1.AddToScheme(scheme)
	operatorv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	kubeconfig, err := config.GetConfig()
	if err != nil {
		log.Error(err)
		return nil
	}
	client, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		log.Error(err)
		return nil
	}

	return &client
}
