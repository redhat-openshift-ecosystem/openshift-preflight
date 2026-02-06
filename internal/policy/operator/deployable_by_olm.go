package operator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/bundle"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/log"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/openshift"

	"github.com/go-logr/logr"
	"github.com/operator-framework/api/pkg/manifests"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	validationerrors "github.com/operator-framework/api/pkg/validation/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Option func(*DeployableByOlmCheck)

var _ check.Check = &DeployableByOlmCheck{}

type operatorData struct {
	CatalogImage     string
	Channel          string
	PackageName      string
	App              string
	InstallNamespace string
	TargetNamespace  string
	InstallModes     map[operatorsv1alpha1.InstallModeType]operatorsv1alpha1.InstallMode
	CsvNamespaces    []string
	InstalledCsv     string
	DeploymentNames  []string
}

type DeployableByOlmCheck struct {
	// dockerConfig is optional. If empty, we will not use one.
	dockerConfig string
	// indexImage is the catalog containing the operator bundle.
	indexImage string
	// channel is optional. If empty, we will introspect.
	channel string

	openshiftClient     openshift.Client
	client              crclient.Client
	k8sClientset        kubernetes.Interface
	csvReady            bool
	validImages         bool
	csvTimeout          time.Duration
	subscriptionTimeout time.Duration
}

func (p *DeployableByOlmCheck) initClient() error {
	if p.client != nil {
		return nil
	}
	scheme := apiruntime.NewScheme()
	if err := appsv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("could not add appsv1 scheme to scheme: %w", err)
	}

	if err := openshift.AddSchemes(scheme); err != nil {
		return fmt.Errorf("could not add new schemes to client: %w", err)
	}

	// TODO(): GetConfig generates a rest config from environment paths. We already have
	// the Kubeconfig at this point, so we should potentially find another way to generate
	// a rest config that doesn't rely on ctrl's implicit locations for it.
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("could not get kubeconfig: %w", err)
	}

	client, err := crclient.New(kubeconfig, crclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		return fmt.Errorf("could not get controller-runtime client: %w", err)
	}

	p.client = client

	k8sClientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("could not get k8s clientset: %w", err)
	}

	p.k8sClientset = k8sClientset

	return nil
}

func (p *DeployableByOlmCheck) initOpenShiftEngine() {
	if p.openshiftClient == nil {
		p.openshiftClient = openshift.NewClient(p.client, p.k8sClientset)
	}
}

// WithCSVTimeout customizes how long to wait for a ClusterServiceVersion to become healthy.
func WithCSVTimeout(csvTimeout time.Duration) Option {
	return func(oc *DeployableByOlmCheck) {
		oc.csvTimeout = csvTimeout
	}
}

// WithSubscriptionTimeout customizes how long to wait for a subscription to become healthy.
func WithSubscriptionTimeout(subscriptionTimeout time.Duration) Option {
	return func(oc *DeployableByOlmCheck) {
		oc.subscriptionTimeout = subscriptionTimeout
	}
}

