package reconcile

import (
	"context"
	"strconv"

	"github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	"github.com/go-logr/logr"
	"go.openly.dev/pointy"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("Misc utils testing", Ordered, func() {
	const (
		CustomSecretName       = "custom-secret"
		CustomUsernameKey      = "usr"
		CustomPasswordKey      = "pwd"
		CustomAstarteName      = "my-astarte"
		CustomAstarteNamespace = "default"
		CustomRabbitMQHost     = "custom-rabbitmq-host"
		CustomRabbitMQPort     = 5673
		CustomVerneMQHost      = "vernemq.example.com"
		CustomVerneMQPort      = 8884
		AstarteVersion         = "1.3.0"
	)

	var cr *v2alpha1.Astarte
	var log logr.Logger

	BeforeAll(func() {
		log = log.WithValues("test", "astarte_data_updater_plant reconcile")
		if CustomAstarteNamespace != "default" {
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: CustomAstarteNamespace,
				},
			}

			Eventually(func() error {
				return k8sClient.Create(context.Background(), ns)
			}, "10s", "250ms").Should(Succeed())
		}
	})

	AfterAll(func() {
		if CustomAstarteNamespace != "default" {
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: CustomAstarteNamespace,
				},
			}
			Expect(k8sClient.Delete(context.Background(), ns)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteNamespace}, &v1.Namespace{})
			}, "10s", "250ms").ShouldNot(Succeed())
		}
	})

	BeforeEach(func() {
		// Create and initialize a basic Astarte CR
		cr = &v2alpha1.Astarte{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CustomAstarteName,
				Namespace: CustomAstarteNamespace,
			},
			Spec: v2alpha1.AstarteSpec{
				Version: AstarteVersion,
				RabbitMQ: v2alpha1.AstarteRabbitMQSpec{
					Connection: &v2alpha1.AstarteRabbitMQConnectionSpec{
						HostAndPort: v2alpha1.HostAndPort{
							Host: CustomRabbitMQHost,
							Port: pointy.Int32(CustomRabbitMQPort),
						},
					},
				},
				VerneMQ: v2alpha1.AstarteVerneMQSpec{
					HostAndPort: v2alpha1.HostAndPort{
						Host: CustomVerneMQHost,
						Port: pointy.Int32(CustomVerneMQPort),
					},
				},
				Cassandra: v2alpha1.AstarteCassandraSpec{
					Connection: &v2alpha1.AstarteCassandraConnectionSpec{
						Nodes: []v2alpha1.HostAndPort{
							{
								Host: "cassandra.example.com",
								Port: pointy.Int32(9042),
							},
						},
					},
				},
				Components: v2alpha1.AstarteComponentsSpec{
					DataUpdaterPlant: v2alpha1.AstarteDataUpdaterPlantSpec{},
				},
			},
		}

		Expect(k8sClient.Create(context.Background(), cr)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
		}, "10s", "250ms").Should(Succeed())
	})

	AfterEach(func() {
		astartes := &v2alpha1.AstarteList{}
		Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, a := range astartes.Items {
			Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &v2alpha1.Astarte{})
			}, "10s", "250ms").ShouldNot(Succeed())
		}

		deployments := &appsv1.DeploymentList{}
		Expect(k8sClient.List(context.Background(), deployments, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, d := range deployments.Items {
			Expect(k8sClient.Delete(context.Background(), &d)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: d.Name, Namespace: d.Namespace}, &appsv1.Deployment{})
			}, "10s", "250ms").ShouldNot(Succeed())
		}
	})

	Describe("Test EnsureAstarteDataUpdaterPlant", func() {
		It("should return error if it is not possible to list current DUP Deployments", func() {
			// To make this fail, we will use a non existing namespace
			brokenCR := cr.DeepCopy()
			brokenCR.Namespace = "non-existing-namespace"

			dup := v2alpha1.AstarteDataUpdaterPlantSpec{
				AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
					Deploy:   pointy.Bool(true),
					Replicas: pointy.Int32(2),
				},
			}

			Expect(EnsureAstarteDataUpdaterPlant(brokenCR, dup, k8sClient, scheme.Scheme)).ToNot(Succeed())
		})

		It("should return nil if the component is not enabled", func() {
			dup := v2alpha1.AstarteDataUpdaterPlantSpec{
				AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
					Deploy: pointy.Bool(false),
				},
			}

			Expect(EnsureAstarteDataUpdaterPlant(cr, dup, k8sClient, scheme.Scheme)).To(Succeed())
		})

		It("should create the right number of DUP deployments", func() {
			// To test this we create 2 DUP deployments, then update the Astarte CR to have only 1 DUP replica and check
			// that only 1 is left
			cr1 := cr.DeepCopy()
			cr1.ResourceVersion = ""
			cr1.Name = "two-replicas-dup"
			cr1.Spec.Components.DataUpdaterPlant.Deploy = pointy.Bool(true)
			cr1.Spec.Components.DataUpdaterPlant.Replicas = pointy.Int32(2)
			dups := &appsv1.DeploymentList{}

			// We should have 2 deployments now
			Expect(k8sClient.Create(context.Background(), cr1)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr1.Name, Namespace: cr1.Namespace}, cr1)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteDataUpdaterPlant(cr1, cr1.Spec.Components.DataUpdaterPlant, k8sClient, scheme.Scheme)).To(Succeed())
			Expect(k8sClient.List(context.Background(), dups, client.InNamespace(cr1.Namespace),
				client.MatchingLabels{"astarte-component": "data-updater-plant"})).To(Succeed())
			Expect(dups.Items).To(HaveLen(2))

			// Update the CR to have only 1 DUP replica
			cr1.Spec.Components.DataUpdaterPlant.Replicas = pointy.Int32(1)
			Expect(k8sClient.Update(context.Background(), cr1)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr1.Name, Namespace: cr1.Namespace}, cr1)
			}, "10s", "250ms").Should(Succeed())

			// We should have only 1 deployment now
			Expect(EnsureAstarteDataUpdaterPlant(cr1, cr1.Spec.Components.DataUpdaterPlant, k8sClient, scheme.Scheme)).To(Succeed())

			Expect(k8sClient.List(context.Background(), dups, client.InNamespace(cr1.Namespace),
				client.MatchingLabels{"astarte-component": "data-updater-plant"})).To(Succeed())
			Expect(dups.Items).To(HaveLen(1))

			// Cleanup
			Expect(k8sClient.Delete(context.Background(), cr1)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr1.Name, Namespace: cr1.Namespace}, &v2alpha1.Astarte{})
			}, "10s", "250ms").ShouldNot(Succeed())
		})
	})

	Describe("Test createIndexedDataUpdaterPlantDeployment", func() {
		It("should create the requested number of deployments with the right labels", func() {
			cr.Spec.Components.DataUpdaterPlant.Deploy = pointy.Bool(true)
			cr.Spec.Components.DataUpdaterPlant.Replicas = pointy.Int32(3)

			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(createIndexedDataUpdaterPlantDeployment(0, 3, cr, cr.Spec.Components.DataUpdaterPlant, k8sClient, scheme.Scheme)).To(Succeed())
			Expect(createIndexedDataUpdaterPlantDeployment(1, 3, cr, cr.Spec.Components.DataUpdaterPlant, k8sClient, scheme.Scheme)).To(Succeed())
			Expect(createIndexedDataUpdaterPlantDeployment(2, 3, cr, cr.Spec.Components.DataUpdaterPlant, k8sClient, scheme.Scheme)).To(Succeed())

			dups := &appsv1.DeploymentList{}
			Expect(k8sClient.List(context.Background(), dups, client.InNamespace(cr.Namespace),
				client.MatchingLabels{"astarte-component": "data-updater-plant"})).To(Succeed())

			Expect(dups.Items).To((HaveLen(3)))
			// Check that the deployments have the right names and labels
			Expect(dups.Items[0].Name).To(Equal(CustomAstarteName + "-data-updater-plant"))
			Expect(dups.Items[1].Name).To(Equal(CustomAstarteName + "-data-updater-plant-1"))
			Expect(dups.Items[2].Name).To(Equal(CustomAstarteName + "-data-updater-plant-2"))

			for i, d := range dups.Items {
				Expect(d.Labels).ToNot(BeNil())
				if i == 0 {
					Expect(d.Labels["app"]).To(Equal(CustomAstarteName + "-data-updater-plant"))
				} else {
					Expect(d.Labels["app"]).To(Equal(CustomAstarteName + "-data-updater-plant-" + strconv.Itoa(i)))
				}
				Expect(d.Labels["component"]).To(Equal("astarte"))
				Expect(d.Labels["astarte-component"]).To(Equal("data-updater-plant"))
				Expect(d.Labels["astarte-instance-name"]).To(Equal(CustomAstarteName))
				// Check that each deployment has exactly one replica
				Expect(*d.Spec.Replicas).To(Equal(int32(1)))
			}
		})
	})
})
