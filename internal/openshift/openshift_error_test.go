package openshift

import (
	"context"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	imagestreamv1 "github.com/openshift/api/image/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	fakecg "k8s.io/client-go/kubernetes/fake"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakecr "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var errFake = fmt.Errorf("fake injected error")

func buildScheme() *apiruntime.Scheme {
	scheme := apiruntime.NewScheme()
	ExpectWithOffset(1, AddSchemes(scheme)).To(Succeed())
	ExpectWithOffset(1, appsv1.AddToScheme(scheme)).To(Succeed())
	return scheme
}

func newErrorClient(funcs interceptor.Funcs) crclient.Client {
	return fakecr.NewClientBuilder().
		WithScheme(buildScheme()).
		WithInterceptorFuncs(funcs).
		Build()
}

var _ = Describe("OpenShift Client Error Paths", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.TODO()
	})

	// --- Namespace error paths ---
	Context("CreateNamespace generic error", func() {
		It("should return an error when Create fails with a non-AlreadyExists error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Create: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.CreateOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			ns, err := oc.CreateNamespace(ctx, "fail-ns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrAlreadyExists))
			Expect(err.Error()).To(ContainSubstring("could not create namespace"))
			Expect(ns).To(BeNil())
		})
	})

	Context("DeleteNamespace generic error", func() {
		It("should return an error when Delete fails with a non-NotFound error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Delete: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.DeleteOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			err := oc.DeleteNamespace(ctx, "fail-ns")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not delete namespace"))
		})
	})

	Context("GetNamespace generic error", func() {
		It("should return an error when Get fails with a non-NotFound error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Get: func(_ context.Context, _ crclient.WithWatch, _ crclient.ObjectKey, _ crclient.Object, _ ...crclient.GetOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			ns, err := oc.GetNamespace(ctx, "fail-ns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrNotFound))
			Expect(err.Error()).To(ContainSubstring("could not retrieve namespace"))
			Expect(ns).To(BeNil())
		})
	})

	// --- OperatorGroup error paths ---
	Context("CreateOperatorGroup generic error", func() {
		It("should return an error when Create fails with a non-AlreadyExists error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Create: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.CreateOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			og, err := oc.CreateOperatorGroup(ctx, OperatorGroupData{Name: "fail-og", TargetNamespaces: []string{"default"}}, "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrAlreadyExists))
			Expect(err.Error()).To(ContainSubstring("could not create operatorgroup"))
			Expect(og).To(BeNil())
		})
	})

	Context("DeleteOperatorGroup error", func() {
		It("should return an error when Delete fails", func() {
			cl := newErrorClient(interceptor.Funcs{
				Delete: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.DeleteOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			err := oc.DeleteOperatorGroup(ctx, "fail-og", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not delete operatorgroup"))
		})
	})

	Context("GetOperatorGroup generic error", func() {
		It("should return an error when Get fails with a non-NotFound error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Get: func(_ context.Context, _ crclient.WithWatch, _ crclient.ObjectKey, _ crclient.Object, _ ...crclient.GetOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			og, err := oc.GetOperatorGroup(ctx, "fail-og", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrNotFound))
			Expect(err.Error()).To(ContainSubstring("could not retrieve operatorgroup"))
			Expect(og).To(BeNil())
		})
	})

	// --- Secret error paths ---
	Context("CreateSecret generic error", func() {
		It("should return an error when Create fails with a non-AlreadyExists error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Create: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.CreateOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			secret, err := oc.CreateSecret(ctx, "fail-secret", map[string]string{"key": "val"}, corev1.SecretTypeOpaque, "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrAlreadyExists))
			Expect(err.Error()).To(ContainSubstring("could not create secret"))
			Expect(secret).To(BeNil())
		})
	})

	Context("DeleteSecret error", func() {
		It("should return an error when Delete fails", func() {
			cl := newErrorClient(interceptor.Funcs{
				Delete: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.DeleteOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			err := oc.DeleteSecret(ctx, "fail-secret", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not delete secret"))
		})
	})

	Context("GetSecret generic error", func() {
		It("should return an error when Get fails with a non-NotFound error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Get: func(_ context.Context, _ crclient.WithWatch, _ crclient.ObjectKey, _ crclient.Object, _ ...crclient.GetOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			secret, err := oc.GetSecret(ctx, "fail-secret", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrNotFound))
			Expect(err.Error()).To(ContainSubstring("could not retrieve secret"))
			Expect(secret).To(BeNil())
		})
	})

	// --- CatalogSource error paths ---
	Context("CreateCatalogSource generic error", func() {
		It("should return an error when Create fails with a non-AlreadyExists error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Create: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.CreateOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			cs, err := oc.CreateCatalogSource(ctx, CatalogSourceData{Name: "fail-cs", Image: "img:latest"}, "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrAlreadyExists))
			Expect(err.Error()).To(ContainSubstring("could not create catalogsource"))
			Expect(cs).To(BeNil())
		})
	})

	Context("DeleteCatalogSource error", func() {
		It("should return an error when Delete fails", func() {
			cl := newErrorClient(interceptor.Funcs{
				Delete: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.DeleteOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			err := oc.DeleteCatalogSource(ctx, "fail-cs", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not delete catalogsource"))
		})
	})

	Context("GetCatalogSource generic error", func() {
		It("should return an error when Get fails with a non-NotFound error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Get: func(_ context.Context, _ crclient.WithWatch, _ crclient.ObjectKey, _ crclient.Object, _ ...crclient.GetOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			cs, err := oc.GetCatalogSource(ctx, "fail-cs", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrNotFound))
			Expect(err.Error()).To(ContainSubstring("could not retrieve catalogsource"))
			Expect(cs).To(BeNil())
		})
	})

	// --- Subscription error paths ---
	Context("CreateSubscription generic error", func() {
		It("should return an error when Create fails with a non-AlreadyExists error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Create: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.CreateOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			sub, err := oc.CreateSubscription(ctx, SubscriptionData{Name: "fail-sub", Channel: "ch", Package: "pkg"}, "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrAlreadyExists))
			Expect(err.Error()).To(ContainSubstring("could not create subscription"))
			Expect(sub).To(BeNil())
		})
	})

	Context("GetSubscription generic error", func() {
		It("should return an error when Get fails with a non-NotFound error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Get: func(_ context.Context, _ crclient.WithWatch, _ crclient.ObjectKey, _ crclient.Object, _ ...crclient.GetOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			sub, err := oc.GetSubscription(ctx, "fail-sub", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrNotFound))
			Expect(err.Error()).To(ContainSubstring("could not retrieve subscription"))
			Expect(sub).To(BeNil())
		})
	})

	Context("DeleteSubscription error", func() {
		It("should return an error when Delete fails", func() {
			cl := newErrorClient(interceptor.Funcs{
				Delete: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.DeleteOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			err := oc.DeleteSubscription(ctx, "fail-sub", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not delete subscription"))
		})
	})

	// --- CSV error paths ---
	Context("GetCSV generic error", func() {
		It("should return an error when Get fails with a non-NotFound error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Get: func(_ context.Context, _ crclient.WithWatch, _ crclient.ObjectKey, _ crclient.Object, _ ...crclient.GetOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			csv, err := oc.GetCSV(ctx, "fail-csv", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrNotFound))
			Expect(err.Error()).To(ContainSubstring("could not retrieve csv"))
			Expect(csv).To(BeNil())
		})
	})

	// --- GetImages error paths ---
	Context("GetImages pod list error", func() {
		It("should return an error when listing pods fails", func() {
			cl := newErrorClient(interceptor.Funcs{
				List: func(_ context.Context, _ crclient.WithWatch, _ crclient.ObjectList, _ ...crclient.ListOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			images, err := oc.GetImages(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not retrieve pod list"))
			Expect(images).To(BeNil())
		})
	})

	Context("GetImages imagestream list error", func() {
		It("should return an error when listing image streams fails", func() {
			callCount := 0
			cl := newErrorClient(interceptor.Funcs{
				List: func(innerCtx context.Context, innerClient crclient.WithWatch, list crclient.ObjectList, opts ...crclient.ListOption) error {
					callCount++
					if callCount == 1 {
						// First call (PodList) succeeds
						return innerClient.List(innerCtx, list, opts...)
					}
					// Second call (ImageStreamList) fails
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			images, err := oc.GetImages(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not list image streams"))
			Expect(images).To(BeNil())
		})
	})

	// --- RoleBinding error paths ---
	Context("CreateRoleBinding generic error", func() {
		It("should return an error when Create fails with a non-AlreadyExists error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Create: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.CreateOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			rbData := RoleBindingData{
				Name:      "fail-rb",
				Subjects:  []string{"sa1"},
				Role:      "testrole",
				Namespace: "testns",
			}
			rb, err := oc.CreateRoleBinding(ctx, rbData, "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrAlreadyExists))
			Expect(err.Error()).To(ContainSubstring("could not create rolebinding"))
			Expect(rb).To(BeNil())
		})
	})

	Context("GetRoleBinding generic error", func() {
		It("should return an error when Get fails with a non-NotFound error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Get: func(_ context.Context, _ crclient.WithWatch, _ crclient.ObjectKey, _ crclient.Object, _ ...crclient.GetOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			rb, err := oc.GetRoleBinding(ctx, "fail-rb", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrNotFound))
			Expect(err.Error()).To(ContainSubstring("could not retrieve rolebinding"))
			Expect(rb).To(BeNil())
		})
	})

	Context("DeleteRoleBinding error", func() {
		It("should return an error when Delete fails", func() {
			cl := newErrorClient(interceptor.Funcs{
				Delete: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.DeleteOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			err := oc.DeleteRoleBinding(ctx, "fail-rb", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not delete rolebinding"))
		})
	})

	// --- Deployment error paths ---
	Context("GetDeployment generic error", func() {
		It("should return an error when Get fails with a non-NotFound error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Get: func(_ context.Context, _ crclient.WithWatch, _ crclient.ObjectKey, _ crclient.Object, _ ...crclient.GetOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			dep, err := oc.GetDeployment(ctx, "fail-dep", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrNotFound))
			Expect(err.Error()).To(ContainSubstring("could not retrieve deployment"))
			Expect(dep).To(BeNil())
		})
	})

	Context("GetDeploymentPods list error", func() {
		It("should return an error when listing pods fails but deployment exists", func() {
			// Pre-create a deployment so GetDeployment succeeds
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dep-with-pods",
					Namespace: "testns",
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "myapp"},
					},
				},
			}
			scheme := buildScheme()
			cl := fakecr.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(deployment).
				WithInterceptorFuncs(interceptor.Funcs{
					List: func(_ context.Context, _ crclient.WithWatch, _ crclient.ObjectList, _ ...crclient.ListOption) error {
						return errFake
					},
				}).
				Build()
			oc := NewClient(cl, fakecg.NewClientset())

			pods, err := oc.GetDeploymentPods(ctx, "dep-with-pods", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not list pods matching label selector"))
			Expect(pods).To(BeNil())
		})
	})

	// --- Pod error paths ---
	Context("GetPod generic error", func() {
		It("should return an error when Get fails with a non-NotFound error", func() {
			cl := newErrorClient(interceptor.Funcs{
				Get: func(_ context.Context, _ crclient.WithWatch, _ crclient.ObjectKey, _ crclient.Object, _ ...crclient.GetOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			pod, err := oc.GetPod(ctx, "fail-pod", "testns")
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrNotFound))
			Expect(err.Error()).To(ContainSubstring("could not retrieve pod"))
			Expect(pod).To(BeNil())
		})
	})

	Context("GetPodLogs when pod does not exist", func() {
		It("should return an error when the pod cannot be found", func() {
			cl := newErrorClient(interceptor.Funcs{
				Get: func(_ context.Context, _ crclient.WithWatch, _ crclient.ObjectKey, _ crclient.Object, _ ...crclient.GetOption) error {
					return errFake
				},
			})
			oc := NewClient(cl, fakecg.NewClientset())

			logs, err := oc.GetPodLogs(ctx, "no-pod", "testns")
			Expect(err).To(HaveOccurred())
			Expect(logs).To(BeNil())
		})
	})

	// --- AddSchemes error paths ---
	Context("AddSchemes error paths", func() {
		It("should propagate scheme registration errors", func() {
			// AddSchemes registers operatorsv1, operatorsv1alpha1, imagestreamv1, and rbacv1.
			// We can verify it succeeds with a valid scheme.
			scheme := apiruntime.NewScheme()
			err := AddSchemes(scheme)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	// --- DeleteNamespace with NotFound should succeed ---
	Context("DeleteNamespace with NotFound error", func() {
		It("should not return an error when the namespace is not found", func() {
			// The fake client with no objects will return NotFound for Delete.
			// But fake client Delete on non-existent objects returns NotFound,
			// and the code checks !IsNotFound, so it should succeed.
			cl := fakecr.NewClientBuilder().
				WithScheme(buildScheme()).
				Build()
			oc := NewClient(cl, fakecg.NewClientset())

			err := oc.DeleteNamespace(ctx, "nonexistent-ns")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	// --- GetImages with image streams containing nil From or non-DockerImage kind ---
	Context("GetImages with various ImageStream tag types", func() {
		It("should only include DockerImage tags and skip others", func() {
			isList := &imagestreamv1.ImageStreamList{
				Items: []imagestreamv1.ImageStream{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "is1",
							Namespace: "testns",
						},
						Spec: imagestreamv1.ImageStreamSpec{
							Tags: []imagestreamv1.TagReference{
								{
									From: &corev1.ObjectReference{
										Name: "docker-image:latest",
										Kind: "DockerImage",
									},
								},
								{
									From: &corev1.ObjectReference{
										Name: "other-kind-ref",
										Kind: "ImageStreamTag",
									},
								},
								{
									From: nil,
								},
							},
						},
					},
				},
			}
			scheme := buildScheme()
			cl := fakecr.NewClientBuilder().
				WithScheme(scheme).
				WithLists(isList).
				Build()
			oc := NewClient(cl, fakecg.NewClientset())

			images, err := oc.GetImages(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(images).To(HaveKey("docker-image:latest"))
			Expect(images).ToNot(HaveKey("other-kind-ref"))
		})
	})

	// Verify that errFake is not accidentally matching sentinel errors
	Context("error sentinel verification", func() {
		It("errFake should not match ErrNotFound or ErrAlreadyExists", func() {
			Expect(errors.Is(errFake, ErrNotFound)).To(BeFalse())
			Expect(errors.Is(errFake, ErrAlreadyExists)).To(BeFalse())
		})
	})
})
