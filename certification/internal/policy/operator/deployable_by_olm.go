package operator

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinery "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	kubeErr "k8s.io/apimachinery/pkg/api/errors"
)

type operatorData struct {
	CatalogImage     string
	Channel          string
	PackageName      string
	App              string
	InstallNamespace string
}

type DeployableByOlmCheck struct {
	csvReady    bool
	validImages bool
	K8sclient   client.Client
}

const (
	// catalogSourceNS is the namespace in which the CatalogSource CR is installed
	catalogSourceNS = "openshift-marketplace"
)

var (
	subscriptionTimeout time.Duration = 180 * time.Second
	csvTimeout          time.Duration = 90 * time.Second
	approvedRegistries                = map[string]struct{}{
		"registry.connect.dev.redhat.com":   {},
		"registry.connect.qa.redhat.com":    {},
		"registry.connect.stage.redhat.com": {},
		"registry.connect.redhat.com":       {},
		"registry.redhat.io":                {},
		"registry.access.redhat.com":        {},
	}
)

func NewDeployableByOlmCheck(client *client.Client) *DeployableByOlmCheck {
	if client == nil {
		log.Error("The client is nil. Returning a fake client")
		return &DeployableByOlmCheck{
			K8sclient: fakeclient.NewClientBuilder().Build(),
		}
	}
	return &DeployableByOlmCheck{
		K8sclient: *client,
	}
}

func (p *DeployableByOlmCheck) Validate(bundleRef certification.ImageReference) (bool, error) {
	// gather the list of registry and pod images
	beforeOperatorImages, err := p.getImages()
	if err != nil {
		return false, err
	}

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

	p.csvReady, err = p.isCSVReady(installedCSV, *operatorData)
	if err != nil {
		return false, err
	}

	afterOperatorImages, err := p.getImages()
	if err != nil {
		return false, err
	}

	operatorImages := diffImageList(beforeOperatorImages, afterOperatorImages)
	p.validImages = checkImageSource(operatorImages)

	return p.csvReady, nil
}

func diffImageList(before, after map[string]struct{}) []string {
	var operatorImages []string
	for image := range after {
		if _, ok := before[image]; !ok {
			operatorImages = append(operatorImages, image)
		}
	}
	return operatorImages
}

func checkImageSource(operatorImages []string) bool {
	allApproved := true
	for _, image := range operatorImages {
		userRegistry := strings.Split(image, "/")[0]
		if _, ok := approvedRegistries[userRegistry]; !ok {
			log.Warnf("Unapproved registry found: %s", image)
			allApproved = false
		}
	}
	return allApproved
}

func (p *DeployableByOlmCheck) operatorMetadata(bundleRef certification.ImageReference) (*operatorData, error) {
	// retrieve the operator metadata from bundle image
	annotations, err := getAnnotationsFromBundle(bundleRef.ImageFSPath)

	if err != nil {
		log.Errorf("unable to get annotations.yaml from the bundle")
		return nil, err
	}

	catalogImage := viper.GetString(indexImageKey)

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

	return &operatorData{
		CatalogImage:     catalogImage,
		Channel:          channel,
		PackageName:      packageName,
		App:              packageName,
		InstallNamespace: packageName,
	}, nil
}