// NewDeployableByOlmCheck will return a check that validates if an operator
// is deployable by OLM. An empty dockerConfig value implies that the images
// in scope are public. An empty channel value implies that the check should
// introspect the channel from the bundle. indexImage is required.
func NewDeployableByOlmCheck(
	indexImage,
	dockerConfig,
	channel string,
	opts ...Option,
) *DeployableByOlmCheck {
	c := &DeployableByOlmCheck{
		dockerConfig: dockerConfig,
		indexImage:   indexImage,
		channel:      channel,
	}

	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (p *DeployableByOlmCheck) Validate(ctx context.Context, bundleRef image.ImageReference) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	if err := p.initClient(); err != nil {
		return false, fmt.Errorf("%v", err)
	}
	p.initOpenShiftEngine()
	report, err := bundle.Validate(ctx, bundleRef.ImageFSPath)
	if err != nil {
		return false, fmt.Errorf("%v", err)
	}

	if !report.Passed { // validation didn't throw an error, but it also didn't pass.
		erroredValidations := []validationerrors.ManifestResult{}
		for _, r := range report.Results {
			if r.HasError() {
				erroredValidations = append(erroredValidations, r)
			}
		}

		return false, fmt.Errorf("the bundle cannot be deployed because deployment validation has failed: %+v", erroredValidations)
	}

	// gather the list of registry and pod images
	beforeOperatorImages, err := p.getImages(ctx)
	if err != nil {
		return false, fmt.Errorf("%v", err)
	}

	// retrieve the required data
	operatorData, err := p.operatorMetadata(ctx, bundleRef)
	if err != nil {
		return false, fmt.Errorf("%v", err)
	}

	logger.V(log.DBG).Info("operator metadata", "metadata", *operatorData)

	// create k8s custom resources for the operator deployment
	err = p.setUp(ctx, operatorData)
	defer p.cleanUp(ctx, *operatorData)

	if err != nil {
		return false, fmt.Errorf("%v", err)
	}

	installedCSV, err := p.installedCSV(ctx, *operatorData)
	if err != nil {
		return false, fmt.Errorf("%v", err)
	}
	operatorData.InstalledCsv = installedCSV
	logger.V(log.TRC).Info("installed CSV", "csv", operatorData.InstalledCsv)

	p.csvReady, err = p.isCSVReady(ctx, *operatorData)
	if err != nil {
		return false, fmt.Errorf("%v", err)
	}

	afterOperatorImages, err := p.getImages(ctx)
	if err != nil {
		return false, fmt.Errorf("%v", err)
	}

	operatorImages := diffImageList(beforeOperatorImages, afterOperatorImages)
	p.validImages = checkImageSource(ctx, operatorImages)

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

func checkImageSource(ctx context.Context, operatorImages []string) bool {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.DBG).Info("checking that images are from approved sources")

	registries := slices.Collect(maps.Keys(approvedRegistries))

	logger.V(log.DBG).Info("list of approved registries", "registries", registries)
	allApproved := true
	for _, image := range operatorImages {
		userRegistry := strings.Split(image, "/")[0]
		if _, ok := approvedRegistries[userRegistry]; !ok {
			logger.Info("warning: unapproved registry found for image", "image", image)
			allApproved = false
		}
	}
	if allApproved {
		logger.V(log.DBG).Info("all images are from approved sources")
	}
	return allApproved
}

func (p *DeployableByOlmCheck) operatorMetadata(ctx context.Context, bundleRef image.ImageReference) (*operatorData, error) {
	logger := logr.FromContextOrDiscard(ctx)

	// retrieve the operator metadata from bundle image
	annotationsFileName := filepath.Join(bundleRef.ImageFSPath, "metadata", "annotations.yaml")
	annotationsFile, err := os.Open(annotationsFileName)
	if err != nil {
		return nil, fmt.Errorf("could not open annotations.yaml: %w", err)
	}
	defer annotationsFile.Close()
	annotations, err := bundle.LoadAnnotations(ctx, annotationsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to get annotations.yaml from the bundle: %w", err)
	}

	catalogImage := p.indexImage

	// introspect the channel from the bundle.
	channel := annotations.DefaultChannelName

	// The user provided a channel configuration so we will
	// use that instead of the introspected value.
	if len(p.channel) != 0 {
		channel = p.channel
	}

	packageName := annotations.PackageName

	bundle, err := manifests.GetBundleFromDir(bundleRef.ImageFSPath)
	if err != nil {
		return nil, err
	}

	installModes := make(map[operatorsv1alpha1.InstallModeType]operatorsv1alpha1.InstallMode)
	for _, val := range bundle.CSV.Spec.InstallModes {
		installModes[val.Type] = val
	}

	// validating that package name complies with DNS-1035 labeling constraints
	// ensuring that CatalogSources can be created/referenced in cluster and reconciles properly
	// if there is an error we need to prefix the name so the install/reconciles can continue
	appName := packageName
	if msgs := validation.IsDNS1035Label(packageName); len(msgs) != 0 {
		logger.V(log.DBG).Info(fmt.Sprintf("package name %s does not comply with DNS-1035, prefixing to avoid errors", packageName))
		appName = "p-" + packageName
	}

	deploymentNames := make([]string, 0, len(bundle.CSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs))
	for _, deployment := range bundle.CSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		deploymentNames = append(deploymentNames, deployment.Name)
	}

	return &operatorData{
		CatalogImage:     catalogImage,
		Channel:          channel,
		PackageName:      packageName,
		App:              appName,
		InstallNamespace: appName,
		TargetNamespace:  appName + "-target",
		InstallModes:     installModes,
		DeploymentNames:  deploymentNames,
	}, nil
}

