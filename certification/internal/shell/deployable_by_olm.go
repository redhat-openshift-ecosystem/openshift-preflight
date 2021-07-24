package shell

import (
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/cli"
)

const (
	ooNamespace      = "olm"
	operatorGroup    = "og-single"
	catalogSource    = "operatorhubio-catalog"
	catalogSourceNs  = "openshift-marketplace"
	subscriptionName = "my-operator"
	packageName      = "etcd"
	channel          = "stable"
	catalogImage     = "quay.io/operatorhubio/catalog:latest"
)

var targetNamespaces []string = []string{"default"}

type DeployableByOlmCheck struct{}

func (p *DeployableByOlmCheck) Validate(image string) (bool, error) {
	openshiftEngine.CreateNamespace(ooNamespace, cli.OpenShiftCliOptions{})
	openshiftEngine.CreateOperatorGroup(cli.OperatorGroupData{Name: operatorGroup, TargetNamespaces: targetNamespaces}, cli.OpenShiftCliOptions{Namespace: ooNamespace})
	openshiftEngine.CreateCatalogSource(cli.CatalogSourceData{Name: catalogSource, Image: catalogImage}, cli.OpenShiftCliOptions{Namespace: ooNamespace})
	subscriptionData := cli.SubscriptionData{
		Name:                   subscriptionName,
		Channel:                channel,
		CatalogSource:          catalogSource,
		CatalogSourceNamespace: catalogSourceNs,
		Package:                packageName,
	}
	openshiftEngine.CreateSubscription(subscriptionData, cli.OpenShiftCliOptions{Namespace: ooNamespace})

	// openshiftEngine.DeleteSubscription(subscriptionName, cli.OpenShiftCliOptions{})
	// openshiftEngine.DeleteCatalogSource(catalogSource, cli.OpenShiftCliOptions{})
	// openshiftEngine.DeleteOperatorGroup(operatorGroup, cli.OpenShiftCliOptions{})
	// openshiftEngine.DeleteNamespace(ooNamespace, cli.OpenShiftCliOptions{})
	return true, nil
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