func (p *DeployableByOlmCheck) setUp(operatorData operatorData) error {
	if err := p.createNamespace(context.Background(), operatorData.InstallNamespace); err != nil && !kubeErr.IsAlreadyExists(err) {
		return err
	}

	dockerconfig := viper.GetString("dockerConfig")
	if len(dockerconfig) != 0 {
		content, err := p.readFileAsByteArray(dockerconfig)
		if err != nil {
			return err
		}

		data := map[string]string{".dockerconfigjson": string(content)}

		if err := p.createSecret(context.Background(), secretName, data, corev1.SecretTypeDockerConfigJson, catalogSourceNS); err != nil && !kubeErr.IsAlreadyExists(err) {
			return err
		}
	} else {
		log.Debug("No docker config file is found to access the index image in private registries. Proceeding...")
	}

	if strings.Contains(operatorData.CatalogImage, "image-registry.openshift-image-registry.svc") {
		indexImageNamespace := strings.Split(operatorData.CatalogImage, "/")[1]

		if len(indexImageNamespace) != 0 {
			// create rolebindings for the pipeline service account
			if err := p.grantRegistryPermissionToServiceAccount(pipelineServiceAccount, operatorData.InstallNamespace,
				indexImageNamespace); err != nil {
				return err
			}
			// create rolebinding for the default OperatorHub catalog sources
			if err := p.grantRegistryPermissionToServiceAccount(operatorData.App, openshiftMarketplaceNamespace,
				indexImageNamespace); err != nil {
				return err
			}
			// create rolebindings for the custom catalog
			if err := p.grantRegistryPermissionToServiceAccount(operatorData.App, operatorData.InstallNamespace,
				indexImageNamespace); err != nil {
				return err
			}

		}
	}

	if err := p.createCatalogSource(context.Background(), operatorData.App, operatorData.CatalogImage, []string{secretName}, catalogSourceNS); err != nil && !kubeErr.IsAlreadyExists(err) {
		return err
	}

	targetNamespaces := []string{operatorData.InstallNamespace}
	if err := p.createOperatorGroup(context.Background(), operatorData.App, operatorData.InstallNamespace, targetNamespaces); err != nil && !kubeErr.IsAlreadyExists(err) {
		return err
	}

	if err := p.createSubscription(context.Background(), operatorData, operatorData.InstallNamespace); err != nil && !kubeErr.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (p *DeployableByOlmCheck) grantRegistryPermissionToServiceAccount(serviceAccount, serviceAccountNamespace, indexImageNamespace string) error {
	for _, role := range []string{registryViewerRole, imagePullerRole} {
		if err := p.createRoleBinding(context.Background(), serviceAccount, role, serviceAccountNamespace, indexImageNamespace); err != nil && !kubeErr.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

type watchFunc func(ctx context.Context, client client.Client, name, namespace string) (string, bool, error)

func watch(ctx context.Context, client client.Client, name, namespace string, timeout time.Duration, doneChannel chan string, timeoutChannel chan error, fn watchFunc) {
	ctx, cancel := context.WithTimeout(ctx, subscriptionTimeout)
	defer cancel()
	for {
		log.Debug("Waiting for object to become ready...")
		obj, done, err := fn(ctx, client, name, namespace)
		if err != nil {
			// Something bad happened. Get out of town
			log.Error(fmt.Sprintf("could not retrieve the object %s: ", name), err)
			timeoutChannel <- err
			return
		}
		if done {
			log.Debug(fmt.Sprintf("Object is %s", obj))
			doneChannel <- obj
			return
		}
		log.Debug("Object is not set yet, retrying...")

		select {
		case <-ctx.Done():
			log.Error("failed to fetch object: ", ctx.Err())
			timeoutChannel <- ctx.Err()
			return
		default:
			time.Sleep(2 * time.Second)
		}
	}
}

func csvStatusSucceeded(ctx context.Context, client client.Client, name, namespace string) (string, bool, error) {
	csv := &operatorv1alpha1.ClusterServiceVersion{}
	err := client.Get(ctx, apimachinery.NamespacedName{Namespace: namespace, Name: name}, csv)
	if err != nil && !kubeErr.IsNotFound(err) {
		// This is not a normal error. Get out of town
		log.Error(fmt.Sprintf("failed to fetch the csv %s: ", name), err)
		return "", false, err
	}
	// if the CSV phase is succeeded, stop the querying
	if csv.Status.Phase == operatorv1alpha1.CSVPhaseSucceeded {
		log.Debug("CSV is created successfully: ", name)
		return name, true, nil
	}

	return "", false, nil
}

func (p *DeployableByOlmCheck) isCSVReady(installedCSV string, operatorData operatorData) (bool, error) {
	log.Trace(fmt.Sprintf("Looking for csv %s in namespace %s", installedCSV, operatorData.InstallNamespace))

	ctx := context.Background()

	csvReadyDone := make(chan string, 1)
	defer close(csvReadyDone)

	contextTimeOut := make(chan error, 1)
	defer close(contextTimeOut)

	go watch(ctx, p.K8sclient, installedCSV, operatorData.InstallNamespace, csvTimeout, csvReadyDone, contextTimeOut, csvStatusSucceeded)

	select {
	case csv := <-csvReadyDone:
		return len(csv) > 0, nil
	case err := <-contextTimeOut:
		return false, fmt.Errorf("%w: %s", errors.ErrK8sAPICallFailed, err)
	}

}

func subscriptionCsvIsInstalled(ctx context.Context, client client.Client, name, namespace string) (string, bool, error) {
	sub := &operatorv1alpha1.Subscription{}
	err := client.Get(ctx, apimachinery.NamespacedName{Namespace: namespace, Name: name}, sub)
	if err != nil && !kubeErr.IsNotFound(err) {
		log.Error(fmt.Sprintf("failed to fetch the subscription %s: ", name), err)
		return "", false, err
	}
	installedCSV := sub.Status.InstalledCSV
	// if the installedCSV field is present, stop the querying
	if len(installedCSV) > 0 {
		return installedCSV, true, nil
	}

	return "", false, nil
}

func (p *DeployableByOlmCheck) installedCSV(operatorData operatorData) (string, error) {
	ctx := context.Background()

	installedCSVDone := make(chan string, 1)
	defer close(installedCSVDone)

	contextTimeout := make(chan error, 1)
	defer close(contextTimeout)

	// query API server for the installed CSV field of the created subscription
	go watch(ctx, p.K8sclient, operatorData.App, operatorData.InstallNamespace, subscriptionTimeout, installedCSVDone, contextTimeout, subscriptionCsvIsInstalled)

	select {
	case installedCSV := <-installedCSVDone:
		return installedCSV, nil
	case err := <-contextTimeout:
		return "", fmt.Errorf("%w: %s", errors.ErrK8sAPICallFailed, err)
	}
}

func (p *DeployableByOlmCheck) cleanUp(operatorData operatorData) {
	log.Debug("Dumping data in artifacts/ directory")

	ctx := context.TODO()

	subs, err := p.getSubscription(ctx, operatorData.App, operatorData.InstallNamespace)
	if err != nil {
		log.Error("unable to retrieve the subscription")
	}
	p.writeToFile(subs, operatorData.App, "subscription")
	defer p.deleteSubscription(ctx, subs)

	cs, err := p.getCatalogSource(ctx, operatorData.App, catalogSourceNS)
	if err != nil {
		log.Error("unable to retrieve the catalogsource")
	}
	p.writeToFile(cs, operatorData.App, "catalogsource")
	defer p.deleteCatalogSource(ctx, cs)

	og, err := p.getOperatorGroup(ctx, operatorData.App, operatorData.InstallNamespace)
	if err != nil {
		log.Error("unable to retrieve the operatorgroup")
	}
	p.writeToFile(og, operatorData.App, "operatorgroup")
	defer p.deleteOperatorGroup(ctx, og)

	secret, err := p.getSecret(ctx, secretName, operatorData.InstallNamespace)
	if err != nil {
		log.Error("unable to retrieve secret")
	}
	defer p.deleteSecret(ctx, secret)

	if strings.Contains(operatorData.CatalogImage, "image-registry.openshift-image-registry.svc") {
		indexImageNamespace := strings.Split(operatorData.CatalogImage, "/")[1]
		operatorServiceAccount := operatorData.App
		operatorNamespace := operatorData.InstallNamespace

		// remove pipeline-related rolebindings
		rb, err := p.getRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", pipelineServiceAccount, operatorNamespace, registryViewerRole), indexImageNamespace)
		if err != nil {
			log.Error("unable to retrieve rolebinding")
		}
		defer p.deleteRoleBinding(ctx, rb)

		rb, err = p.getRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", pipelineServiceAccount, operatorNamespace, imagePullerRole), indexImageNamespace)
		if err != nil {
			log.Error("unable to retrieve rolebinding")
		}
		defer p.deleteRoleBinding(ctx, rb)

		// remove rolebindings required for the default OperatorHub catalog sources
		p.getRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", operatorServiceAccount, openshiftMarketplaceNamespace, registryViewerRole), indexImageNamespace)
		// p.deleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", operatorServiceAccount, openshiftMarketplaceNamespace, imagePullerRole), indexImageNamespace)
		// //remove rolebindings required for custom catalog sources
		// p.deleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", operatorServiceAccount, operatorNamespace, registryViewerRole), indexImageNamespace)
		// p.deleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", operatorServiceAccount, operatorNamespace, imagePullerRole), indexImageNamespace)
	}

	ns, err := p.getNamespace(ctx, operatorData.InstallNamespace)
	if err != nil {
		log.Error("unable to retrieve the namespace")
	}
	p.writeToFile(ns, operatorData.InstallNamespace, "namespace")
	defer p.deleteNamespace(ctx, ns)
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

func (p *DeployableByOlmCheck) readFileAsByteArray(filename string) ([]byte, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		log.Error(fmt.Sprintf("error reading the file: %s", filename))
		return nil, err
	}
	return content, nil
}

func (p *DeployableByOlmCheck) Name() string {
	return "DeployableByOLM"
}

func (p *DeployableByOlmCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the operator could be deployed by OLM, and images are from approved sources",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide", // Placeholder
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *DeployableByOlmCheck) Help() certification.HelpText {
	if !p.validImages {
		return certification.HelpText{
			Message:    "It is required that your operator contains images from valid sources",
			Suggestion: "Images should only be sourced from approved registries",
		}
	}
	return certification.HelpText{
		Message:    "It is required that your operator could be deployed by OLM",
		Suggestion: "Follow the guidelines on the operatorsdk website to learn how to package your operator https://sdk.operatorframework.io/docs/olm-integration/cli-overview/",
	}
}

func (p *DeployableByOlmCheck) createNamespace(ctx context.Context, namespace string) error {
	nsSpec := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	return p.K8sclient.Create(ctx, nsSpec)
}

func (p *DeployableByOlmCheck) createCatalogSource(ctx context.Context, name, image string, secrets []string, namespace string) error {
	catalogSpec := &operatorv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType:  operatorv1alpha1.SourceTypeGrpc,
			Image:       image,
			DisplayName: name,
			Secrets:     secrets,
		},
	}

	return p.K8sclient.Create(ctx, catalogSpec)
}

func (p *DeployableByOlmCheck) createOperatorGroup(ctx context.Context, name, namespace string, targetNamespaces []string) error {
	operatorGroupSpec := &operatorv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: operatorv1.OperatorGroupSpec{
			TargetNamespaces: targetNamespaces,
		},
	}
	return p.K8sclient.Create(ctx, operatorGroupSpec)
}

