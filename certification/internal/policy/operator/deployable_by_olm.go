package operator

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
	openshiftengine "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/engine"
	containerutil "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/container"
	viperutil "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/viper"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	kubeErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd"
)

type OperatorData struct {
	CatalogImage     string
	Channel          string
	PackageName      string
	App              string
	InstallNamespace string
}

type DeployableByOlmCheck struct{}

func (p *DeployableByOlmCheck) Validate(bundleRef certification.ImageReference) (bool, error) {

	// create a new instance of openshift engine
	openshiftEngine, err := p.newOpenshiftEngine()
	if err != nil {
		return false, err
	}

	// retrieve the required data
	operatorData, err := p.operatorMetadata(bundleRef)
	if err != nil {
		return false, err
	}

	// create k8s custom resources for the operator deployment
	err = p.setUp(*operatorData, *openshiftEngine)
	defer p.cleanUp(*operatorData, *openshiftEngine)

	if err != nil {
		return false, err
	}

	installedCSV, err := p.installedCSV(*operatorData, *openshiftEngine)
	if err != nil {
		return false, err
	}

	return p.isCSVReady(installedCSV, *operatorData, *openshiftEngine)
}

func (p *DeployableByOlmCheck) newOpenshiftEngine() (*openshiftengine.OpenshiftEngine, error) {
	k8sconfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		log.Error("unable to create a kubernetes client: ", err)
		return nil, err
	}

	return &openshiftengine.OpenshiftEngine{
		KubeConfig: k8sconfig,
	}, nil
}

func (p *DeployableByOlmCheck) operatorMetadata(bundleRef certification.ImageReference) (*OperatorData, error) {
	// retrieve the operator metadata from bundle image
	annotations, err := containerutil.GetAnnotationsFromBundle(bundleRef.ImageFSPath)

	if err != nil {
		log.Errorf("unable to get annotations.yaml from the bundle")
		return nil, err
	}

	catalogImage, err := viperutil.GetString(indexImageKey)
	if err != nil {
		log.Error(fmt.Sprintf("To set the key, export PFLT_%s or add %s:<value> to config.yaml in the current working directory", strings.ToUpper(indexImageKey), indexImageKey))
		return nil, err
	}

	channel, err := containerutil.Annotation(annotations, channelKey)
	if err != nil {
		log.Error("unable to extract channel name from ClusterServicVersion", err)
		return nil, err
	}

	packageName, err := containerutil.Annotation(annotations, packageKey)
	if err != nil {
		log.Error("unable to extract package name from ClusterServicVersion", err)
		return nil, err
	}

	return &OperatorData{
		CatalogImage:     catalogImage,
		Channel:          channel,
		PackageName:      packageName,
		App:              packageName,
		InstallNamespace: packageName,
	}, nil
}

func (p *DeployableByOlmCheck) setUp(operatorData OperatorData, openshiftengine openshiftengine.OpenshiftEngine) error {

	if _, err := openshiftengine.CreateNamespace(operatorData.InstallNamespace, cli.OpenshiftOptions{}); err != nil && !kubeErr.IsAlreadyExists(err) {
		return err
	}

	if _, err := openshiftengine.CreateCatalogSource(cli.CatalogSourceData{Name: operatorData.App, Image: operatorData.CatalogImage}, cli.OpenshiftOptions{Namespace: catalogSourceNS}); err != nil && !kubeErr.IsAlreadyExists(err) {
		return err
	}

	targetNamespaces := []string{operatorData.InstallNamespace}
	if _, err := openshiftengine.CreateOperatorGroup(cli.OperatorGroupData{Name: operatorData.App, TargetNamespaces: targetNamespaces}, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace}); err != nil && !kubeErr.IsAlreadyExists(err) {
		return err
	}

	subscriptionData := cli.SubscriptionData{
		Name:                   operatorData.App,
		Channel:                operatorData.Channel,
		CatalogSource:          operatorData.App,
		CatalogSourceNamespace: catalogSourceNS,
		Package:                operatorData.PackageName,
	}
	if _, err := openshiftengine.CreateSubscription(subscriptionData, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace}); err != nil && !kubeErr.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (p *DeployableByOlmCheck) isCSVReady(installedCSV string, operatorData OperatorData, openshiftengine openshiftengine.OpenshiftEngine) (bool, error) {

	log.Trace(fmt.Sprintf("Looking for csv %s in namespace %s", installedCSV, operatorData.InstallNamespace))

	// query API server for CSV by name
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	csvReadyDone := make(chan string, 1)

	go func() {
		defer close(csvReadyDone)
		for {
			log.Debug("Waiting for ClusterServiceVersion to become ready...")
			csv, _ := openshiftengine.GetCSV(installedCSV, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
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

func (p *DeployableByOlmCheck) installedCSV(operatorData OperatorData, openshiftengine openshiftengine.OpenshiftEngine) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	installedCSVDone := make(chan string, 1)

	// query API server for the installed CSV field of the created subscription
	go func() {
		defer close(installedCSVDone)
		for {
			time.Sleep(2 * time.Second)
			log.Debug("Waiting for Subscription.status.installedCSV to become ready...")
			subs, _ := openshiftengine.GetSubscription(operatorData.App, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
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

func (p *DeployableByOlmCheck) cleanUp(operatorData OperatorData, openshiftengine openshiftengine.OpenshiftEngine) {

	log.Debug("Dumping data in artifacts/ directory")

	subs, err := openshiftengine.GetSubscription(operatorData.App, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
	if err != nil {
		log.Error("unable to retrieve the subscription")
	}
	p.writeToFile(subs, operatorData.App, "subscription")

	cs, err := openshiftengine.GetCatalogSource(operatorData.App, cli.OpenshiftOptions{Namespace: catalogSourceNS})
	if err != nil {
		log.Error("unable to retrieve the catalogsource")
	}
	p.writeToFile(cs, operatorData.App, "catalogsource")

	og, err := openshiftengine.GetOperatorGroup(operatorData.App, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
	if err != nil {
		log.Error("unable to retrieve the operatorgroup")
	}
	p.writeToFile(og, operatorData.App, "operatorgroup")

	ns, err := openshiftengine.GetNamespace(operatorData.InstallNamespace)
	if err != nil {
		log.Error("unable to retrieve the namespace")
	}
	p.writeToFile(ns, operatorData.InstallNamespace, "namespace")

	log.Trace("Deleting the resources created by Check")
	openshiftengine.DeleteSubscription(operatorData.App, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
	openshiftengine.DeleteCatalogSource(operatorData.App, cli.OpenshiftOptions{Namespace: catalogSourceNS})
	openshiftengine.DeleteOperatorGroup(operatorData.App, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
	openshiftengine.DeleteNamespace(operatorData.InstallNamespace, cli.OpenshiftOptions{})
}

func (p *DeployableByOlmCheck) writeToFile(data interface{}, resource string, resourceType string) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		log.Error("unable to serialize the data")
	}

	if err = ioutil.WriteFile(filepath.Join("artifacts", fmt.Sprintf("%s-%s.yaml", resource, resourceType)), yamlData, 0644); err != nil {
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
