package engine

import (
	"context"
	"fmt"
	"os"

	configv1Client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/runtime"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func GetOpenshiftClusterVersion() (runtime.OpenshiftClusterVersion, error) {
	if _, ok := os.LookupEnv("KUBECONFIG"); !ok {
		return runtime.UnknownOpenshiftClusterVersion(), errors.ErrNoKubeconfig
	}
	kubeConfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error("unable to load the config, check if KUBECONFIG is set correctly: ", err)
		return runtime.UnknownOpenshiftClusterVersion(), err
	}
	configV1Client, err := configv1Client.NewForConfig(kubeConfig)
	if err != nil {
		log.Error("unable to create a client with the provided kubeconfig: ", err)
		return runtime.UnknownOpenshiftClusterVersion(), err
	}
	openshiftApiServer, err := configV1Client.ClusterOperators().Get(context.Background(), "openshift-apiserver", metav1.GetOptions{})
	if err != nil {
		log.Error("unable to get openshift-apiserver cluster operator: ", err)
		return runtime.UnknownOpenshiftClusterVersion(), err
	}

	log.Debug(fmt.Sprintf("fetching operator version and openshift-apiserver version %s from %s", openshiftApiServer.Status.Versions, kubeConfig.Host))
	return runtime.OpenshiftClusterVersion{
		Name:    "OpenShift",
		Version: openshiftApiServer.Status.Versions[1].Version,
	}, nil
}
