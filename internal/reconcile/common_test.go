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
	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Common reconcile testing", Ordered, func() {
	const (
		CustomAstarteName      = "my-astarte"
		CustomAstarteNamespace = "common-test"
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
			}, Timeout, Interval).Should(Succeed())
		}
	})

	AfterAll(func() {
		if CustomAstarteNamespace != "default" {
			astartes := &apiv2alpha1.AstarteList{}
			Expect(k8sClient.List(context.Background(), astartes, client.InNamespace(CustomAstarteNamespace))).To(Succeed())
			for _, a := range astartes.Items {
				_ = k8sClient.Delete(context.Background(), &a)
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &apiv2alpha1.Astarte{})
				}, Timeout, Interval).ShouldNot(Succeed())
			}
			_ = k8sClient.Delete(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}})
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
				AstarteInstanceID: "astarteinstancecommon",
				Version:           AstarteVersion,
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
		}, Timeout, Interval).Should(Succeed())
	})

	AfterEach(func() {
		astartes := &v2alpha1.AstarteList{}
		Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, a := range astartes.Items {
			Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &v2alpha1.Astarte{})
			}, Timeout, Interval).ShouldNot(Succeed())
		}

		deployments := &appsv1.DeploymentList{}
		Expect(k8sClient.List(context.Background(), deployments, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, d := range deployments.Items {
			Expect(k8sClient.Delete(context.Background(), &d)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: d.Name, Namespace: d.Namespace}, &appsv1.Deployment{})
			}, Timeout, Interval).ShouldNot(Succeed())
		}
	})

	Describe("Test EnsureHousekeepingKey", func() {
		It("Should create a valid Housekeeping keypair", func() {
			// Ensure the housekeeping key is created
			Expect(EnsureHousekeepingKey(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Check that the housekeeping keypair secrets are present
			secret_private := &v1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      CustomAstarteName + "-housekeeping-private-key",
					Namespace: CustomAstarteNamespace,
				}, secret_private)
			}, Timeout, Interval).Should(Succeed())
			Expect(secret_private.Data).To(HaveKey("private-key"))

			secret_public := &v1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      CustomAstarteName + "-housekeeping-public-key",
					Namespace: CustomAstarteNamespace,
				}, secret_public)
			}, Timeout, Interval).Should(Succeed())
			Expect(secret_public.Data).To(HaveKey("public-key"))
		})

		Describe("Test EnsureGenericErlangConfiguration", func() {
			It("should create the Generic Erlang Configuration ConfigMap", func() {
				Expect(EnsureGenericErlangConfiguration(cr, k8sClient, scheme.Scheme)).To(Succeed())

				cm := &v1.ConfigMap{}
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      CustomAstarteName + "-generic-erlang-configuration",
						Namespace: CustomAstarteNamespace,
					}, cm)
				}, Timeout, Interval).Should(Succeed())
				Expect(cm.Data["vm.args"]).To(ContainSubstring("-name ${RELEASE_NAME}@${MY_POD_IP}"))
			})
		})

		Describe("Test EnsureErlangClusteringCookie", func() {
			It("should create the Erlang Clustering Cookie secret", func() {
				Expect(EnsureErlangClusteringCookie(cr, k8sClient, scheme.Scheme)).To(Succeed())

				secret := &v1.Secret{}
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      CustomAstarteName + "-erlang-clustering-cookie",
						Namespace: CustomAstarteNamespace,
					}, secret)
				}, Timeout, Interval).Should(Succeed())
				Expect(secret.Data["erlang-cookie"]).ToNot(BeEmpty())
			})
		})

		Describe("Test GetAstarteClusteredServicePolicyRules", func() {
			It("should return the correct PolicyRules for a clustered Astarte service", func() {
				rules := GetAstarteClusteredServicePolicyRules()
				Expect(rules).ToNot(BeNil())
				Expect(rules).To(HaveLen(1))
				Expect(rules[0].APIGroups).To(Equal([]string{""}))
				Expect(rules[0].Resources).To(Equal([]string{"pods", "endpoints"}))
				Expect(rules[0].Verbs).To(Equal([]string{"list", "get"}))
			})
		})
	})
})
