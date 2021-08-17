package k8s

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	viperutil "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/viper"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/tools/clientcmd"
)

const (
	catalogSourceNS = "openshift-marketplace"
)

var (
	k8sconfig            *rest.Config
	err                  error
	packageName, app     string
	channel, ooNamespace string
	catalogImage         string
	targetNamespaces     []string
)

type DeployableByOlmCheck struct{}

func (p *DeployableByOlmCheck) Validate(image string) (bool, error) {

	// create k8s custom resources for the operator deployment
	err = p.setUp()
	defer p.cleanUp()
	if err != nil {
		return false, err
	}
	installedCSV, err := p.installedCSV()
	if err != nil {
		return false, err
	}
	return p.isCSVReady(installedCSV)
}

func (p *DeployableByOlmCheck) setUp() error {
	kubeconfig := os.Getenv("KUBECONFIG")
	if len(kubeconfig) > 0 {
		k8sconfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return err
		}
	}

	key := "catalogimage"
	catalogImage, err = viperutil.GetString(key)
	if err != nil {
		log.Error(fmt.Sprintf("To set the key, export PFLT_%s or add %s:<value> to config.yaml in the current working directory", strings.ToUpper(key), key))
		return err
	}
	key = "channel"
	channel, err = viperutil.GetString(key)
	if err != nil {
		log.Error(fmt.Sprintf("To set the key, export PFLT_%s or add `%s:<value>` to config.yaml in the current working directory", strings.ToUpper(key), key))
		return err
	}
	key = "package"
	packageName, err = viperutil.GetString(key)
	if err != nil {
		log.Error(fmt.Sprintf("To set the key, export PFLT_%s or add %s:<value> to config.yaml in the current working directory", strings.ToUpper(key), key))
		return err
	}
	key = "app"
	app, err = viperutil.GetString(key)
	if err != nil {
		log.Error(fmt.Sprintf("To set the key, export PFLT_%s or add %s:<value> to config.yaml in the current working directory", strings.ToUpper(key), key))
		return err
	}
	key = "namespace"
	ooNamespace, err = viperutil.GetString(key)
	if err != nil {
		log.Error(fmt.Sprintf("To set the key, export PFLT_%s or add `%s:<value>` to config.yaml in the current working directory", strings.ToUpper(key), key))
		return err
	}

	_, err = openshiftEngine.CreateNamespace(ooNamespace, cli.OpenshiftOptions{}, k8sconfig)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	_, err = openshiftEngine.CreateCatalogSource(cli.CatalogSourceData{Name: app, Image: catalogImage}, cli.OpenshiftOptions{Namespace: catalogSourceNS}, k8sconfig)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	targetNamespaces = []string{ooNamespace}
	_, err = openshiftEngine.CreateOperatorGroup(cli.OperatorGroupData{Name: app, TargetNamespaces: targetNamespaces}, cli.OpenshiftOptions{Namespace: ooNamespace}, k8sconfig)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	subscriptionData := cli.SubscriptionData{
		Name:                   app,
		Channel:                channel,
		CatalogSource:          app,
		CatalogSourceNamespace: catalogSourceNS,
		Package:                packageName,
	}
	_, err = openshiftEngine.CreateSubscription(subscriptionData, cli.OpenshiftOptions{Namespace: ooNamespace}, k8sconfig)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (p *DeployableByOlmCheck) isCSVReady(installedCSV string) (bool, error) {
	log.Trace(fmt.Sprintf("Looking for csv %s in namespace %s", installedCSV, ooNamespace))

	// query API server for CSV by name
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	csvReadyDone := make(chan string, 1)

	go func() {
		defer close(csvReadyDone)
		for {
			log.Debug("Waiting for ClusterServiceVersion to become ready...")
			csv, _ := openshiftEngine.GetCSV(installedCSV, cli.OpenshiftOptions{Namespace: ooNamespace}, k8sconfig)
			// if the CSV phase is succeeded, stop the querying
			if csv.Status.Phase == operatorv1alpha1.CSVPhaseSucceeded {
				log.Debug("CSV is created successfully: ", installedCSV)
				csvReadyDone <- fmt.Sprintf("%#v", csv)
			}
			log.Debug("CSV is not ready yet, retrying...")
			time.Sleep(2 * time.Second)
		}
	}()

	select {
	case csv := <-csvReadyDone:
		return len(csv) > 0, nil
	case <-ctx.Done():
		log.Error(fmt.Sprintf("failed to fetch the csv %s: ", installedCSV), ctx.Err())
		return false, nil
	}

}

