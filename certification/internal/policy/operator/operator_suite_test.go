package operator

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOperator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator Suite")
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.TraceLevel)

	viper.SetEnvPrefix("pflt")
	viper.AutomaticEnv()
}

type FakeOperatorSdkEngine struct {
	OperatorSdkReport   cli.OperatorSdkScorecardReport
	OperatorSdkBVReport cli.OperatorSdkBundleValidateReport
}

func (f FakeOperatorSdkEngine) BundleValidate(image string, opts cli.OperatorSdkBundleValidateOptions) (*cli.OperatorSdkBundleValidateReport, error) {
	return &f.OperatorSdkBVReport, nil
}

func (f FakeOperatorSdkEngine) Scorecard(image string, opts cli.OperatorSdkScorecardOptions) (*cli.OperatorSdkScorecardReport, error) {
	return &f.OperatorSdkReport, nil
}

type BadOperatorSdkEngine struct{}

func (bose BadOperatorSdkEngine) Scorecard(bundleImage string, opts cli.OperatorSdkScorecardOptions) (*cli.OperatorSdkScorecardReport, error) {
	operatorSdkReport := cli.OperatorSdkScorecardReport{
		Stdout: "Bad Stdout",
		Stderr: "Bad Stderr",
		Items:  []cli.OperatorSdkScorecardItem{},
	}
	return &operatorSdkReport, errors.New("the Operator Sdk Scorecard has failed")
}

func (bose BadOperatorSdkEngine) BundleValidate(bundleImage string, opts cli.OperatorSdkBundleValidateOptions) (*cli.OperatorSdkBundleValidateReport, error) {
	operatorSdkReport := cli.OperatorSdkBundleValidateReport{
		Stdout:  "Bad Stdout",
		Stderr:  "Bad Stderr",
		Passed:  false,
		Outputs: []cli.OperatorSdkBundleValidateOutput{},
	}
	return &operatorSdkReport, errors.New("the Operator Sdk Bundle Validate has failed")
}

type FakeOpenshiftEngine struct{}

func (foe FakeOpenshiftEngine) CreateNamespace(name string, opts cli.OpenshiftOptions) (*corev1.Namespace, error) {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}, nil
}

func (foe FakeOpenshiftEngine) DeleteNamespace(name string, opts cli.OpenshiftOptions) error {
	return nil
}

func (foe FakeOpenshiftEngine) GetNamespace(name string) (*corev1.Namespace, error) {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}, nil
}

func (foe FakeOpenshiftEngine) CreateSecret(name string, content map[string]string, secretType corev1.SecretType, opts cli.OpenshiftOptions) (*corev1.Secret, error) {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pull-image-secret",
			Namespace: "test-ns",
		},
		Type:       "kubernetes.io/dockerconfigjson",
		StringData: map[string]string{".dockerconfigjson": "secretData"},
	}, nil
}

func (foe FakeOpenshiftEngine) DeleteSecret(name string, opts cli.OpenshiftOptions) error {
	return nil
}

func (foe FakeOpenshiftEngine) GetSecret(name string, opts cli.OpenshiftOptions) (*corev1.Secret, error) {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pull-image-secret",
			Namespace: "test-ns",
		},
		Type:       "kubernetes.io/dockerconfigjson",
		StringData: map[string]string{".dockerconfigjson": "secretData"},
	}, nil
}

func (foe FakeOpenshiftEngine) CreateOperatorGroup(data cli.OperatorGroupData, opts cli.OpenshiftOptions) (*operatorv1.OperatorGroup, error) {
	return &operatorv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-og",
			Namespace: "test-ns",
		},
		Spec: operatorv1.OperatorGroupSpec{
			TargetNamespaces: []string{"test-ns"},
		},
	}, nil
}

func (foe FakeOpenshiftEngine) DeleteOperatorGroup(name string, opts cli.OpenshiftOptions) error {
	return nil
}

