package engine

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	imagestreamv1 "github.com/openshift/api/image/v1"
	operatorv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/internal/cli"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("OpenShift Engine", func() {
	var oe *openshiftEngine

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

		csv := operatorv1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testcsv",
				Namespace: "testns",
			},
		}

		scheme := apiruntime.NewScheme()
		Expect(operatorv1.AddToScheme(scheme)).To(Succeed())
		Expect(operatorv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(imagestreamv1.AddToScheme(scheme)).To(Succeed())
		Expect(rbacv1.AddToScheme(scheme)).To(Succeed())
		cl := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(&csv).
			WithLists(&pods, &isList).
			Build()
		oe = &openshiftEngine{Client: cl}
	})
	Context("Namespaces", func() {
		It("should exercise Namespaces", func() {
			By("creating a Namespace", func() {
				ns, err := oe.CreateNamespace(context.TODO(), "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(ns).ToNot(BeNil())
			})
			By("getting that Namespace", func() {
				ns, err := oe.GetNamespace(context.TODO(), "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(ns).ToNot(BeNil())
			})
			By("deleting that Namespace", func() {
				err := oe.DeleteNamespace(context.TODO(), "testns")
				Expect(err).ToNot(HaveOccurred())
			})
			By("trying to get it again, but failing", func() {
				ns, err := oe.GetNamespace(context.TODO(), "testns")
				Expect(err).To(HaveOccurred())
				Expect(ns).To(BeNil())
			})
		})
	})
	Context("OperatorGroups", func() {
		It("should exercise OperatorGroups", func() {
			operatorGroupData := cli.OperatorGroupData{
				Name:             "testog",
				TargetNamespaces: []string{"default", "testns"},
			}
			By("creating a OperatorGroup", func() {
				og, err := oe.CreateOperatorGroup(context.TODO(), operatorGroupData, "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(og).ToNot(BeNil())
			})
			By("getting that OperatorGroup", func() {
				og, err := oe.GetOperatorGroup(context.TODO(), "testog", "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(og).ToNot(BeNil())
				Expect(og.Spec.TargetNamespaces).To(ContainElement("testns"))
			})
			By("deleting that OperatorGroup", func() {
				err := oe.DeleteOperatorGroup(context.TODO(), "testog", "testns")
				Expect(err).ToNot(HaveOccurred())
			})
			By("trying to get it again, but failing", func() {
				og, err := oe.GetOperatorGroup(context.TODO(), "testog", "testns")
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
				secret, err := oe.CreateSecret(context.TODO(), "testsecret", content, corev1.SecretTypeDockerConfigJson, "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(secret).ToNot(BeNil())
			})
			By("getting that Secret", func() {
				secret, err := oe.GetSecret(context.TODO(), "testsecret", "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(secret).ToNot(BeNil())
				// This wouldn't be a readable field in "normal" client operations. But rather
				// than changing the main code to use Data, we're just asserting that what
				// we sent in is what we get back.
				Expect(secret.StringData).To(ContainElement("testdata"))
			})
			By("deleting that Secret", func() {
				err := oe.DeleteSecret(context.TODO(), "testsecret", "testns")
				Expect(err).ToNot(HaveOccurred())
			})
			By("trying to get it again, but failing", func() {
				secret, err := oe.GetSecret(context.TODO(), "testsecret", "testns")
				Expect(err).To(HaveOccurred())
				Expect(secret).To(BeNil())
			})
		})
	})
	Context("CatalogSources", func() {
		It("should exercise CatalogSources", func() {
			By("creating a CatalogSource", func() {
				csData := cli.CatalogSourceData{
					Name:    "testcs",
					Image:   "this/is/my-image:now",
					Secrets: []string{"my-secrets"},
				}
				cs, err := oe.CreateCatalogSource(context.TODO(), csData, "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(cs).ToNot(BeNil())
			})
			By("getting that CatalogSource", func() {
				cs, err := oe.GetCatalogSource(context.TODO(), "testcs", "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(cs).ToNot(BeNil())
				Expect(cs.Spec.Image).To(Equal("this/is/my-image:now"))
			})
			By("deleting that CatalogSource", func() {
				err := oe.DeleteCatalogSource(context.TODO(), "testcs", "testns")
				Expect(err).ToNot(HaveOccurred())
			})
			By("trying to get it again, but failing", func() {
				cs, err := oe.GetCatalogSource(context.TODO(), "testcs", "testns")
				Expect(err).To(HaveOccurred())
				Expect(cs).To(BeNil())
			})
		})
	})
	Context("Subscriptions", func() {
		It("should exercise Subscriptions", func() {
			By("creating a Subscription", func() {
				subData := cli.SubscriptionData{
					Name:                   "testsub",
					Channel:                "testchannel",
					CatalogSource:          "testcs",
					CatalogSourceNamespace: "testns",
					Package:                "testpackage",
				}
				sub, err := oe.CreateSubscription(context.TODO(), subData, "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(sub).ToNot(BeNil())
			})
			By("getting that Subscription", func() {
				sub, err := oe.GetSubscription(context.TODO(), "testsub", "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(sub).ToNot(BeNil())
				Expect(sub.Spec.Channel).To(Equal("testchannel"))
			})
			By("deleting that Subscription", func() {
				err := oe.DeleteSubscription(context.TODO(), "testsub", "testns")
				Expect(err).ToNot(HaveOccurred())
			})
			By("trying to get it again, but failing", func() {
				sub, err := oe.GetSubscription(context.TODO(), "testsub", "testns")
				Expect(err).To(HaveOccurred())
				Expect(sub).To(BeNil())
			})
		})
	})
	Context("RoleBindings", func() {
		It("should exercise RoleBindings", func() {
			By("creating a RoleBinding", func() {
				rbData := cli.RoleBindingData{
					Name:      "testrb",
					Subjects:  []string{"testsubject"},
					Role:      "testrole",
					Namespace: "testns",
				}
				rb, err := oe.CreateRoleBinding(context.TODO(), rbData, "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(rb).ToNot(BeNil())
			})
			By("getting that RoleBinding", func() {
				rb, err := oe.GetRoleBinding(context.TODO(), "testrb", "testns")
				Expect(err).ToNot(HaveOccurred())
				Expect(rb).ToNot(BeNil())
				Expect(rb.Name).To(Equal("testrb"))
			})
			By("deleting that RoleBinding", func() {
				err := oe.DeleteRoleBinding(context.TODO(), "testrb", "testns")
				Expect(err).ToNot(HaveOccurred())
			})
			By("trying to get it again, but failing", func() {
				rb, err := oe.GetRoleBinding(context.TODO(), "testrb", "testns")
				Expect(err).To(HaveOccurred())
				Expect(rb).To(BeNil())
			})
		})
	})
	Context("Images", func() {
		It("should exercise GetImages", func() {
			images, err := oe.GetImages(context.TODO())
			Expect(err).ToNot(HaveOccurred())
			Expect(images).ToNot(BeNil())
		})
	})
	Context("CSVs", func() {
		It("should exercise GetCSV", func() {
			csv, err := oe.GetCSV(context.TODO(), "testcsv", "testns")
			Expect(err).ToNot(HaveOccurred())
			Expect(csv).ToNot(BeNil())
		})
	})
})