func (p *DeployableByOlmCheck) installedCSV() (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	installedCSVDone := make(chan string, 1)

	// query API server for the installed CSV field of the created subscription
	go func() {
		defer close(installedCSVDone)
		for {
			time.Sleep(2 * time.Second)
			log.Debug("Waiting for Subscription.status.installedCSV to become ready...")
			subs, _ := openshiftEngine.GetSubscription(app, cli.OpenshiftOptions{Namespace: ooNamespace}, k8sconfig)
			installedCSV := subs.Status.InstalledCSV
			// if the installedCSV field is present, stop the querying
			if len(installedCSV) > 0 {
				log.Debug(fmt.Sprintf("Subscription.status.installedCSV is %s", installedCSV))
				installedCSVDone <- installedCSV
			}
			log.Debug("Subscription.status.installedCSV is not set yet, retrying...")
		}
	}()

	select {
	case installedCSV := <-installedCSVDone:
		return installedCSV, nil
	case <-ctx.Done():
		log.Error("failed to fetch Subscription.status.installedCSV: ", ctx.Err())
		return "", errors.ErrK8sAPICallFailed
	}
}

func (p *DeployableByOlmCheck) cleanUp() {
	log.Debug("Dumping data in artifacts/ directory")

	subs, err := openshiftEngine.GetSubscription(app, cli.OpenshiftOptions{Namespace: ooNamespace}, k8sconfig)
	if err != nil {
		log.Error("unable to retrieve the subscription")
	}
	p.writeToFile(subs, app, "subscription")

	cs, err := openshiftEngine.GetCatalogSource(app, cli.OpenshiftOptions{Namespace: catalogSourceNS}, k8sconfig)
	if err != nil {
		log.Error("unable to retrieve the catalogsource")
	}
	p.writeToFile(cs, app, "catalogsource")

	og, err := openshiftEngine.GetOperatorGroup(app, cli.OpenshiftOptions{Namespace: ooNamespace}, k8sconfig)
	if err != nil {
		log.Error("unable to retrieve the operatorgroup")
	}
	p.writeToFile(og, app, "operatorgroup")

	ns, err := openshiftEngine.GetNamespace(ooNamespace, k8sconfig)
	if err != nil {
		log.Error("unable to retrieve the namespace")
	}
	p.writeToFile(ns, ooNamespace, "namespace")

	log.Trace("Deleting the resources created by Check")
	openshiftEngine.DeleteSubscription(app, cli.OpenshiftOptions{Namespace: ooNamespace}, k8sconfig)
	openshiftEngine.DeleteCatalogSource(app, cli.OpenshiftOptions{Namespace: catalogSourceNS}, k8sconfig)
	openshiftEngine.DeleteOperatorGroup(app, cli.OpenshiftOptions{Namespace: ooNamespace}, k8sconfig)
	openshiftEngine.DeleteNamespace(ooNamespace, cli.OpenshiftOptions{}, k8sconfig)
}

func (p *DeployableByOlmCheck) writeToFile(data interface{}, resource string, resourceType string) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		log.Error("unable to serialize the data")
	}
	err = ioutil.WriteFile(filepath.Join("artifacts", fmt.Sprintf("%s-%s.yaml", resource, resourceType)), yamlData, 0644)
	if err != nil {
		log.Error("failed to write the k8s object to the file")
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
		Message:    "It is required that your operator could be deployed by OLM",
		Suggestion: "Follow the guidelines on the operatorsdk website to learn how to package your operator https://sdk.operatorframework.io/docs/olm-integration/cli-overview/",
	}
}