func (foe FakeOpenshiftEngine) GetOperatorGroup(name string, opts cli.OpenshiftOptions) (*operatorv1.OperatorGroup, error) {
	return &operatorv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-og",
			Namespace: "test-ns",
		},
		Spec: operatorv1.OperatorGroupSpec{
			TargetNamespaces: []string{"test-ns"},
		},
	}, nil
}

func (foe FakeOpenshiftEngine) CreateCatalogSource(data cli.CatalogSourceData, opts cli.OpenshiftOptions) (*operatorv1alpha1.CatalogSource, error) {
	return &operatorv1alpha1.CatalogSource{
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType: operatorv1alpha1.SourceTypeGrpc,
			Image:      "indexImageUri",
		},
	}, nil
}

func (foe FakeOpenshiftEngine) DeleteCatalogSource(name string, opts cli.OpenshiftOptions) error {
	return nil
}

func (foe FakeOpenshiftEngine) GetCatalogSource(name string, opts cli.OpenshiftOptions) (*operatorv1alpha1.CatalogSource, error) {
	return &operatorv1alpha1.CatalogSource{
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType: operatorv1alpha1.SourceTypeGrpc,
			Image:      "indexImageUri",
		},
	}, nil
}

func (foe FakeOpenshiftEngine) CreateSubscription(data cli.SubscriptionData, opts cli.OpenshiftOptions) (*operatorv1alpha1.Subscription, error) {
	return &operatorv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sub",
			Namespace: "test-ns",
		},
		Spec: &operatorv1alpha1.SubscriptionSpec{
			CatalogSource:          "test-cs",
			CatalogSourceNamespace: "openshift-marketplace",
			Channel:                "stable",
			Package:                "test-operator",
		},
		Status: operatorv1alpha1.SubscriptionStatus{
			InstalledCSV: "csv-v0.0.0",
		},
	}, nil
}

func (foe FakeOpenshiftEngine) DeleteSubscription(name string, opts cli.OpenshiftOptions) error {
	return nil
}

func (foe FakeOpenshiftEngine) GetSubscription(name string, opts cli.OpenshiftOptions) (*operatorv1alpha1.Subscription, error) {
	return &operatorv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sub",
			Namespace: "test-ns",
		},
		Spec: &operatorv1alpha1.SubscriptionSpec{
			CatalogSource:          "test-cs",
			CatalogSourceNamespace: "openshift-marketplace",
			Channel:                "stable",
			Package:                "test-operator",
		},
		Status: operatorv1alpha1.SubscriptionStatus{
			InstalledCSV: "csv-v0.0.0",
		},
	}, nil
}

func (foe FakeOpenshiftEngine) GetCSV(name string, opts cli.OpenshiftOptions) (*operatorv1alpha1.ClusterServiceVersion, error) {
	return &operatorv1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "csv-v0.0.0",
			Namespace: "test-ns",
		},
		Spec: operatorv1alpha1.ClusterServiceVersionSpec{},
		Status: operatorv1alpha1.ClusterServiceVersionStatus{
			Phase: operatorv1alpha1.CSVPhaseSucceeded,
		},
	}, nil
}

func (foe FakeOpenshiftEngine) GetImages() (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

type BadOpenshiftEngine struct{}

func (foe BadOpenshiftEngine) CreateNamespace(name string, opts cli.OpenshiftOptions) (*corev1.Namespace, error) {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}, nil
}

func (foe BadOpenshiftEngine) DeleteNamespace(name string, opts cli.OpenshiftOptions) error {
	return nil
}

func (foe BadOpenshiftEngine) GetNamespace(name string) (*corev1.Namespace, error) {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}, nil
}

func (foe BadOpenshiftEngine) CreateSecret(name string, content map[string]string, secretType corev1.SecretType, opts cli.OpenshiftOptions) (*corev1.Secret, error) {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pull-image-secret",
			Namespace: "test-ns",
		},
		Type:       "kubernetes.io/dockerconfigjson",
		StringData: map[string]string{".dockerconfigjson": "secretData"},
	}, nil
}

