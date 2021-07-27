package k8s

import (
	"context"
	"fmt"
	"time"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/tools/clientcmd"
)

const (
	ooNamespace   = "argocd"
	operatorGroup = "argocd-operator"
	catalogSource = "argocd-catalog"
	catalogImage  = "quay.io/jmckind/argocd-operator-registry@sha256:82d6ebd5bbfc9c2b1a5058d29dd01b7fee5e3557f4936c3ed47b633900513b11"

	catalogSourceNs  = "openshift-marketplace"
	subscriptionName = "argocd-operator"
	packageName      = "argocd-operator"
	channel          = "alpha"
)

var targetNamespaces []string = []string{"argocd"}

type DeployableByOlmCheck struct{}

func (p *DeployableByOlmCheck) Validate(image string) (bool, error) {

	var k8sconfig *rest.Config
	var err error

	kubeconfig := viper.GetString("kubeconfig")

	if len(kubeconfig) > 0 {
		k8sconfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)

		if err != nil {
			return false, err
		}
	}

	// create k8s custom resources for the operator deployment
	p.setUp(k8sconfig)

	var csv *operatorv1alpha1.ClusterServiceVersion
	var installedCSV string
	var subs *operatorv1alpha1.Subscription

	// query API server for the installed CSV field of the created subscription
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	for {
		subs, _ = openshiftEngine.GetSubscription(subscriptionName, cli.OpenShiftCliOptions{Namespace: ooNamespace}, k8sconfig)
		installedCSV = subs.Status.InstalledCSV
		// if the installedCSV field is present, stop the querying
		if len(installedCSV) > 0 {
			log.Debug(fmt.Sprintf("Subscription.status.installedCSV is %s", installedCSV))
			break
		}
		// if the context deadline exceeds, fail the check
		if ctx.Err() != nil {
			log.Error("failed to fetch .status.installedCSV from Subscription", ctx.Err())
			return false, errors.ErrK8sApiCallFailed
		}
		log.Debug("InstalledCSV is not ready yet, retrying...")
		time.Sleep(2 * time.Second)
	}

	log.Trace(fmt.Sprintf("Looking for csv %s in namespace %s", installedCSV, ooNamespace))

	// query API server for CSV by name
	ctx, cancel = context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	for {
		log.Debug("Waiting for ClusterServiceVersion to become ready...")
		csv, _ = openshiftEngine.GetCSV(installedCSV, cli.OpenShiftCliOptions{Namespace: ooNamespace}, k8sconfig)
		// if the CSV phase is succeeded or the context deadline exceeds, stop the querying
		if csv.Status.Phase == operatorv1alpha1.CSVPhaseSucceeded {
			log.Debug("CSV is created successfully: ", installedCSV)
			break
		}
		// if the context deadline exceeds, fail the check
		if ctx.Err() != nil {
			log.Error(fmt.Sprintf("failed to fetch the csv: %s ", installedCSV), ctx.Err())
			return false, errors.ErrK8sApiCallFailed
		}
		log.Debug("CSV is not ready yet, retrying...")
		time.Sleep(2 * time.Second)
	}

	defer p.cleanUp(k8sconfig)

	return true, nil
}

func (p *DeployableByOlmCheck) setUp(k8sconfig *rest.Config) {
	openshiftEngine.CreateNamespace(ooNamespace, cli.OpenShiftCliOptions{}, k8sconfig)
	openshiftEngine.CreateCatalogSource(cli.CatalogSourceData{Name: catalogSource, Image: catalogImage}, cli.OpenShiftCliOptions{Namespace: "openshift-marketplace"}, k8sconfig)
	openshiftEngine.CreateOperatorGroup(cli.OperatorGroupData{Name: operatorGroup, TargetNamespaces: targetNamespaces}, cli.OpenShiftCliOptions{Namespace: ooNamespace}, k8sconfig)
	subscriptionData := cli.SubscriptionData{
		Name:                   subscriptionName,
		Channel:                channel,
		CatalogSource:          catalogSource,
		CatalogSourceNamespace: catalogSourceNs,
		Package:                packageName,
	}
	openshiftEngine.CreateSubscription(subscriptionData, cli.OpenShiftCliOptions{Namespace: ooNamespace}, k8sconfig)
}

func (p *DeployableByOlmCheck) cleanUp(k8sconfig *rest.Config) {
	log.Trace("Deleting the resources created by Check")
	openshiftEngine.DeleteSubscription(subscriptionName, cli.OpenShiftCliOptions{}, k8sconfig)
	openshiftEngine.DeleteCatalogSource(catalogSource, cli.OpenShiftCliOptions{Namespace: "openshift-marketplace"}, k8sconfig)
	openshiftEngine.DeleteOperatorGroup(operatorGroup, cli.OpenShiftCliOptions{}, k8sconfig)
	openshiftEngine.DeleteNamespace(ooNamespace, cli.OpenShiftCliOptions{}, k8sconfig)
}

func (p *DeployableByOlmCheck) Name() string {
	return "DeployableByOLM"
}

// TODO
func (p *DeployableByOlmCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the operator bundle image could be deployed by OLM",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide", // Placeholder
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

// TODO
func (p *DeployableByOlmCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "It is recommened that your image be based upon the Red Hat Universal Base Image (UBI)",
		Suggestion: "Change the FROM directive in your Dockerfile or Containerfile to FROM registry.access.redhat.com/ubi8/ubi",
	}
}