func (p *DeployableByOlmCheck) createSubscription(ctx context.Context, operatorData operatorData, namespace string) error {
	subscriptionSpec := &operatorv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorData.App,
			Namespace: namespace,
		},
		Spec: &operatorv1alpha1.SubscriptionSpec{
			CatalogSource:          operatorData.App,
			CatalogSourceNamespace: catalogSourceNS,
			Channel:                operatorData.Channel,
			Package:                operatorData.PackageName,
		},
	}

	return p.K8sclient.Create(ctx, subscriptionSpec)
}

func (p *DeployableByOlmCheck) createSecret(ctx context.Context, secretName string, content map[string]string, secretType corev1.SecretType, namespace string) error {
	secretSpec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		StringData: content,
		Type:       secretType,
	}
	return p.K8sclient.Create(ctx, secretSpec)
}

func (p *DeployableByOlmCheck) createRoleBinding(ctx context.Context, serviceAccount, role, serviceAccountNamespace, namespace string) error {
	subject := rbacv1.Subject{Kind: "ServiceAccount", Name: serviceAccount, Namespace: serviceAccountNamespace}
	rbacSpec := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s:%s:%s", serviceAccount, serviceAccountNamespace, role),
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{subject},
		RoleRef:  rbacv1.RoleRef{Kind: "ClusterRole", APIGroup: "rbac.authorization.k8s.io", Name: role},
	}

	return p.K8sclient.Create(ctx, rbacSpec)
}

