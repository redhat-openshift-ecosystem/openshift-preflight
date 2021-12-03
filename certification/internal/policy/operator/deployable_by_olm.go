package operator

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/artifacts"
	pflterr "github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"

	log "github.com/sirupsen/logrus"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
)

type OperatorData struct {
	CatalogImage     string
	Channel          string
	PackageName      string
	App              string
	InstallNamespace string
	TargetNamespace  string
	InstallModes     map[string]bool
	CsvNamespaces    []string
	InstalledCsv     string
}

type DeployableByOlmCheck struct {
	OpenshiftEngine cli.OpenshiftEngine
	csvReady        bool
	validImages     bool
}

func NewDeployableByOlmCheck(openshiftEngine *cli.OpenshiftEngine) *DeployableByOlmCheck {
	return &DeployableByOlmCheck{
		OpenshiftEngine: *openshiftEngine,
	}
}

func (p *DeployableByOlmCheck) Validate(bundleRef certification.ImageReference) (bool, error) {
	ctx := context.Background()

	// gather the list of registry and pod images
	beforeOperatorImages, err := p.getImages(ctx)
	if err != nil {
		return false, err
	}

	// retrieve the required data
	operatorData, err := p.operatorMetadata(bundleRef)
	if err != nil {
		return false, err
	}

	log.Debug("The operator Metadata is: ", fmt.Sprintf("%+v", *operatorData))

	// create k8s custom resources for the operator deployment
	err = p.setUp(ctx, operatorData)
	defer p.cleanUp(ctx, *operatorData)

	if err != nil {
		return false, err
	}

	installedCSV, err := p.installedCSV(ctx, *operatorData)
	if err != nil {
		return false, err
	}
	operatorData.InstalledCsv = installedCSV
	log.Trace("the installed CSV is ", operatorData.InstalledCsv)

	p.csvReady, err = p.isCSVReady(ctx, *operatorData)
	if err != nil {
		return false, err
	}

	afterOperatorImages, err := p.getImages(ctx)
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
	log.Info("Checking that images are from approved sources...")

	registries := make([]string, len(approvedRegistries))
	i := 0
	for registry := range approvedRegistries {
		registries[i] = registry
		i++
	}

	log.Debug("List of approved registries are: ", registries)
	allApproved := true
	for _, image := range operatorImages {
		userRegistry := strings.Split(image, "/")[0]
		if _, ok := approvedRegistries[userRegistry]; !ok {
			log.Warnf("Unapproved registry found for image %s", image)
			allApproved = false
		}
	}
	if allApproved {
		log.Info("All images are from approved sources...")
	}
	return allApproved
}

func (p *DeployableByOlmCheck) operatorMetadata(bundleRef certification.ImageReference) (*OperatorData, error) {
	// retrieve the operator metadata from bundle image
	annotations, err := getAnnotationsFromBundle(bundleRef.ImageFSPath)

	if err != nil {
		log.Errorf("unable to get annotations.yaml from the bundle")
		return nil, err
	}

	catalogImage := viper.GetString(indexImageKey)

	channel, err := annotation(annotations, channelKeyInBundle)
	if err != nil {
		log.Error("unable to extract channel name from the bundle: ", err)
		return nil, err
	}

	if len(viper.GetString(channelKey)) != 0 {
		channel = viper.GetString(channelKey)
	}

	packageName, err := annotation(annotations, packageKey)
	if err != nil {
		log.Error("unable to extract package name from the bundle: ", err)
		return nil, err
	}

	installedModes, err := getSupportedInstalledModes(bundleRef.ImageFSPath)
	if err != nil {
		log.Error("unable to extract operator install modes from ClusterServicVersion: ", err)
		return nil, err
	}

	return &OperatorData{
		CatalogImage:     catalogImage,
		Channel:          channel,
		PackageName:      packageName,
		App:              packageName,
		InstallNamespace: packageName,
		TargetNamespace:  packageName + "-target",
		InstallModes:     installedModes,
	}, nil
}

