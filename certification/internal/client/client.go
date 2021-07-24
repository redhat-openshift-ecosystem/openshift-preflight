package client

import (
	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	scheme                   = runtime.NewScheme()
	operatorV1SchemeGV       = schema.GroupVersion{Group: "operators.coreos.com", Version: "v1"}
	OperatorV1Alpha1SchemeGV = schema.GroupVersion{Group: "operators.coreos.com", Version: "v1alpha1"}
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(operatorV1SchemeGV,
		&operatorv1.OperatorGroup{},
		&operatorv1.OperatorGroupList{},
	)
	metav1.AddToGroupVersion(scheme, operatorV1SchemeGV)

	scheme.AddKnownTypes(OperatorV1Alpha1SchemeGV,
		&operatorv1alpha1.CatalogSource{},
		&operatorv1alpha1.CatalogSourceList{},
	)

	scheme.AddKnownTypes(OperatorV1Alpha1SchemeGV,
		&operatorv1alpha1.Subscription{},
		&operatorv1alpha1.SubscriptionList{},
	)
	metav1.AddToGroupVersion(scheme, OperatorV1Alpha1SchemeGV)

	return nil
}