func (p *DeployableByOlmCheck) setUp(ctx context.Context, operatorData *operatorData) error {
	logger := logr.FromContextOrDiscard(ctx)

	if _, err := p.openshiftClient.CreateNamespace(ctx, operatorData.InstallNamespace); err != nil && !errors.Is(err, openshift.ErrAlreadyExists) {
		return err
	}

	if _, err := p.openshiftClient.CreateNamespace(ctx, operatorData.TargetNamespace); err != nil && !errors.Is(err, openshift.ErrAlreadyExists) {
		return err
	}

	dockerconfig := p.dockerConfig
	if len(dockerconfig) != 0 {
		// the user provided a dockerConfig to pass through for use with scorecard.
		content, err := p.readFileAsByteArray(dockerconfig)
		if err != nil {
			return err
		}
		data := map[string]string{".dockerconfigjson": string(content)}
		if _, err := p.openshiftClient.CreateSecret(
			ctx,
			secretName,
			data,
			corev1.SecretTypeDockerConfigJson,
			operatorData.InstallNamespace,
		); err != nil && !errors.Is(err, openshift.ErrAlreadyExists) {
			return err
		}
	} else {
		logger.V(log.DBG).Info("no docker config file is found to access the index image in private registries, using anonymous auth")
	}

	if strings.Contains(operatorData.CatalogImage, imageRegistryService) {
		indexImageNamespace := strings.Split(operatorData.CatalogImage, "/")[1]
		if len(indexImageNamespace) != 0 {
			// create rolebindings for the pipeline service account
			if err := p.grantRegistryPermissionToServiceAccount(
				ctx,
				pipelineServiceAccount,
				operatorData.InstallNamespace,
				indexImageNamespace,
			); err != nil {
				return err
			}
			// create rolebinding for the default OperatorHub catalog sources
			if err := p.grantRegistryPermissionToServiceAccount(
				ctx,
				operatorData.App,
				openshiftMarketplaceNamespace,
				indexImageNamespace,
			); err != nil {
				return err
			}
			// create rolebindings for the custom catalog
			if err := p.grantRegistryPermissionToServiceAccount(
				ctx,
				operatorData.App,
				operatorData.InstallNamespace,
				indexImageNamespace,
			); err != nil {
				return err
			}
		}
	}

	catalogSourceData := openshift.CatalogSourceData{
		Name:    operatorData.App,
		Image:   operatorData.CatalogImage,
		Secrets: []string{secretName},
	}
	if _, err := p.openshiftClient.CreateCatalogSource(
		ctx,
		catalogSourceData,
		operatorData.InstallNamespace,
	); err != nil && !errors.Is(err, openshift.ErrAlreadyExists) {
		return err
	}

	operatorGroupData := p.generateOperatorGroupData(ctx, operatorData)
	if _, err := p.openshiftClient.CreateOperatorGroup(
		ctx,
		operatorGroupData,
		operatorData.InstallNamespace,
	); err != nil && !errors.Is(err, openshift.ErrAlreadyExists) {
		return err
	}

	subscriptionData := openshift.SubscriptionData{
		Name:                   operatorData.App,
		Channel:                operatorData.Channel,
		CatalogSource:          operatorData.App,
		CatalogSourceNamespace: operatorData.InstallNamespace,
		Package:                operatorData.PackageName,
	}
	if _, err := p.openshiftClient.CreateSubscription(
		ctx,
		subscriptionData,
		operatorData.InstallNamespace,
	); err != nil && !errors.Is(err, openshift.ErrAlreadyExists) {
		return err
	}
	return nil
}

