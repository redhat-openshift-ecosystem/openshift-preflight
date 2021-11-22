package operator

import (
	"context"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
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

func (foe FakeOpenshiftEngine) CreateNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}, nil
}

func (foe FakeOpenshiftEngine) DeleteNamespace(ctx context.Context, name string) error {
	return nil
}

func (foe FakeOpenshiftEngine) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}, nil
}

func (foe FakeOpenshiftEngine) CreateSecret(ctx context.Context, name string, content map[string]string, secretType corev1.SecretType, namespace string) (*corev1.Secret, error) {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pull-image-secret",
			Namespace: "test-ns",
		},
		Type:       "kubernetes.io/dockerconfigjson",
		StringData: map[string]string{".dockerconfigjson": "secretData"},
	}, nil
}

func (foe FakeOpenshiftEngine) DeleteSecret(ctx context.Context, name, namespace string) error {
	return nil
}

func (foe FakeOpenshiftEngine) GetSecret(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pull-image-secret",
			Namespace: "test-ns",
		},
		Type:       "kubernetes.io/dockerconfigjson",
		StringData: map[string]string{".dockerconfigjson": "secretData"},
	}, nil
}

func (foe FakeOpenshiftEngine) CreateOperatorGroup(ctx context.Context, data cli.OperatorGroupData, namespace string) (*operatorv1.OperatorGroup, error) {
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

func (foe FakeOpenshiftEngine) DeleteOperatorGroup(ctx context.Context, name, namespace string) error {
	return nil
}

func (foe FakeOpenshiftEngine) GetOperatorGroup(ctx context.Context, name, namespace string) (*operatorv1.OperatorGroup, error) {
	return &operatorv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-og",
			Namespace: "test-ns",
		},
		Spec: operatorv1.OperatorGroupSpec{
			TargetNamespaces: []string{"test-ns"},
		},
		Status: operatorv1.OperatorGroupStatus{
			LastUpdated: &metav1.Time{Time: time.Now()},
		},
	}, nil
}

func (foe FakeOpenshiftEngine) CreateRoleBinding(ctx context.Context, data cli.RoleBindingData, namespace string) (*rbacv1.RoleBinding, error) {
	subjectsObj := make([]rbacv1.Subject, 1)

	subjectsObj[0] = rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      "test-sa",
		Namespace: "test-ns",
	}
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rolebinding",
			Namespace: "a namespace",
		},
		Subjects: subjectsObj,
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "a role",
		},
	}, nil
}

func (foe FakeOpenshiftEngine) GetRoleBinding(ctx context.Context, name, namespace string) (*rbacv1.RoleBinding, error) {
	subjectsObj := make([]rbacv1.Subject, 1)

	subjectsObj[0] = rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      "test-sa",
		Namespace: "test-ns",
	}
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rolebinding",
			Namespace: "a namespace",
		},
		Subjects: subjectsObj,
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "a role",
		},
	}, nil
}

func (foe FakeOpenshiftEngine) DeleteRoleBinding(ctx context.Context, name, namespace string) error {
	return nil
}

func (foe FakeOpenshiftEngine) CreateCatalogSource(ctx context.Context, data cli.CatalogSourceData, namespace string) (*operatorv1alpha1.CatalogSource, error) {
	return &operatorv1alpha1.CatalogSource{
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType: operatorv1alpha1.SourceTypeGrpc,
			Image:      "indexImageUri",
		},
	}, nil
}

func (foe FakeOpenshiftEngine) DeleteCatalogSource(ctx context.Context, name, namespace string) error {
	return nil
}

func (foe FakeOpenshiftEngine) GetCatalogSource(ctx context.Context, name, namespace string) (*operatorv1alpha1.CatalogSource, error) {
	return &operatorv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cs",
			Namespace: "test-ns",
		},
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType: operatorv1alpha1.SourceTypeGrpc,
			Image:      "indexImageUri",
		},
	}, nil
}

func (foe FakeOpenshiftEngine) CreateSubscription(ctx context.Context, data cli.SubscriptionData, namespace string) (*operatorv1alpha1.Subscription, error) {
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

func (foe FakeOpenshiftEngine) DeleteSubscription(ctx context.Context, name, namespace string) error {
	return nil
}

func (foe FakeOpenshiftEngine) GetSubscription(ctx context.Context, name, namespace string) (*operatorv1alpha1.Subscription, error) {
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

func (foe FakeOpenshiftEngine) GetCSV(ctx context.Context, name, namespace string) (*operatorv1alpha1.ClusterServiceVersion, error) {
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

func (foe FakeOpenshiftEngine) GetImages(ctx context.Context) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

type BadOpenshiftEngine struct{}

func (foe BadOpenshiftEngine) CreateNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}, nil
}

func (foe BadOpenshiftEngine) DeleteNamespace(ctx context.Context, name string) error {
	return nil
}

func (foe BadOpenshiftEngine) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}, nil
}

