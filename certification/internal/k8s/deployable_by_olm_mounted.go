package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	containerutil "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/utils/container"

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
	// catalogSourceNS is the namespace in which the CatalogSource CR is installed
	catalogSourceNS = "openshift-marketplace"

	// packageKey is the packageKey in annotations.yaml that contains the package name.
	packageKey = "operators.operatorframework.io.bundle.package.v1"

	// channelKey is the channel in annotations.yaml that contains the channel name.
	channelKey = "operators.operatorframework.io.bundle.channel.default.v1"

	// IndexImageKey is the key in viper that contains the index (catalog) image name
	IndexImageKey = "indexImage"
)

var (
	k8sconfig            *rest.Config
	err                  error
	packageName, app     string
	channel, ooNamespace string
	catalogImage         string
	targetNamespaces     []string
)

type DeployableByOlmMountedCheck struct{}

func (p *DeployableByOlmMountedCheck) Validate(bundleRef certification.ImageReference) (bool, error) {
	// create k8s custom resources for the operator deployment
	err = p.setUp(bundleRef.ImageURI)
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

func (p *DeployableByOlmMountedCheck) setUp(bundle string) error {
	kubeconfig := os.Getenv("KUBECONFIG")

	if len(kubeconfig) > 0 {
		k8sconfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return err
		}
	}

	// retrieve the operator metadata from bundle image
	data, err := containerutil.GetAnnotationsFromBundle(bundle)

	if err != nil {
		log.Errorf("unable to get annotations.yaml from the bundle")
		return err
	}

	catalogImage, err = viperutil.GetString(IndexImageKey)
	if err != nil {
		log.Error(fmt.Sprintf("To set the key, export PFLT_%s or add %s:<value> to config.yaml in the current working directory", strings.ToUpper(IndexImageKey), IndexImageKey))
		return err
	}

	channel, err = p.getValue(data, channelKey)
	if err != nil {
		log.Error("unable to extract channel name from ClusterServicVersion", err)
		return err
	}

	packageName, err = p.getValue(data, packageKey)
	if err != nil {
		log.Error("unable to extract package name from ClusterServicVersion", err)
		return err
	}
	app = packageName
	ooNamespace = packageName

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

func (p *DeployableByOlmMountedCheck) isCSVReady(installedCSV string) (bool, error) {
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

func (p *DeployableByOlmMountedCheck) installedCSV() (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
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

func (p *DeployableByOlmMountedCheck) cleanUp() {
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

// getPackageName accepts the annotations map and searches for the specified annotation corresponding
// with the complete bundle name for an operator, which is then returned.
func (p *DeployableByOlmMountedCheck) getValue(annotations map[string]string, key string) (string, error) {
	log.Tracef("searching for package key (%s) in bundle", packageKey)
	log.Trace("bundle data: ", annotations)
	value, found := annotations[key]
	if !found {
		return "", fmt.Errorf("did not find value at the key %s in the annotations.yaml", key)
	}

	return value, nil
}

func (p *DeployableByOlmMountedCheck) writeToFile(data interface{}, resource string, resourceType string) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		log.Error("unable to serialize the data")
	}
	err = os.WriteFile(filepath.Join("artifacts", fmt.Sprintf("%s-%s.yaml", resource, resourceType)), yamlData, 0644)
	if err != nil {
		log.Error("failed to write the k8s object to the file")
	}
}

func (p *DeployableByOlmMountedCheck) Name() string {
	return "DeployableByOLMMounted"
}

func (p *DeployableByOlmMountedCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the operator could be deployed by OLM",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide", // Placeholder
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *DeployableByOlmMountedCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "It is required that your operator could be deployed by OLM",
		Suggestion: "Follow the guidelines on the operatorsdk website to learn how to package your operator https://sdk.operatorframework.io/docs/olm-integration/cli-overview/",
	}
}