func (p *DeployableByOlmCheck) generateOperatorGroupData(ctx context.Context, operatorData *operatorData) openshift.OperatorGroupData {
	logger := logr.FromContextOrDiscard(ctx)

	var installMode operatorsv1alpha1.InstallModeType
	for _, v := range prioritizedInstallModes {
		if operatorData.InstallModes[v].Supported {
			installMode = operatorData.InstallModes[v].Type
			break
		}
	}
	logger.V(log.DBG).Info("operator install mode", "installMode", installMode)
	targetNamespaces := make([]string, 2)

	switch installMode {
	case operatorsv1alpha1.InstallModeTypeOwnNamespace:
		targetNamespaces = []string{operatorData.InstallNamespace}
	case operatorsv1alpha1.InstallModeTypeSingleNamespace:
		targetNamespaces = []string{operatorData.TargetNamespace}
	case operatorsv1alpha1.InstallModeTypeMultiNamespace:
		targetNamespaces = []string{operatorData.TargetNamespace, operatorData.InstallNamespace}
	case operatorsv1alpha1.InstallModeTypeAllNamespaces:
		targetNamespaces = []string{}
	}
	logger.V(log.DBG).Info("OperatorGroup TargetNamespaces", "namespace", targetNamespaces)
	operatorData.CsvNamespaces = targetNamespaces
	return openshift.OperatorGroupData{Name: operatorData.App, TargetNamespaces: targetNamespaces}
}

func (p *DeployableByOlmCheck) grantRegistryPermissionToServiceAccount(ctx context.Context, serviceAccount, serviceAccountNamespace, indexImageNamespace string) error {
	for _, role := range []string{registryViewerRole, imagePullerRole} {
		roleBindingData := openshift.RoleBindingData{
			Name:      fmt.Sprintf("%s:%s:%s", serviceAccount, serviceAccountNamespace, role),
			Subjects:  []string{serviceAccount},
			Role:      role,
			Namespace: serviceAccountNamespace,
		}
		if _, err := p.openshiftClient.CreateRoleBinding(
			ctx,
			roleBindingData,
			indexImageNamespace,
		); err != nil && !errors.Is(err, openshift.ErrAlreadyExists) {
			return err
		}
	}
	return nil
}

type watchFunc func(ctx context.Context, client openshift.Client, name, namespace string) (string, bool, error)

func watch(ctx context.Context, client openshift.Client, wg *sync.WaitGroup, name, namespace string, timeout time.Duration, channel chan string, fn watchFunc) {
	logger := logr.FromContextOrDiscard(ctx)

	defer wg.Done()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		logger.V(log.DBG).Info("watch: waiting for object to become ready", "namespace", namespace, "name", name)
		obj, done, err := fn(ctx, client, name, namespace)
		if err != nil {
			// Something bad happened. Get out of town
			err := fmt.Errorf("watch: could not retrieve the object %s/%s: %v", namespace, name, err)
			channel <- fmt.Sprintf("%s %v", errorPrefix, err)
			return
		}
		if done {
			logger.V(log.DBG).Info("watch: successfully retrieved object", "namespace", namespace, "object", obj)
			channel <- obj
			return
		}
		logger.V(log.DBG).Info("watch: object is not set yet, retrying", "namespace", namespace, "name", name)

		select {
		case <-ctx.Done():
			channel <- fmt.Sprintf("%s %v", errorPrefix, ctx.Err())
			return
		default:
			time.Sleep(2 * time.Second)
		}
	}
}

