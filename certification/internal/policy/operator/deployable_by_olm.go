package operator

import (
	"context"
	"fmt"
	"strings"
	"time"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/viper"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	kubeErr "k8s.io/apimachinery/pkg/api/errors"
)

type OperatorData struct {
	CatalogImage     string
	Channel          string
	PackageName      string
	App              string
	InstallNamespace string
}

type DeployableByOlmCheck struct {
	OpenshiftEngine cli.OpenshiftEngine
}

var (
	subscriptionTimeout time.Duration = 180 * time.Second
	csvTimeout          time.Duration = 90 * time.Second
)

func NewDeployableByOlmCheck(openshiftEngine *cli.OpenshiftEngine) *DeployableByOlmCheck {
	return &DeployableByOlmCheck{
		OpenshiftEngine: *openshiftEngine,
	}
}

func (p *DeployableByOlmCheck) Validate(bundleRef certification.ImageReference) (bool, error) {
	// retrieve the required data
	operatorData, err := p.operatorMetadata(bundleRef)
	if err != nil {
		return false, err
	}

	// create k8s custom resources for the operator deployment
	err = p.setUp(*operatorData)
	defer p.cleanUp(*operatorData)

	if err != nil {
		return false, err
	}

	installedCSV, err := p.installedCSV(*operatorData)
	if err != nil {
		return false, err
	}

	return p.isCSVReady(installedCSV, *operatorData)
}

