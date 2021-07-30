package k8s

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/tools/clientcmd"
)

const (
	catalogSourceNs = "openshift-marketplace"
)

var (
	k8sconfig                     *rest.Config
	err                           error
	subscriptionName, packageName string
	channel, ooNamespace          string
	operatorGroup, catalogSource  string
	catalogImage                  string
	targetNamespaces              []string = []string{ooNamespace}
)

type DeployableByOlmCheck struct{}

func (p *DeployableByOlmCheck) Validate(image string) (bool, error) {

	// create k8s custom resources for the operator deployment
	err = p.setUp()
	defer p.cleanUp()
	if err != nil {
		return false, err
	}

	var csv *operatorv1alpha1.ClusterServiceVersion
	var installedCSV string
	var subs *operatorv1alpha1.Subscription

	// query API server for the installed CSV field of the created subscription
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
			log.Error("failed to fetch .status.installedCSV from Subscription: ", ctx.Err())
			return false, errors.ErrK8sApiCallFailed
		}
		log.Debug("InstalledCSV is not ready yet, retrying...")
		time.Sleep(2 * time.Second)
	}

	log.Trace(fmt.Sprintf("Looking for csv %s in namespace %s", installedCSV, ooNamespace))

	// query API server for CSV by name
	ctx, cancel = context.WithTimeout(context.Background(), 90*time.Second)
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

	return true, nil
}

func (p *DeployableByOlmCheck) setUp() error {
	kubeconfig := viper.GetString("kubeconfig")

	if len(kubeconfig) > 0 {
		k8sconfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)

		if err != nil {
			return err
		}
	}

	catalogImage = viper.GetString("catalogImage")
	channel = viper.GetString("channel")
	packageName = viper.GetString("package")
	operatorGroup = viper.GetString("appName") + "-og"
	ooNamespace = viper.GetString("installNamespace")
	subscriptionName = viper.GetString("appName") + "-sub"
	catalogSource = viper.GetString("appName") + "-cs"

	_, err = openshiftEngine.CreateNamespace(ooNamespace, cli.OpenShiftCliOptions{}, k8sconfig)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	_, err = openshiftEngine.CreateCatalogSource(cli.CatalogSourceData{Name: catalogSource, Image: catalogImage}, cli.OpenShiftCliOptions{Namespace: catalogSourceNs}, k8sconfig)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	_, err = openshiftEngine.CreateOperatorGroup(cli.OperatorGroupData{Name: operatorGroup, TargetNamespaces: targetNamespaces}, cli.OpenShiftCliOptions{Namespace: ooNamespace}, k8sconfig)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	subscriptionData := cli.SubscriptionData{
		Name:                   subscriptionName,
		Channel:                channel,
		CatalogSource:          catalogSource,
		CatalogSourceNamespace: catalogSourceNs,
		Package:                packageName,
	}
	_, err = openshiftEngine.CreateSubscription(subscriptionData, cli.OpenShiftCliOptions{Namespace: ooNamespace}, k8sconfig)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (p *DeployableByOlmCheck) cleanUp() {
	log.Debug("Dumping data in artifacts/ directory")

	subs, err := openshiftEngine.GetSubscription(subscriptionName, cli.OpenShiftCliOptions{Namespace: ooNamespace}, k8sconfig)
	if err != nil {
		log.Error("unable to retrieve the subscription")
	}
	p.writeToFile(subs, subscriptionName, "subscription")

	cs, err := openshiftEngine.GetCatalogSource(catalogSource, cli.OpenShiftCliOptions{Namespace: catalogSourceNs}, k8sconfig)
	if err != nil {
		log.Error("unable to retrieve the catalogsource")
	}
	p.writeToFile(cs, catalogSource, "catalogsource")

	og, err := openshiftEngine.GetOperatorGroup(operatorGroup, cli.OpenShiftCliOptions{Namespace: ooNamespace}, k8sconfig)
	if err != nil {
		log.Error("unable to retrieve the operatorgroup")
	}
	p.writeToFile(og, operatorGroup, "operatorgroup")

	ns, err := openshiftEngine.GetNamespace(ooNamespace, k8sconfig)
	if err != nil {
		log.Error("unable to retrieve the namespace")
	}
	p.writeToFile(ns, ooNamespace, "namespace")

	log.Trace("Deleting the resources created by Check")
	openshiftEngine.DeleteSubscription(subscriptionName, cli.OpenShiftCliOptions{}, k8sconfig)
	openshiftEngine.DeleteCatalogSource(catalogSource, cli.OpenShiftCliOptions{Namespace: catalogSourceNs}, k8sconfig)
	openshiftEngine.DeleteOperatorGroup(operatorGroup, cli.OpenShiftCliOptions{}, k8sconfig)
	openshiftEngine.DeleteNamespace(ooNamespace, cli.OpenShiftCliOptions{}, k8sconfig)
}

func (p *DeployableByOlmCheck) writeToFile(data interface{}, resource string, resourceType string) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		log.Error("unable to serialize the subscription")
	}
	err = ioutil.WriteFile(fmt.Sprintf("artifacts/%s-%s.yaml", resource, resourceType), yamlData, 0644)
	if err != nil {
		log.Error("failed to write the subscription object to the file")
	}
}

func (p *DeployableByOlmCheck) Name() string {
	return "DeployableByOLM"
}

func (p *DeployableByOlmCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the operator could be deployed by OLM",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide", // Placeholder
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *DeployableByOlmCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "It is recommened that your operator could be deployed by OLM",
		Suggestion: "Follow the guidelines on the operatorsdk website to learn how to package your operator https://sdk.operatorframework.io/docs/olm-integration/cli-overview/",
	}
}
