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

	"github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
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

var _ = Describe("CFSSL testing", Ordered, func() {
	const (
		CustomAstarteName      = "my-astarte"
		CustomAstarteNamespace = "default"
		CustomRabbitMQHost     = "custom-rabbitmq-host"
		CustomRabbitMQPort     = 5673
		CustomVerneMQHost      = "vernemq.example.com"
		CustomVerneMQPort      = 8884
		AstarteVersion         = "1.3.0"
	)

	var cr *v2alpha1.Astarte

	BeforeAll(func() {
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
				AstarteInstanceID: "astarteinstancecfssl",
				CFSSL: v2alpha1.AstarteCFSSLSpec{
					Deploy: pointy.Bool(false),
				},
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
				Components: v2alpha1.AstarteComponentsSpec{},
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

	Describe("Test EnsureCFSSL", func() {
		It("should create/update the CFSSL pod", func() {
			deploymentName := CustomAstarteName + "-cfssl"
			cr.Spec.CFSSL.Deploy = pointy.Bool(true)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())

			// First reconciliation
			Expect(EnsureCFSSL(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Check that the deployment is created
			cfsslDeployment := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: CustomAstarteNamespace}, cfsslDeployment)
			}, "10s", "250ms").Should(Succeed())

			// Store the checksum
			initialChecksum := cfsslDeployment.Spec.Template.Annotations["checksum/config"]
			Expect(initialChecksum).ToNot(BeEmpty())

			// Update the Astarte CR
			cr.Spec.CFSSL.CaExpiry = "1h"
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())

			// Second reconciliation
			Expect(EnsureCFSSL(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Check that the deployment is updated
			Eventually(func() string {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: CustomAstarteNamespace}, cfsslDeployment)
				if err != nil {
					return ""
				}
				return cfsslDeployment.Spec.Template.Annotations["checksum/config"]
			}, "10s", "250ms").ShouldNot(Equal(initialChecksum))
		})
	})
})