func (p *DeployableByOlmCheck) operatorMetadata(bundleRef certification.ImageReference) (*OperatorData, error) {
	// retrieve the operator metadata from bundle image
	annotations, err := getAnnotationsFromBundle(bundleRef.ImageFSPath)

	if err != nil {
		log.Errorf("unable to get annotations.yaml from the bundle")
		return nil, err
	}

	catalogImage := viper.GetString(indexImageKey)
	if len(catalogImage) == 0 {
		log.Error(fmt.Sprintf("To set the key, export PFLT_%s or add %s:<value> to config.yaml in the current working directory", strings.ToUpper(indexImageKey), indexImageKey))
		return nil, errors.ErrIndexImageUndefined
	}

	channel, err := annotation(annotations, channelKey)
	if err != nil {
		log.Error("unable to extract channel name from ClusterServicVersion", err)
		return nil, err
	}

	packageName, err := annotation(annotations, packageKey)
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

func (p *DeployableByOlmCheck) setUp(operatorData OperatorData) error {

	if _, err := p.OpenshiftEngine.CreateNamespace(operatorData.InstallNamespace, cli.OpenshiftOptions{}); err != nil && !kubeErr.IsAlreadyExists(err) {
		return err
	}

	if _, err := p.OpenshiftEngine.CreateCatalogSource(cli.CatalogSourceData{Name: operatorData.App, Image: operatorData.CatalogImage}, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace}); err != nil && !kubeErr.IsAlreadyExists(err) {
		return err
	}

	targetNamespaces := []string{operatorData.InstallNamespace}
	if _, err := p.OpenshiftEngine.CreateOperatorGroup(cli.OperatorGroupData{Name: operatorData.App, TargetNamespaces: targetNamespaces}, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace}); err != nil && !kubeErr.IsAlreadyExists(err) {
		return err
	}

	subscriptionData := cli.SubscriptionData{
		Name:                   operatorData.App,
		Channel:                operatorData.Channel,
		CatalogSource:          operatorData.App,
		CatalogSourceNamespace: operatorData.InstallNamespace,
		Package:                operatorData.PackageName,
	}
	if _, err := p.OpenshiftEngine.CreateSubscription(subscriptionData, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace}); err != nil && !kubeErr.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (p *DeployableByOlmCheck) isCSVReady(installedCSV string, operatorData OperatorData) (bool, error) {

	log.Trace(fmt.Sprintf("Looking for csv %s in namespace %s", installedCSV, operatorData.InstallNamespace))

	ctx := context.Background()

	csvReadyDone := make(chan string, 1)
	defer close(csvReadyDone)

	contextTimeOut := make(chan error, 1)
	defer close(contextTimeOut)

	go func(parent context.Context) {
		ctx, cancel := context.WithTimeout(parent, csvTimeout)
		defer cancel()

		for {
			log.Debug("Waiting for ClusterServiceVersion to become ready...")
			csv, _ := p.OpenshiftEngine.GetCSV(installedCSV, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
			// if the CSV phase is succeeded, stop the querying
			if csv.Status.Phase == operatorv1alpha1.CSVPhaseSucceeded {
				log.Debug("CSV is created successfully: ", installedCSV)
				csvReadyDone <- fmt.Sprintf("%#v", csv)
				return
			}
			log.Debug("CSV is not ready yet, retrying...")

			select {
			case <-ctx.Done():
				log.Error(fmt.Sprintf("failed to fetch the csv %s: ", installedCSV), ctx.Err())
				contextTimeOut <- ctx.Err()
				return
			default:
				time.Sleep(2 * time.Second)
			}
		}
	}(ctx)

	select {
	case csv := <-csvReadyDone:
		return len(csv) > 0, nil
	case err := <-contextTimeOut:
		return false, fmt.Errorf("%w: %s", errors.ErrK8sAPICallFailed, err)
	}

}

func (p *DeployableByOlmCheck) installedCSV(operatorData OperatorData) (string, error) {

	ctx := context.Background()

	installedCSVDone := make(chan string, 1)
	defer close(installedCSVDone)

	contextTimeOut := make(chan error, 1)
	defer close(contextTimeOut)

	// query API server for the installed CSV field of the created subscription
	go func(parent context.Context) {
		ctx, cancel := context.WithTimeout(parent, subscriptionTimeout)
		defer cancel()
		for {
			log.Debug("Waiting for Subscription.status.installedCSV to become ready...")
			subs, _ := p.OpenshiftEngine.GetSubscription(operatorData.App, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
			installedCSV := subs.Status.InstalledCSV
			// if the installedCSV field is present, stop the querying
			if len(installedCSV) > 0 {
				log.Debug(fmt.Sprintf("Subscription.status.installedCSV is %s", installedCSV))
				installedCSVDone <- installedCSV
				return
			}
			log.Debug("Subscription.status.installedCSV is not set yet, retrying...")

			select {
			case <-ctx.Done():
				log.Error("failed to fetch Subscription.status.installedCSV: ", ctx.Err())
				contextTimeOut <- ctx.Err()
				return
			default:
				time.Sleep(2 * time.Second)
			}
		}
	}(ctx)

	select {
	case installedCSV := <-installedCSVDone:
		return installedCSV, nil
	case err := <-contextTimeOut:
		return "", fmt.Errorf("%w: %s", errors.ErrK8sAPICallFailed, err)
	}
}

func (p *DeployableByOlmCheck) cleanUp(operatorData OperatorData) {

	log.Debug("Dumping data in artifacts/ directory")

	subs, err := p.OpenshiftEngine.GetSubscription(operatorData.App, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
	if err != nil {
		log.Error("unable to retrieve the subscription")
	}
	p.writeToFile(subs, operatorData.App, "subscription")

	cs, err := p.OpenshiftEngine.GetCatalogSource(operatorData.App, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
	if err != nil {
		log.Error("unable to retrieve the catalogsource")
	}
	p.writeToFile(cs, operatorData.App, "catalogsource")

	og, err := p.OpenshiftEngine.GetOperatorGroup(operatorData.App, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
	if err != nil {
		log.Error("unable to retrieve the operatorgroup")
	}
	p.writeToFile(og, operatorData.App, "operatorgroup")

	ns, err := p.OpenshiftEngine.GetNamespace(operatorData.InstallNamespace)
	if err != nil {
		log.Error("unable to retrieve the namespace")
	}
	p.writeToFile(ns, operatorData.InstallNamespace, "namespace")

	log.Trace("Deleting the resources created by Check")
	p.OpenshiftEngine.DeleteSubscription(operatorData.App, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
	p.OpenshiftEngine.DeleteCatalogSource(operatorData.App, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
	p.OpenshiftEngine.DeleteOperatorGroup(operatorData.App, cli.OpenshiftOptions{Namespace: operatorData.InstallNamespace})
	p.OpenshiftEngine.DeleteNamespace(operatorData.InstallNamespace, cli.OpenshiftOptions{})
}

func (p *DeployableByOlmCheck) writeToFile(data interface{}, resource string, resourceType string) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		log.Error("unable to serialize the data")
		return err
	}

	filename := fmt.Sprintf("%s-%s.yaml", resource, resourceType)
	if _, err := artifacts.WriteFile(filename, string(yamlData)); err != nil {
		log.Error("failed to write the k8s object to the file", err)
		return err
	}
	return nil
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