func (foe BadOpenshiftEngine) DeleteSecret(name string, opts cli.OpenshiftOptions) error {
	return nil
}

func (foe BadOpenshiftEngine) GetSecret(name string, opts cli.OpenshiftOptions) (*corev1.Secret, error) {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pull-image-secret",
			Namespace: "test-ns",
		},
		Type:       "kubernetes.io/dockerconfigjson",
		StringData: map[string]string{".dockerconfigjson": "secretData"},
	}, nil
}

func (foe BadOpenshiftEngine) CreateOperatorGroup(data cli.OperatorGroupData, opts cli.OpenshiftOptions) (*operatorv1.OperatorGroup, error) {
	return &operatorv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-og",
			Namespace: "test-ns",
		},
		Spec: operatorv1.OperatorGroupSpec{
			TargetNamespaces: []string{"test-ns"},
		},
	}, nil
}

func (foe BadOpenshiftEngine) DeleteOperatorGroup(name string, opts cli.OpenshiftOptions) error {
	return nil
}

func (foe BadOpenshiftEngine) GetOperatorGroup(name string, opts cli.OpenshiftOptions) (*operatorv1.OperatorGroup, error) {
	return &operatorv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-og",
			Namespace: "test-ns",
		},
		Spec: operatorv1.OperatorGroupSpec{
			TargetNamespaces: []string{"test-ns"},
		},
	}, nil
}

func (foe BadOpenshiftEngine) CreateCatalogSource(data cli.CatalogSourceData, opts cli.OpenshiftOptions) (*operatorv1alpha1.CatalogSource, error) {
	return &operatorv1alpha1.CatalogSource{
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType: operatorv1alpha1.SourceTypeGrpc,
			Image:      "indexImageUri",
		},
	}, nil
}

func (foe BadOpenshiftEngine) DeleteCatalogSource(name string, opts cli.OpenshiftOptions) error {
	return nil
}

func (foe BadOpenshiftEngine) GetCatalogSource(name string, opts cli.OpenshiftOptions) (*operatorv1alpha1.CatalogSource, error) {
	return &operatorv1alpha1.CatalogSource{
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType: operatorv1alpha1.SourceTypeGrpc,
			Image:      "indexImageUri",
		},
	}, nil
}

func (foe BadOpenshiftEngine) CreateSubscription(data cli.SubscriptionData, opts cli.OpenshiftOptions) (*operatorv1alpha1.Subscription, error) {
	return &operatorv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sub",
			Namespace: "test-ns",
		},
		Spec: &operatorv1alpha1.SubscriptionSpec{
			CatalogSource:          "test-cs",
			CatalogSourceNamespace: "openshift-marketplace",
			Channel:                "stable",
			Package:                "test-operator",
		},
		Status: operatorv1alpha1.SubscriptionStatus{},
	}, nil
}

func (foe BadOpenshiftEngine) DeleteSubscription(name string, opts cli.OpenshiftOptions) error {
	return nil
}

func (foe BadOpenshiftEngine) GetSubscription(name string, opts cli.OpenshiftOptions) (*operatorv1alpha1.Subscription, error) {
	return &operatorv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sub",
			Namespace: "test-ns",
		},
		Spec: &operatorv1alpha1.SubscriptionSpec{
			CatalogSource:          "test-cs",
			CatalogSourceNamespace: "openshift-marketplace",
			Channel:                "stable",
			Package:                "test-operator",
		},
		Status: operatorv1alpha1.SubscriptionStatus{},
	}, nil
}

func (foe BadOpenshiftEngine) GetCSV(name string, opts cli.OpenshiftOptions) (*operatorv1alpha1.ClusterServiceVersion, error) {
	return nil, nil
}

func (foe BadOpenshiftEngine) GetImages() (map[string]struct{}, error) {
	return nil, nil
}
