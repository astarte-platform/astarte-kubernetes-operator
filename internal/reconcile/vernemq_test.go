/*
This file is part of Astarte.

Copyright 2020-25 SECO Mind Srl.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

//nolint:goconst
package reconcile

import (
	"context"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	"go.openly.dev/pointy"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("VerneMQ testing", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "my-astarte"
		CustomAstarteNamespace = "vernemq-test"
		CustomRabbitMQHost     = "custom-rabbitmq-host"
		CustomRabbitMQPort     = 5673
		CustomVerneMQHost      = "vernemq.example.com"
		CustomVerneMQPort      = 8884
		AstarteVersion         = "1.3.0"
		DefaultTimeout         = "10s"
		DefaultPollingInterval = "250ms"
	)

	var cr *apiv2alpha1.Astarte

	BeforeAll(func() {
		if CustomAstarteNamespace != "default" {
			ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}}
			Eventually(func() error {
				err := k8sClient.Create(context.Background(), ns)
				if apierrors.IsAlreadyExists(err) {
					return nil
				}
				return err
			}, DefaultTimeout, DefaultPollingInterval).Should(Succeed())
		}
	})

	AfterAll(func() {
		if CustomAstarteNamespace != "default" {
			astartes := &apiv2alpha1.AstarteList{}
			Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
			for _, a := range astartes.Items {
				_ = k8sClient.Delete(context.Background(), &a)
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &apiv2alpha1.Astarte{})
				}, DefaultTimeout, DefaultPollingInterval).ShouldNot(Succeed())
			}
			_ = k8sClient.Delete(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}})
		}
	})
	BeforeEach(func() {
		// Create and initialize a basic Astarte CR
		cr = &apiv2alpha1.Astarte{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CustomAstarteName,
				Namespace: CustomAstarteNamespace,
			},
			Spec: apiv2alpha1.AstarteSpec{
				AstarteInstanceID: "astarteinstancevernemq",
				Version:           AstarteVersion,
				RabbitMQ: apiv2alpha1.AstarteRabbitMQSpec{
					Connection: &apiv2alpha1.AstarteRabbitMQConnectionSpec{
						HostAndPort: apiv2alpha1.HostAndPort{
							Host: CustomRabbitMQHost,
							Port: pointy.Int32(CustomRabbitMQPort),
						},
					},
				},
				VerneMQ: apiv2alpha1.AstarteVerneMQSpec{
					HostAndPort: apiv2alpha1.HostAndPort{
						Host: CustomVerneMQHost,
						Port: pointy.Int32(CustomVerneMQPort),
					},
				},
				Cassandra: apiv2alpha1.AstarteCassandraSpec{
					Connection: &apiv2alpha1.AstarteCassandraConnectionSpec{
						Nodes: []apiv2alpha1.HostAndPort{
							{
								Host: "cassandra.example.com",
								Port: pointy.Int32(9042),
							},
						},
					},
				},
				Components: apiv2alpha1.AstarteComponentsSpec{},
			},
		}

		Expect(k8sClient.Create(context.Background(), cr)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
		}, DefaultTimeout, DefaultPollingInterval).Should(Succeed())
	})

	AfterEach(func() {
		astartes := &apiv2alpha1.AstarteList{}
		Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, a := range astartes.Items {
			Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &apiv2alpha1.Astarte{})
			}, DefaultTimeout, DefaultPollingInterval).ShouldNot(Succeed())
		}

		deployments := &appsv1.DeploymentList{}
		Expect(k8sClient.List(context.Background(), deployments, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, d := range deployments.Items {
			Expect(k8sClient.Delete(context.Background(), &d)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: d.Name, Namespace: d.Namespace}, &appsv1.Deployment{})
			}, DefaultTimeout, DefaultPollingInterval).ShouldNot(Succeed())
		}

		Eventually(func() int {
			list := &apiv2alpha1.AstarteList{}
			if err := k8sClient.List(context.Background(), list, &client.ListOptions{Namespace: CustomAstarteNamespace}); err != nil {
				return -1
			}
			return len(list.Items)
		}, DefaultTimeout, DefaultPollingInterval).Should(Equal(0))
	})

	Describe("Test EnsureVerneMQ", func() {
		It("should create/update the VerneMQ StatefulSet", func() {
			Expect(EnsureVerneMQ(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Check that the VerneMQ statefulSet is created
			statefulSetName := GetVerneMQStatefulSetName(cr)
			Expect(statefulSetName).To(Equal(CustomAstarteName + "-vernemq"))
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: statefulSetName, Namespace: cr.Namespace}, &appsv1.StatefulSet{})
			}, DefaultTimeout, DefaultPollingInterval).Should(Succeed())

			// Update the VerneMQ spec to use a different image
			originalStatefulSet := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: statefulSetName, Namespace: cr.Namespace}, originalStatefulSet)).To(Succeed())
			cr.Spec.VerneMQ.Image = "vernemq/vernemq:0.12.3"
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())

			// update astarte cr.vernemq.deploy to true to force reconciliation
			cr.Spec.VerneMQ.Deploy = pointy.Bool(true)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())

			// Reconcile again
			Expect(EnsureVerneMQ(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Check that the VerneMQ statefulSet is updated
			updatedStatefulSet := &appsv1.StatefulSet{}
			Eventually(func() string {
				if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: statefulSetName, Namespace: cr.Namespace}, updatedStatefulSet); err != nil {
					return ""
				}
				if len(updatedStatefulSet.Spec.Template.Spec.Containers) == 0 {
					return ""
				}
				return updatedStatefulSet.Spec.Template.Spec.Containers[0].Image
			}, DefaultTimeout, DefaultPollingInterval).Should(Equal("vernemq/vernemq:0.12.3"))

			// Now, set cr.spec.vernemq.deploy to false and ensure the statefulSet is deleted
			cr.Spec.VerneMQ.Deploy = pointy.Bool(false)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Expect(EnsureVerneMQ(cr, k8sClient, scheme.Scheme)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: statefulSetName, Namespace: cr.Namespace}, &appsv1.StatefulSet{})
			}, DefaultTimeout, DefaultPollingInterval).ShouldNot(Succeed())

		})
	})
})
