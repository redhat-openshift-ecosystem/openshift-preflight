package operator

import (
	"context"
	"errors"
	"testing"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/check"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/operatorsdk"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	imagestreamv1 "github.com/openshift/api/image/v1"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOperator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator Suite")
}

type FakeOperatorSdk struct {
	OperatorSdkReport   operatorsdk.OperatorSdkScorecardReport
	OperatorSdkBVReport operatorsdk.OperatorSdkBundleValidateReport
}

func (f FakeOperatorSdk) BundleValidate(ctx context.Context, image string, opts operatorsdk.OperatorSdkBundleValidateOptions) (*operatorsdk.OperatorSdkBundleValidateReport, error) {
	return &f.OperatorSdkBVReport, nil
}

func (f FakeOperatorSdk) Scorecard(ctx context.Context, image string, opts operatorsdk.OperatorSdkScorecardOptions) (*operatorsdk.OperatorSdkScorecardReport, error) {
	return &f.OperatorSdkReport, nil
}

type BadOperatorSdk struct{}

func (bose BadOperatorSdk) Scorecard(ctx context.Context, bundleImage string, opts operatorsdk.OperatorSdkScorecardOptions) (*operatorsdk.OperatorSdkScorecardReport, error) {
	operatorSdkReport := operatorsdk.OperatorSdkScorecardReport{
		Stdout: "Bad Stdout",
		Stderr: "Bad Stderr",
		Items:  []operatorsdk.OperatorSdkScorecardItem{},
	}
	return &operatorSdkReport, errors.New("the Operator Sdk Scorecard has failed")
}

func (bose BadOperatorSdk) BundleValidate(ctx context.Context, bundleImage string, opts operatorsdk.OperatorSdkBundleValidateOptions) (*operatorsdk.OperatorSdkBundleValidateReport, error) {
	operatorSdkReport := operatorsdk.OperatorSdkBundleValidateReport{
		Stdout:  "Bad Stdout",
		Stderr:  "Bad Stderr",
		Passed:  false,
		Outputs: []operatorsdk.OperatorSdkBundleValidateOutput{},
	}
	return &operatorSdkReport, errors.New("the Operator Sdk Bundle Validate has failed")
}

var pod1 = corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pod1",
		Namespace: "testns",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "cont1",
				Image: "my.container/image/1:latest",
			},
			{
				Name:  "cont2",
				Image: "my.container/image/2:3",
			},
		},
	},
}

var pod2 = corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pod2",
		Namespace: "testns",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "cont3",
				Image: "my.container/image/my3:4",
			},
			{
				Name:  "cont2",
				Image: "my.container/image/2:3",
			},
		},
	},
}

var pods = corev1.PodList{
	Items: []corev1.Pod{
		pod1,
		pod2,
	},
}

var csv = operatorsv1alpha1.ClusterServiceVersion{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "csv-v0.0.0",
		Namespace: "testPackage-target",
	},
	Spec: operatorsv1alpha1.ClusterServiceVersionSpec{},
	Status: operatorsv1alpha1.ClusterServiceVersionStatus{
		Phase: operatorsv1alpha1.CSVPhaseSucceeded,
	},
}

var csvDefault = operatorsv1alpha1.ClusterServiceVersion{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "csv-v0.0.0",
		Namespace: "default",
	},
	Spec: operatorsv1alpha1.ClusterServiceVersionSpec{},
	Status: operatorsv1alpha1.ClusterServiceVersionStatus{
		Phase: operatorsv1alpha1.CSVPhaseSucceeded,
	},
}

var csvMarketplace = operatorsv1alpha1.ClusterServiceVersion{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "csv-v0.0.0",
		Namespace: "openshift-marketplace",
	},
	Spec: operatorsv1alpha1.ClusterServiceVersionSpec{},
	Status: operatorsv1alpha1.ClusterServiceVersionStatus{
		Phase: operatorsv1alpha1.CSVPhaseSucceeded,
	},
}

var ns = corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "test-ns",
	},
}

var secret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "pull-image-secret",
		Namespace: "test-ns",
	},
	Type:       "kubernetes.io/dockerconfigjson",
	StringData: map[string]string{".dockerconfigjson": "secretData"},
}

var sub = operatorsv1alpha1.Subscription{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "testPackage",
		Namespace: "testPackage",
	},
	Status: operatorsv1alpha1.SubscriptionStatus{
		InstalledCSV: "csv-v0.0.0",
	},
}

var og = operatorsv1.OperatorGroup{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "testPackage",
		Namespace: "testPackage",
	},
	Status: operatorsv1.OperatorGroupStatus{
		LastUpdated: nil,
	},
}

var isList = imagestreamv1.ImageStreamList{
	Items: []imagestreamv1.ImageStream{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "imagestream1",
				Namespace: "testns",
			},
			Spec: imagestreamv1.ImageStreamSpec{
				Tags: []imagestreamv1.TagReference{
					{
						From: &corev1.ObjectReference{
							Name: "stream1",
							Kind: "DockerImage",
						},
					},
				},
			},
		},
	},
}

var AssertMetaData = func(check check.Check) {
	Context("When checking metadata", func() {
		Context("The check name should not be empty", func() {
			Expect(check.Name()).ToNot(BeEmpty())
		})

		Context("The metadata keys should not be empty", func() {
			meta := check.Metadata()
			Expect(meta.CheckURL).ToNot(BeEmpty())
			Expect(meta.Description).ToNot(BeEmpty())
			Expect(meta.KnowledgeBaseURL).ToNot(BeEmpty())
			// Level is optional.
		})

		Context("The help text should not be empty", func() {
			help := check.Help()
			Expect(help.Message).ToNot(BeEmpty())
			Expect(help.Suggestion).ToNot(BeEmpty())
		})
	})
}