func (p *DeployableByOlmCheck) getSubscription(ctx context.Context, name, namespace string) (*operatorv1alpha1.Subscription, error) {
	subscription := new(operatorv1alpha1.Subscription)
	err := p.K8sclient.Get(ctx, apimachinery.NamespacedName{Namespace: namespace, Name: name}, subscription)
	return subscription, err
}

func (p *DeployableByOlmCheck) getCatalogSource(ctx context.Context, name, namespace string) (*operatorv1alpha1.CatalogSource, error) {
	catalogSource := new(operatorv1alpha1.CatalogSource)
	err := p.K8sclient.Get(ctx, apimachinery.NamespacedName{Namespace: namespace, Name: name}, catalogSource)
	return catalogSource, err
}

func (p *DeployableByOlmCheck) getOperatorGroup(ctx context.Context, name, namespace string) (*operatorv1.OperatorGroup, error) {
	operatorGroup := new(operatorv1.OperatorGroup)
	err := p.K8sclient.Get(ctx, apimachinery.NamespacedName{Namespace: namespace, Name: name}, operatorGroup)
	return operatorGroup, err
}

func (p *DeployableByOlmCheck) getNamespace(ctx context.Context, namespace string) (*corev1.Namespace, error) {
	ns := new(corev1.Namespace)
	err := p.K8sclient.Get(ctx, apimachinery.NamespacedName{Name: namespace}, ns)
	return ns, err
}