func (foe BadOpenshiftEngine) CreateSecret(ctx context.Context, name string, content map[string]string, secretType corev1.SecretType, namespace string) (*corev1.Secret, error) {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pull-image-secret",
			Namespace: "test-ns",
		},
		Type:       "kubernetes.io/dockerconfigjson",
		StringData: map[string]string{".dockerconfigjson": "secretData"},
	}, nil
}

func (foe BadOpenshiftEngine) DeleteSecret(ctx context.Context, name, namespace string) error {
	return nil
}

func (foe BadOpenshiftEngine) GetSecret(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pull-image-secret",
			Namespace: "test-ns",
		},
		Type:       "kubernetes.io/dockerconfigjson",
		StringData: map[string]string{".dockerconfigjson": "secretData"},
	}, nil
}

func (foe BadOpenshiftEngine) CreateOperatorGroup(ctx context.Context, data cli.OperatorGroupData, namespace string) (*operatorv1.OperatorGroup, error) {
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

func (foe BadOpenshiftEngine) DeleteOperatorGroup(ctx context.Context, name, namespace string) error {
	return nil
}

func (foe BadOpenshiftEngine) GetOperatorGroup(ctx context.Context, name, namespace string) (*operatorv1.OperatorGroup, error) {
	return &operatorv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-og",
			Namespace: "test-ns",
		},
		Spec: operatorv1.OperatorGroupSpec{
			TargetNamespaces: []string{"test-ns"},
		},
		Status: operatorv1.OperatorGroupStatus{
			LastUpdated: &metav1.Time{Time: time.Now()},
		},
	}, nil
}

func (foe BadOpenshiftEngine) CreateCatalogSource(ctx context.Context, data cli.CatalogSourceData, namespace string) (*operatorv1alpha1.CatalogSource, error) {
	return &operatorv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cs",
			Namespace: "test-ns",
		},
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType: operatorv1alpha1.SourceTypeGrpc,
			Image:      "indexImageUri",
		},
	}, nil
}

func (foe BadOpenshiftEngine) DeleteCatalogSource(ctx context.Context, name, namespace string) error {
	return nil
}

func (foe BadOpenshiftEngine) GetCatalogSource(ctx context.Context, name, namespace string) (*operatorv1alpha1.CatalogSource, error) {
	return &operatorv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cs",
			Namespace: "test-ns",
		},
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType: operatorv1alpha1.SourceTypeGrpc,
			Image:      "indexImageUri",
		},
	}, nil
}

func (foe BadOpenshiftEngine) CreateSubscription(ctx context.Context, data cli.SubscriptionData, namespace string) (*operatorv1alpha1.Subscription, error) {
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

func (foe BadOpenshiftEngine) DeleteSubscription(ctx context.Context, name, namespace string) error {
	return nil
}

func (foe BadOpenshiftEngine) GetSubscription(ctx context.Context, name, namespace string) (*operatorv1alpha1.Subscription, error) {
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

func (foe BadOpenshiftEngine) GetCSV(ctx context.Context, name, namespace string) (*operatorv1alpha1.ClusterServiceVersion, error) {
	return nil, nil
}

func (foe BadOpenshiftEngine) GetImages(ctx context.Context) (map[string]struct{}, error) {
	return nil, nil
}

func (foe BadOpenshiftEngine) CreateRoleBinding(ctx context.Context, data cli.RoleBindingData, namespace string) (*rbacv1.RoleBinding, error) {
	subjectsObj := make([]rbacv1.Subject, 1)

	subjectsObj[0] = rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      "test-sa",
		Namespace: "test-ns",
	}
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rolebinding",
			Namespace: "a namespace",
		},
		Subjects: subjectsObj,
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "a role",
		},
	}, nil
}

func (foe BadOpenshiftEngine) GetRoleBinding(ctx context.Context, name, namespace string) (*rbacv1.RoleBinding, error) {
	subjectsObj := make([]rbacv1.Subject, 1)

	subjectsObj[0] = rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      "test-sa",
		Namespace: "test-ns",
	}
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rolebinding",
			Namespace: "a namespace",
		},
		Subjects: subjectsObj,
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "a role",
		},
	}, nil
}

func (foe BadOpenshiftEngine) DeleteRoleBinding(ctx context.Context, name, namespace string) error {
	return nil
}
