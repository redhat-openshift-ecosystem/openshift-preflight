package openshift

import (
	"context"
	"fmt"
	"os"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"

	configv1Client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func GetOpenshiftClusterVersion() (runtime.OpenshiftClusterVersion, error) {
	if _, ok := os.LookupEnv("KUBECONFIG"); !ok {
		return runtime.UnknownOpenshiftClusterVersion(), fmt.Errorf("KUBECONFIG not specified")
	}
	kubeConfig, err := ctrl.GetConfig()
	if err != nil {
		return runtime.UnknownOpenshiftClusterVersion(), fmt.Errorf("unable to load the config, check if KUBECONFIG is set correctly: %v", err)
	}
	configV1Client, err := configv1Client.NewForConfig(kubeConfig)
	if err != nil {
		return runtime.UnknownOpenshiftClusterVersion(), fmt.Errorf("unable to create a client with the provided kubeconfig: %v", err)
	}
	openshiftAPIServer, err := configV1Client.ClusterOperators().Get(context.Background(), "openshift-apiserver", metav1.GetOptions{})
	if err != nil {
		return runtime.UnknownOpenshiftClusterVersion(), fmt.Errorf("unable to get openshift-apiserver cluster operator: %v", err)
	}

	log.Debug(fmt.Sprintf("fetching operator version and openshift-apiserver version %s from %s", openshiftAPIServer.Status.Versions, kubeConfig.Host))
	return runtime.OpenshiftClusterVersion{
		Name:    "OpenShift",
		Version: openshiftAPIServer.Status.Versions[1].Version,
	}, nil
}