func csvStatusSucceeded(ctx context.Context, client openshift.Client, name, namespace string) (string, bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	csv, err := client.GetCSV(ctx, name, namespace)
	if err != nil && !errors.Is(err, openshift.ErrNotFound) {
		// This is not a normal error. Get out of town
		return "", false, fmt.Errorf("failed to fetch the csv %s from namespace %s: %w", name, namespace, err)
	}
	// if the CSV phase is succeeded, stop the querying
	if csv != nil && csv.Status.Phase == operatorsv1alpha1.CSVPhaseSucceeded {
		logger.V(log.DBG).Info("CSV created successfully", "namespace", namespace, "name", name)
		return name, true, nil
	}
	return "", false, nil
}

func (p *DeployableByOlmCheck) isCSVReady(ctx context.Context, operatorData operatorData) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	var CsvNamespaces []string
	if len(operatorData.CsvNamespaces) == 0 {
		CsvNamespaces = []string{operatorData.TargetNamespace, "default", openshiftMarketplaceNamespace}
	} else {
		CsvNamespaces = []string{operatorData.CsvNamespaces[0]}
	}
	logger.V(log.TRC).Info("looking for csv", "namespace", CsvNamespaces, "csv", operatorData.InstalledCsv)

	csvChannel := make(chan string)

	var wg sync.WaitGroup

	for _, CsvNamespace := range CsvNamespaces {
		wg.Add(1)
		go watch(ctx, p.openshiftClient, &wg, operatorData.InstalledCsv, CsvNamespace, p.csvTimeout, csvChannel, csvStatusSucceeded)
	}

	go func() {
		wg.Wait()
		close(csvChannel)
	}()

	for msg := range csvChannel {
		if strings.Contains(msg, errorPrefix) {
			return false, fmt.Errorf("%s", msg)
		}
		if len(msg) == 0 {
			return false, nil
		}
	}
	return true, nil
}

func subscriptionCsvIsInstalled(ctx context.Context, client openshift.Client, name, namespace string) (string, bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	sub, err := client.GetSubscription(ctx, name, namespace)
	if err != nil && !errors.Is(err, openshift.ErrNotFound) {
		return "", false, fmt.Errorf("failed to fetch the subscription %s from namespace %s: %w", name, namespace, err)
	}
	logger.V(log.TRC).Info("current subscription status", "status", sub.Status)
	installedCSV := sub.Status.InstalledCSV
	// if the installedCSV field is present, stop the querying
	if len(installedCSV) > 0 {
		return installedCSV, true, nil
	}
	return "", false, nil
}

func (p *DeployableByOlmCheck) installedCSV(ctx context.Context, operatorData operatorData) (string, error) {
	installedCSVChannel := make(chan string)

	var wg sync.WaitGroup
	// query API server for the installed CSV field of the created subscription
	wg.Add(1)
	go watch(ctx, p.openshiftClient, &wg, operatorData.App, operatorData.InstallNamespace, p.subscriptionTimeout, installedCSVChannel, subscriptionCsvIsInstalled)

	go func() {
		wg.Wait()
		close(installedCSVChannel)
	}()

	installedCsv := ""
	for msg := range installedCSVChannel {
		if strings.Contains(msg, errorPrefix) {
			return "", fmt.Errorf("%s", msg)
		}
		installedCsv = msg
	}

	return installedCsv, nil
}