func (p *DeployableByOlmCheck) setUp(ctx context.Context, operatorData *OperatorData) error {
	if _, err := p.OpenshiftEngine.CreateNamespace(ctx, operatorData.InstallNamespace); err != nil && !k8serr.IsAlreadyExists(err) {
		return err
	}

	if _, err := p.OpenshiftEngine.CreateNamespace(ctx, operatorData.TargetNamespace); err != nil && !k8serr.IsAlreadyExists(err) {
		return err
	}

	dockerconfig := viper.GetString("dockerConfig")
	if len(dockerconfig) != 0 {
		content, err := p.readFileAsByteArray(dockerconfig)
		if err != nil {
			return err
		}
		data := map[string]string{".dockerconfigjson": string(content)}
		if _, err := p.OpenshiftEngine.CreateSecret(ctx, secretName, data, corev1.SecretTypeDockerConfigJson, operatorData.InstallNamespace); err != nil && !k8serr.IsAlreadyExists(err) {
			return err
		}
	} else {
		log.Debug("No docker config file is found to access the index image in private registries. Proceeding...")
	}

	if strings.Contains(operatorData.CatalogImage, imageRegistryService) {
		indexImageNamespace := strings.Split(operatorData.CatalogImage, "/")[1]
		if len(indexImageNamespace) != 0 {
			// create rolebindings for the pipeline service account
			if err := p.grantRegistryPermissionToServiceAccount(ctx, pipelineServiceAccount, operatorData.InstallNamespace,
				indexImageNamespace); err != nil {
				return err
			}
			// create rolebinding for the default OperatorHub catalog sources
			if err := p.grantRegistryPermissionToServiceAccount(ctx, operatorData.App, openshiftMarketplaceNamespace,
				indexImageNamespace); err != nil {
				return err
			}
			// create rolebindings for the custom catalog
			if err := p.grantRegistryPermissionToServiceAccount(ctx, operatorData.App, operatorData.InstallNamespace,
				indexImageNamespace); err != nil {
				return err
			}

		}
	}

	if _, err := p.OpenshiftEngine.CreateCatalogSource(ctx, cli.CatalogSourceData{Name: operatorData.App, Image: operatorData.CatalogImage, Secrets: []string{secretName}}, operatorData.InstallNamespace); err != nil && !k8serr.IsAlreadyExists(err) {
		return err
	}

	operatorGroupData, err := p.generateOperatorGroupData(operatorData)
	if err != nil {
		return err
	}
	if _, err := p.OpenshiftEngine.CreateOperatorGroup(ctx, operatorGroupData, operatorData.InstallNamespace); err != nil && !k8serr.IsAlreadyExists(err) {
		return err
	}

	subscriptionData := cli.SubscriptionData{
		Name:                   operatorData.App,
		Channel:                operatorData.Channel,
		CatalogSource:          operatorData.App,
		CatalogSourceNamespace: operatorData.InstallNamespace,
		Package:                operatorData.PackageName,
	}
	if _, err := p.OpenshiftEngine.CreateSubscription(ctx, subscriptionData, operatorData.InstallNamespace); err != nil && !k8serr.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (p *DeployableByOlmCheck) generateOperatorGroupData(operatorData *OperatorData) (cli.OperatorGroupData, error) {
	var installMode string
	for i := 0; i < len(prioritizedInstallModes); i++ {
		if _, ok := operatorData.InstallModes[prioritizedInstallModes[i]]; ok {
			installMode = prioritizedInstallModes[i]
			break
		}
	}
	log.Debug(fmt.Sprintf("The operator install mode is %s", installMode))
	targetNamespaces := make([]string, 2)

	switch installMode {
	case string(operatorv1alpha1.InstallModeTypeOwnNamespace):
		targetNamespaces = []string{operatorData.InstallNamespace}
	case string(operatorv1alpha1.InstallModeTypeSingleNamespace):
		targetNamespaces = []string{operatorData.TargetNamespace}
	case string(operatorv1alpha1.InstallModeTypeMultiNamespace):
		targetNamespaces = []string{operatorData.TargetNamespace, operatorData.InstallNamespace}
	case string(operatorv1alpha1.InstallModeTypeAllNamespaces):
		targetNamespaces = []string{}

	}
	log.Debug(fmt.Sprintf("The OperatorGroup's TargetNamespaces is %s", targetNamespaces))
	operatorData.CsvNamespaces = targetNamespaces
	return cli.OperatorGroupData{Name: operatorData.App, TargetNamespaces: targetNamespaces}, nil
}

func (p *DeployableByOlmCheck) grantRegistryPermissionToServiceAccount(ctx context.Context, serviceAccount, serviceAccountNamespace, indexImageNamespace string) error {
	for _, role := range []string{registryViewerRole, imagePullerRole} {
		roleBindingData := cli.RoleBindingData{
			Name:      fmt.Sprintf("%s:%s:%s", serviceAccount, serviceAccountNamespace, role),
			Subjects:  []string{serviceAccount},
			Role:      role,
			Namespace: serviceAccountNamespace,
		}
		if _, err := p.OpenshiftEngine.CreateRoleBinding(ctx, roleBindingData, indexImageNamespace); err != nil && !k8serr.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

type watchFunc func(ctx context.Context, engine cli.OpenshiftEngine, name, namespace string) (string, bool, error)

func watch(ctx context.Context, engine cli.OpenshiftEngine, wg *sync.WaitGroup, name, namespace string, timeout time.Duration, channel chan string, fn watchFunc) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		log.Debugf("Waiting for object %s/%s to become ready...", namespace, name)
		obj, done, err := fn(ctx, engine, name, namespace)
		if err != nil {
			// Something bad happened. Get out of town
			log.Error(fmt.Sprintf("could not retrieve the object %s/%s: ", namespace, name), err)
			channel <- fmt.Sprintf("%s %v", errorPrefix, err)
			return
		}
		if done {
			log.Info(fmt.Sprintf("Successfully retrieved object %s/%s", namespace, obj))
			channel <- obj
			return
		}
		log.Debugf("Object %s/%s is not set yet, retrying...", namespace, name)

		select {
		case <-ctx.Done():
			log.Error(fmt.Sprintf("failed to retrieve object %s/%s: ", namespace, name), ctx.Err())
			channel <- fmt.Sprintf("%s %v", errorPrefix, ctx.Err())
			return
		default:
			time.Sleep(2 * time.Second)
		}
	}
}

func csvStatusSucceeded(ctx context.Context, engine cli.OpenshiftEngine, name, namespace string) (string, bool, error) {
	csv, err := engine.GetCSV(ctx, name, namespace)
	if err != nil && !k8serr.IsNotFound(err) {
		// This is not a normal error. Get out of town
		log.Error(fmt.Sprintf("failed to fetch the csv %s from namespace %s: ", name, namespace), err)
		return "", false, err
	}
	// if the CSV phase is succeeded, stop the querying
	if csv.Status.Phase == operatorv1alpha1.CSVPhaseSucceeded {
		log.Debug(fmt.Sprintf("CSV %s is created successfully in namespace %s", name, namespace))
		return name, true, nil
	}
	return "", false, nil
}

func (p *DeployableByOlmCheck) isCSVReady(ctx context.Context, operatorData OperatorData) (bool, error) {
	var CsvNamespaces []string
	if len(operatorData.CsvNamespaces) == 0 {
		CsvNamespaces = []string{operatorData.TargetNamespace, "default", openshiftMarketplaceNamespace}
	} else {
		CsvNamespaces = []string{operatorData.CsvNamespaces[0]}
	}
	log.Trace(fmt.Sprintf("Looking for csv %s in namespace(s) %s", operatorData.InstalledCsv, CsvNamespaces))

	csvChannel := make(chan string)

	var wg sync.WaitGroup
	go func() {
		wg.Wait()
		close(csvChannel)
	}()

	for _, CsvNamespace := range CsvNamespaces {
		wg.Add(1)
		go watch(ctx, p.OpenshiftEngine, &wg, operatorData.InstalledCsv, CsvNamespace, csvTimeout, csvChannel, csvStatusSucceeded)
	}

	for msg := range csvChannel {
		if strings.Contains(msg, errorPrefix) {
			return false, fmt.Errorf("%w: %s", pflterr.ErrK8sAPICallFailed, msg)
		}
		if len(msg) == 0 {
			return false, nil
		}
	}
	return true, nil
}

func subscriptionCsvIsInstalled(ctx context.Context, engine cli.OpenshiftEngine, name, namespace string) (string, bool, error) {
	sub, err := engine.GetSubscription(ctx, name, namespace)
	if err != nil && !k8serr.IsNotFound(err) {
		log.Error(fmt.Sprintf("failed to fetch the subscription %s from namespace %s: ", name, namespace), err)
		return "", false, err
	}
	log.Trace("current subscription status is: ", sub.Status)
	installedCSV := sub.Status.InstalledCSV
	// if the installedCSV field is present, stop the querying
	if len(installedCSV) > 0 {
		return installedCSV, true, nil
	}
	return "", false, nil
}

func (p *DeployableByOlmCheck) installedCSV(ctx context.Context, operatorData OperatorData) (string, error) {
	installedCSVChannel := make(chan string)

	var wg sync.WaitGroup
	go func() {
		wg.Wait()
		close(installedCSVChannel)
	}()
	// query API server for the installed CSV field of the created subscription
	wg.Add(1)
	go watch(ctx, p.OpenshiftEngine, &wg, operatorData.App, operatorData.InstallNamespace, subscriptionTimeout, installedCSVChannel, subscriptionCsvIsInstalled)

	installedCsv := ""
	for msg := range installedCSVChannel {
		if strings.Contains(msg, errorPrefix) {
			return "", fmt.Errorf("%w: %s", pflterr.ErrK8sAPICallFailed, msg)
		}
		installedCsv = msg
	}

	return installedCsv, nil
}

func (p *DeployableByOlmCheck) cleanUp(ctx context.Context, operatorData OperatorData) {

	log.Debug("Dumping data in artifacts/ directory")

	subs, err := p.OpenshiftEngine.GetSubscription(ctx, operatorData.App, operatorData.InstallNamespace)
	if err != nil {
		log.Error("unable to retrieve the subscription")
	} else {
		p.writeToFile(subs)
	}

	cs, err := p.OpenshiftEngine.GetCatalogSource(ctx, operatorData.App, operatorData.InstallNamespace)
	if err != nil {
		log.Error("unable to retrieve the catalogsource")
	} else {
		p.writeToFile(cs)
	}

	og, err := p.OpenshiftEngine.GetOperatorGroup(ctx, operatorData.App, operatorData.InstallNamespace)
	if err != nil {
		log.Error("unable to retrieve the operatorgroup")
	} else {
		p.writeToFile(og)
	}

	installNamespace, err := p.OpenshiftEngine.GetNamespace(ctx, operatorData.InstallNamespace)
	if err != nil {
		log.Error("unable to retrieve the install namespace")
	} else {
		p.writeToFile(installNamespace)
	}

	targetNamespace, err := p.OpenshiftEngine.GetNamespace(ctx, operatorData.TargetNamespace)
	if err != nil {
		log.Error("unable to retrieve the target namespace")
	} else {
		p.writeToFile(targetNamespace)
	}

	log.Trace("Deleting the resources created by Check")
	p.OpenshiftEngine.DeleteSubscription(ctx, operatorData.App, operatorData.InstallNamespace)
	p.OpenshiftEngine.DeleteCatalogSource(ctx, operatorData.App, operatorData.InstallNamespace)
	p.OpenshiftEngine.DeleteOperatorGroup(ctx, operatorData.App, operatorData.InstallNamespace)
	p.OpenshiftEngine.DeleteSecret(ctx, secretName, operatorData.InstallNamespace)

	if strings.Contains(operatorData.CatalogImage, imageRegistryService) {
		indexImageNamespace := strings.Split(operatorData.CatalogImage, "/")[1]
		operatorServiceAccount := operatorData.App
		operatorNamespace := operatorData.InstallNamespace
		// remove pipeline-related rolebindings
		p.OpenshiftEngine.DeleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", pipelineServiceAccount, operatorNamespace, registryViewerRole), indexImageNamespace)
		p.OpenshiftEngine.DeleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", pipelineServiceAccount, operatorNamespace, imagePullerRole), indexImageNamespace)
		// remove rolebindings required for the default OperatorHub catalog sources
		p.OpenshiftEngine.DeleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", operatorServiceAccount, openshiftMarketplaceNamespace, registryViewerRole), indexImageNamespace)
		p.OpenshiftEngine.DeleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", operatorServiceAccount, openshiftMarketplaceNamespace, imagePullerRole), indexImageNamespace)
		//remove rolebindings required for custom catalog sources
		p.OpenshiftEngine.DeleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", operatorServiceAccount, operatorNamespace, registryViewerRole), indexImageNamespace)
		p.OpenshiftEngine.DeleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", operatorServiceAccount, operatorNamespace, imagePullerRole), indexImageNamespace)
	}
	p.OpenshiftEngine.DeleteNamespace(ctx, operatorData.InstallNamespace)
	p.OpenshiftEngine.DeleteNamespace(ctx, operatorData.TargetNamespace)
}

func (p *DeployableByOlmCheck) writeToFile(data interface{}) error {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(data)
	if err != nil {
		log.Error("unable to convert the object to unstructured.Unstructured: ", err)
		return err
	}

	u := &unstructured.Unstructured{Object: obj}
	switch data.(type) {
	case *operatorv1alpha1.CatalogSource:
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "operators.coreos.com",
			Kind:    "CatalogSource",
			Version: "v1alpha1",
		})
	case *operatorv1.OperatorGroup:
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "operators.coreos.com",
			Kind:    "OperatorGroup",
			Version: "v1",
		})
	case *operatorv1alpha1.Subscription:
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "operators.coreos.com",
			Kind:    "Subscription",
			Version: "v1alpha1",
		})
	case *corev1.Namespace:
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "",
			Kind:    "Namespace",
			Version: "v1",
		})
	default:
		return pflterr.ErrUnsupportedGoType
	}

	jsonManifest, err := u.MarshalJSON()
	if err != nil {
		log.Error("unable to marshal to json: ", err)
		return err
	}

	filename := fmt.Sprintf("%s-%s.json", u.GetName(), u.GetKind())
	if _, err := artifacts.WriteFile(filename, string(jsonManifest)); err != nil {
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

func (p *DeployableByOlmCheck) getImages(ctx context.Context) (map[string]struct{}, error) {
	return p.OpenshiftEngine.GetImages(ctx)
}

func (p *DeployableByOlmCheck) Name() string {
	return "DeployableByOLM"
}

func (p *DeployableByOlmCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the operator could be deployed by OLM",
		Level:            "best",
		KnowledgeBaseURL: "https://sdk.operatorframework.io/docs/olm-integration/testing-deployment/",
		CheckURL:         "https://sdk.operatorframework.io/docs/olm-integration/testing-deployment/",
	}
}

func (p *DeployableByOlmCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "It is required that your operator could be deployed by OLM",
		Suggestion: "Follow the guidelines on the operatorsdk website to learn how to package your operator https://sdk.operatorframework.io/docs/olm-integration/cli-overview/",
	}
}
