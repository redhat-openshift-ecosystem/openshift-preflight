package openshift

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	imagestreamv1 "github.com/openshift/api/image/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("OpenShift Engine", func() {
	var oc Client

	BeforeEach(func() {
		pod1 := corev1.Pod{
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

		pod2 := corev1.Pod{
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

		pods := corev1.PodList{
			Items: []corev1.Pod{
				pod1,
				pod2,
			},
		}

		isList := imagestreamv1.ImageStreamList{
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

		csv := operatorsv1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testcsv",
				Namespace: "testns",
			},
		}

		scheme := apiruntime.NewScheme()
		Expect(AddSchemes(scheme)).To(Succeed())
		cl := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(&csv).
			WithLists(&pods, &isList).
			Build()
		oc = NewClient(cl)
	})
	Context("Namespaces", func() {
		It("should exercise Namespaces", func() {
			By("creating a Namespace", func() {
				ns, err := oc.CreateNamespace(context.TODO(), "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(ns).ToNot(BeNil())
			})
			By("creating it again should error", func() {
				ns, err := oc.CreateNamespace(context.TODO(), "testns")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrAlreadyExists))
				Expect(ns).ToNot(BeNil())
			})
			By("getting that Namespace", func() {
				ns, err := oc.GetNamespace(context.TODO(), "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(ns).ToNot(BeNil())
			})
			By("deleting that Namespace", func() {
				err := oc.DeleteNamespace(context.TODO(), "testns")
				Expect(err).ToNot(HaveOccurred())
			})
			By("trying to get it again, but failing", func() {
				ns, err := oc.GetNamespace(context.TODO(), "testns")
				Expect(err).To(HaveOccurred())
				Expect(ns).To(BeNil())
			})
		})
	})
	Context("OperatorGroups", func() {
		It("should exercise OperatorGroups", func() {
			operatorGroupData := OperatorGroupData{
				Name:             "testog",
				TargetNamespaces: []string{"default", "testns"},
			}
			By("creating a OperatorGroup", func() {
				og, err := oc.CreateOperatorGroup(context.TODO(), operatorGroupData, "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(og).ToNot(BeNil())
			})
			By("creating it again should error", func() {
				og, err := oc.CreateOperatorGroup(context.TODO(), operatorGroupData, "testns")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrAlreadyExists))
				Expect(og).ToNot(BeNil())
			})
			By("getting that OperatorGroup", func() {
				og, err := oc.GetOperatorGroup(context.TODO(), "testog", "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(og).ToNot(BeNil())
				Expect(og.Spec.TargetNamespaces).To(ContainElement("testns"))
			})
			By("deleting that OperatorGroup", func() {
				err := oc.DeleteOperatorGroup(context.TODO(), "testog", "testns")
				Expect(err).ToNot(HaveOccurred())
			})
			By("trying to get it again, but failing", func() {
				og, err := oc.GetOperatorGroup(context.TODO(), "testog", "testns")
				Expect(err).To(HaveOccurred())
				Expect(og).To(BeNil())
			})
		})
	})
	Context("Secrets", func() {
		It("should exercise Secrets", func() {
			By("creating a Secret", func() {
				content := make(map[string]string, 1)
				content["test"] = "testdata"
				secret, err := oc.CreateSecret(context.TODO(), "testsecret", content, corev1.SecretTypeDockerConfigJson, "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(secret).ToNot(BeNil())
			})
			By("creating a Secret again should error", func() {
				content := make(map[string]string, 1)
				content["test"] = "testdata"
				secret, err := oc.CreateSecret(context.TODO(), "testsecret", content, corev1.SecretTypeDockerConfigJson, "testns")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrAlreadyExists))
				Expect(secret).ToNot(BeNil())
			})
			By("getting that Secret", func() {
				secret, err := oc.GetSecret(context.TODO(), "testsecret", "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(secret).ToNot(BeNil())
				// This wouldn't be a readable field in "normal" client operations. But rather
				// than changing the main code to use Data, we're just asserting that what
				// we sent in is what we get back.
				Expect(secret.StringData).To(ContainElement("testdata"))
			})
			By("deleting that Secret", func() {
				err := oc.DeleteSecret(context.TODO(), "testsecret", "testns")
				Expect(err).ToNot(HaveOccurred())
			})
			By("trying to get it again, but failing", func() {
				secret, err := oc.GetSecret(context.TODO(), "testsecret", "testns")
				Expect(err).To(HaveOccurred())
				Expect(secret).To(BeNil())
			})
		})
	})
	Context("CatalogSources", func() {
		It("should exercise CatalogSources", func() {
			By("creating a CatalogSource", func() {
				csData := CatalogSourceData{
					Name:    "testcs",
					Image:   "this/is/my-image:now",
					Secrets: []string{"my-secrets"},
				}
				cs, err := oc.CreateCatalogSource(context.TODO(), csData, "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(cs).ToNot(BeNil())
			})
			By("creating a CatalogSource again should error", func() {
				csData := CatalogSourceData{
					Name:    "testcs",
					Image:   "this/is/my-image:now",
					Secrets: []string{"my-secrets"},
				}
				cs, err := oc.CreateCatalogSource(context.TODO(), csData, "testns")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrAlreadyExists))
				Expect(cs).ToNot(BeNil())
			})
			By("getting that CatalogSource", func() {
				cs, err := oc.GetCatalogSource(context.TODO(), "testcs", "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(cs).ToNot(BeNil())
				Expect(cs.Spec.Image).To(Equal("this/is/my-image:now"))
			})
			By("deleting that CatalogSource", func() {
				err := oc.DeleteCatalogSource(context.TODO(), "testcs", "testns")
				Expect(err).ToNot(HaveOccurred())
			})
			By("trying to get it again, but failing", func() {
				cs, err := oc.GetCatalogSource(context.TODO(), "testcs", "testns")
				Expect(err).To(HaveOccurred())
				Expect(cs).To(BeNil())
			})
		})
	})
	Context("Subscriptions", func() {
		It("should exercise Subscriptions", func() {
			By("creating a Subscription", func() {
				subData := SubscriptionData{
					Name:                   "testsub",
					Channel:                "testchannel",
					CatalogSource:          "testcs",
					CatalogSourceNamespace: "testns",
					Package:                "testpackage",
				}
				sub, err := oc.CreateSubscription(context.TODO(), subData, "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(sub).ToNot(BeNil())
			})
			By("creating a Subscription again should error", func() {
				subData := SubscriptionData{
					Name:                   "testsub",
					Channel:                "testchannel",
					CatalogSource:          "testcs",
					CatalogSourceNamespace: "testns",
					Package:                "testpackage",
				}
				sub, err := oc.CreateSubscription(context.TODO(), subData, "testns")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrAlreadyExists))
				Expect(sub).ToNot(BeNil())
			})
			By("getting that Subscription", func() {
				sub, err := oc.GetSubscription(context.TODO(), "testsub", "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(sub).ToNot(BeNil())
				Expect(sub.Spec.Channel).To(Equal("testchannel"))
			})
			By("deleting that Subscription", func() {
				err := oc.DeleteSubscription(context.TODO(), "testsub", "testns")
				Expect(err).ToNot(HaveOccurred())
			})
			By("trying to get it again, but failing", func() {
				sub, err := oc.GetSubscription(context.TODO(), "testsub", "testns")
				Expect(err).To(HaveOccurred())
				Expect(sub).To(BeNil())
			})
		})
	})
	Context("RoleBindings", func() {
		It("should exercise RoleBindings", func() {
			By("creating a RoleBinding", func() {
				rbData := RoleBindingData{
					Name:      "testrb",
					Subjects:  []string{"testsubject"},
					Role:      "testrole",
					Namespace: "testns",
				}
				rb, err := oc.CreateRoleBinding(context.TODO(), rbData, "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(rb).ToNot(BeNil())
			})
			By("creating a RoleBinding again should error", func() {
				rbData := RoleBindingData{
					Name:      "testrb",
					Subjects:  []string{"testsubject"},
					Role:      "testrole",
					Namespace: "testns",
				}
				rb, err := oc.CreateRoleBinding(context.TODO(), rbData, "testns")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrAlreadyExists))
				Expect(rb).ToNot(BeNil())
			})
			By("getting that RoleBinding", func() {
				rb, err := oc.GetRoleBinding(context.TODO(), "testrb", "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(rb).ToNot(BeNil())
				Expect(rb.Name).To(Equal("testrb"))
			})
			By("deleting that RoleBinding", func() {
				err := oc.DeleteRoleBinding(context.TODO(), "testrb", "testns")
				Expect(err).ToNot(HaveOccurred())
			})
			By("trying to get it again, but failing", func() {
				rb, err := oc.GetRoleBinding(context.TODO(), "testrb", "testns")
				Expect(err).To(HaveOccurred())
				Expect(rb).To(BeNil())
			})
		})
	})
	Context("Images", func() {
		It("should exercise GetImages", func() {
			images, err := oc.GetImages(context.TODO())
			Expect(err).ToNot(HaveOccurred())
			Expect(images).ToNot(BeNil())
		})
	})
	Context("CSVs", func() {
		It("should get a CSV", func() {
			csv, err := oc.GetCSV(context.TODO(), "testcsv", "testns")
			Expect(err).ToNot(HaveOccurred())
			Expect(csv).ToNot(BeNil())
		})
		It("should error if CSV doesn't exist", func() {
			csv, err := oc.GetCSV(context.TODO(), "badcsv", "badns")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrNotFound))
			Expect(csv).To(BeNil())
		})
	})
})