func (p *DeployableByOlmCheck) cleanUp(ctx context.Context, operatorData operatorData) {
	logger := logr.FromContextOrDiscard(ctx)

	logger.V(log.DBG).Info("dumping data in artifacts/ directory")

	subs, err := p.openshiftClient.GetSubscription(ctx, operatorData.App, operatorData.InstallNamespace)
	if err != nil {
		logger.Info("warning: unable to retrieve the subscription")
	} else {
		err := p.writeToFile(ctx, subs)
		if err != nil {
			logger.Error(err, "could not write subscription to storage")
		}
	}

	cs, err := p.openshiftClient.GetCatalogSource(ctx, operatorData.App, operatorData.InstallNamespace)
	if err != nil {
		logger.Info("warning: unable to retrieve the catalogsource")
	} else {
		if err := p.writeToFile(ctx, cs); err != nil {
			logger.Error(err, "could not write catalogsource to storage")
		}
	}

	og, err := p.openshiftClient.GetOperatorGroup(ctx, operatorData.App, operatorData.InstallNamespace)
	if err != nil {
		logger.Info("warning: unable to retrieve the operatorgroup")
	} else {
		if err := p.writeToFile(ctx, og); err != nil {
			logger.Error(err, "could not write operatorgroup to storage")
		}
	}

	installNamespace, err := p.openshiftClient.GetNamespace(ctx, operatorData.InstallNamespace)
	if err != nil {
		logger.Info("warning: unable to retrieve the install namespace")
	} else {
		if err := p.writeToFile(ctx, installNamespace); err != nil {
			logger.Error(err, "could not write install namespace to storage")
		}
	}

	targetNamespace, err := p.openshiftClient.GetNamespace(ctx, operatorData.TargetNamespace)
	if err != nil {
		logger.Info("warning: unable to retrieve the target namespace")
	} else {
		if err := p.writeToFile(ctx, targetNamespace); err != nil {
			logger.Error(err, "could not write target namespace to storage")
		}
	}

	for _, deploymentName := range operatorData.DeploymentNames {
		deployment, err := p.openshiftClient.GetDeployment(ctx, deploymentName, operatorData.InstallNamespace)
		if err != nil {
			logger.Info(fmt.Sprintf("warning: unable to retrieve deployment: %s", err))
			continue
		}

		if err := p.writeToFile(ctx, deployment); err != nil {
			logger.Error(err, "could not write deployment to storage")
		}

		pods, err := p.openshiftClient.GetDeploymentPods(ctx, deploymentName, operatorData.InstallNamespace)
		if err != nil {
			logger.Info(fmt.Sprintf("warning: unable to retrieve deployment pods: %s", err))
			continue
		}

		for _, pod := range pods {
			jsonManifest, err := json.Marshal(pod.Status)
			if err != nil {
				logger.Error(err, "unable to marshal to json")
			}

			filename := fmt.Sprintf("%s-PodStatus.json", pod.Name)
			if artifactWriter := artifacts.WriterFromContext(ctx); artifactWriter != nil {
				if _, err := artifactWriter.WriteFile(filename, bytes.NewReader(jsonManifest)); err != nil {
					logger.Error(err, "failed to write the PodStatus to the file")
				}
			}

			logs, err := p.openshiftClient.GetPodLogs(ctx, pod.Name, pod.Namespace)
			if err != nil {
				logger.Info(fmt.Sprintf("warning: unable to retrieve pod logs: %s", err))
				continue
			}

			for container, logContents := range logs {
				filename := fmt.Sprintf("%s-%s.log", pod.Name, container)
				if artifactWriter := artifacts.WriterFromContext(ctx); artifactWriter != nil {
					if _, err := artifactWriter.WriteFile(filename, logContents); err != nil {
						logger.Error(err, "failed to write the pod logs to the file")
					}
				}
			}
		}
	}

	logger.V(log.TRC).Info("deleting the resources created by DeployableByOLM Check")
	_ = p.openshiftClient.DeleteSubscription(ctx, operatorData.App, operatorData.InstallNamespace)
	_ = p.openshiftClient.DeleteCatalogSource(ctx, operatorData.App, operatorData.InstallNamespace)
	_ = p.openshiftClient.DeleteOperatorGroup(ctx, operatorData.App, operatorData.InstallNamespace)
	_ = p.openshiftClient.DeleteSecret(ctx, secretName, operatorData.InstallNamespace)

	if strings.Contains(operatorData.CatalogImage, imageRegistryService) {
		indexImageNamespace := strings.Split(operatorData.CatalogImage, "/")[1]
		operatorServiceAccount := operatorData.App
		operatorNamespace := operatorData.InstallNamespace
		// remove pipeline-related rolebindings
		_ = p.openshiftClient.DeleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", pipelineServiceAccount, operatorNamespace, registryViewerRole), indexImageNamespace)
		_ = p.openshiftClient.DeleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", pipelineServiceAccount, operatorNamespace, imagePullerRole), indexImageNamespace)
		// remove rolebindings required for the default OperatorHub catalog sources
		_ = p.openshiftClient.DeleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", operatorServiceAccount, openshiftMarketplaceNamespace, registryViewerRole), indexImageNamespace)
		_ = p.openshiftClient.DeleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", operatorServiceAccount, openshiftMarketplaceNamespace, imagePullerRole), indexImageNamespace)
		// remove rolebindings required for custom catalog sources
		_ = p.openshiftClient.DeleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", operatorServiceAccount, operatorNamespace, registryViewerRole), indexImageNamespace)
		_ = p.openshiftClient.DeleteRoleBinding(ctx, fmt.Sprintf("%s:%s:%s", operatorServiceAccount, operatorNamespace, imagePullerRole), indexImageNamespace)
	}
	_ = p.openshiftClient.DeleteNamespace(ctx, operatorData.InstallNamespace)
	_ = p.openshiftClient.DeleteNamespace(ctx, operatorData.TargetNamespace)
}