func (p *DeployableByOlmCheck) getSecret(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	secret := new(corev1.Secret)
	err := p.K8sclient.Get(ctx, apimachinery.NamespacedName{Namespace: namespace, Name: name}, secret)
	return secret, err
}

func (p *DeployableByOlmCheck) getRoleBinding(ctx context.Context, name, namespace string) (*rbacv1.RoleBinding, error) {
	rb := new(rbacv1.RoleBinding)
	err := p.K8sclient.Get(ctx, apimachinery.NamespacedName{Namespace: namespace, Name: name}, rb)
	return rb, err
}

func (p *DeployableByOlmCheck) getImages() (map[string]struct{}, error) {
	return nil, nil
}

func (p *DeployableByOlmCheck) deleteSubscription(ctx context.Context, subscription *operatorv1alpha1.Subscription) error {
	deletePolicy := metav1.DeletePropagationForeground
	return p.K8sclient.Delete(ctx, subscription, &client.DeleteOptions{PropagationPolicy: &deletePolicy})
}

func (p *DeployableByOlmCheck) deleteCatalogSource(ctx context.Context, catalogSource *operatorv1alpha1.CatalogSource) error {
	deletePolicy := metav1.DeletePropagationForeground
	return p.K8sclient.Delete(ctx, catalogSource, &client.DeleteOptions{PropagationPolicy: &deletePolicy})
}

func (p *DeployableByOlmCheck) deleteOperatorGroup(ctx context.Context, operatorGroup *operatorv1.OperatorGroup) error {
	deletePolicy := metav1.DeletePropagationForeground
	return p.K8sclient.Delete(ctx, operatorGroup, &client.DeleteOptions{PropagationPolicy: &deletePolicy})
}

func (p *DeployableByOlmCheck) deleteNamespace(ctx context.Context, namespace *corev1.Namespace) error {
	deletePolicy := metav1.DeletePropagationForeground
	return p.K8sclient.Delete(ctx, namespace, &client.DeleteOptions{PropagationPolicy: &deletePolicy})
}

func (p *DeployableByOlmCheck) deleteSecret(ctx context.Context, secret *corev1.Secret) error {
	deletePolicy := metav1.DeletePropagationForeground
	return p.K8sclient.Delete(ctx, secret, &client.DeleteOptions{PropagationPolicy: &deletePolicy})
}

func (p *DeployableByOlmCheck) deleteRoleBinding(ctx context.Context, roleBinding *rbacv1.RoleBinding) error {
	deletePolicy := metav1.DeletePropagationForeground
	return p.K8sclient.Delete(ctx, roleBinding, &client.DeleteOptions{PropagationPolicy: &deletePolicy})
}
