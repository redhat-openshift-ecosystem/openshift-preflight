package openshift

import (
	"context"
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"

	"github.com/go-logr/logr"
	configv1Client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func GetOpenshiftClusterVersion(ctx context.Context, kubeconfig []byte) (runtime.OpenshiftClusterVersion, error) {
	logger := logr.FromContextOrDiscard(ctx)
	if len(kubeconfig) == 0 {
		return runtime.UnknownOpenshiftClusterVersion(), fmt.Errorf("kubeconfig was not provided")
	}

	restconfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return runtime.UnknownOpenshiftClusterVersion(), fmt.Errorf("unable to load the config, check if KUBECONFIG is set correctly: %v", err)
	}

	configV1Client, err := configv1Client.NewForConfig(restconfig)
	if err != nil {
		return runtime.UnknownOpenshiftClusterVersion(), fmt.Errorf("unable to create a client with the provided kubeconfig: %v", err)
	}
	openshiftAPIServer, err := configV1Client.ClusterOperators().Get(context.Background(), "openshift-apiserver", metav1.GetOptions{})
	if err != nil {
		return runtime.UnknownOpenshiftClusterVersion(), fmt.Errorf("unable to get openshift-apiserver cluster operator: %v", err)
	}

	logger.V(log.DBG).Info("fetching operator version and openshift-apiserver version", "version", openshiftAPIServer.Status.Versions, "host", restconfig.Host)
	return runtime.OpenshiftClusterVersion{
		Name:    "OpenShift",
		Version: openshiftAPIServer.Status.Versions[1].Version,
	}, nil
}