func (p *DeployableByOlmCheck) writeToFile(ctx context.Context, data any) error {
	obj, err := apiruntime.DefaultUnstructuredConverter.ToUnstructured(data)
	if err != nil {
		return fmt.Errorf("unable to convert the object to unstructured.Unstructured: %w", err)
	}

	group := "operators.coreos.com"
	var version, kind string
	u := &unstructured.Unstructured{Object: obj}
	switch data.(type) {
	case *operatorsv1alpha1.CatalogSource:
		version = "v1alpha1"
		kind = "CatalogSource"
	case *operatorsv1.OperatorGroup:
		version = "v1"
		kind = "OperatorGroup"
	case *operatorsv1alpha1.Subscription:
		version = "v1alpha1"
		kind = "Subscription"
	case *corev1.Namespace:
		group = ""
		version = "v1"
		kind = "Namespace"
	case *appsv1.Deployment:
		group = "apps"
		version = "v1"
		kind = "Deployment"
	default:
		return fmt.Errorf("go type unsupported")
	}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	})

	jsonManifest, err := json.Marshal(u)
	if err != nil {
		return fmt.Errorf("unable to marshal to json: %w", err)
	}

	filename := fmt.Sprintf("%s-%s.json", u.GetName(), u.GetKind())
	if artifactWriter := artifacts.WriterFromContext(ctx); artifactWriter != nil {
		if _, err := artifactWriter.WriteFile(filename, bytes.NewReader(jsonManifest)); err != nil {
			return fmt.Errorf("failed to write the k8s object to the file: %w", err)
		}
	}

	return nil
}

func (p *DeployableByOlmCheck) readFileAsByteArray(filename string) ([]byte, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading the file: %s: %w", filename, err)
	}
	return content, nil
}

func (p *DeployableByOlmCheck) getImages(ctx context.Context) (map[string]struct{}, error) {
	return p.openshiftClient.GetImages(ctx)
}

func (p *DeployableByOlmCheck) Name() string {
	return "DeployableByOLM"
}

func (p *DeployableByOlmCheck) Metadata() check.Metadata {
	return check.Metadata{
		Description:      "Checking if the operator could be deployed by OLM",
		Level:            "best",
		KnowledgeBaseURL: "https://sdk.operatorframework.io/docs/olm-integration/testing-deployment/",
		CheckURL:         "https://sdk.operatorframework.io/docs/olm-integration/testing-deployment/",
	}
}

func (p *DeployableByOlmCheck) Help() check.HelpText {
	return check.HelpText{
		Message:    "It is required that your operator could be deployed by OLM",
		Suggestion: "Follow the guidelines on the operator-sdk website to learn how to package your operator https://sdk.operatorframework.io/docs/olm-integration/cli-overview/",
	}
}

func (p *DeployableByOlmCheck) RequiredFilePatterns() []string {
	return bundle.BundleFiles
}
