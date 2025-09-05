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
package v2alpha1

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Astarte types testing", Ordered, func() {
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

	var cr *Astarte
	var log logr.Logger

	BeforeAll(func() {
		log = log.WithValues("test", "astarte_types.go")
		log.Info("Starting controllerutils tests")
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
			astartes := &AstarteList{}
			Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())

			for _, a := range astartes.Items {
				Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())
			}

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteNamespace}, &v1.Namespace{})
			}, "10s", "250ms").ShouldNot(Succeed())
		}
	})

	BeforeEach(func() {
		// Create and initialize a basic Astarte CR
		cr = &Astarte{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CustomAstarteName,
				Namespace: CustomAstarteNamespace,
			},
			Spec: AstarteSpec{
				Version: AstarteVersion,
				RabbitMQ: AstarteRabbitMQSpec{
					Connection: &AstarteRabbitMQConnectionSpec{
						HostAndPort: HostAndPort{
							Host: CustomRabbitMQHost,
							Port: pointy.Int32(CustomRabbitMQPort),
						},
					},
				},
				VerneMQ: AstarteVerneMQSpec{
					HostAndPort: HostAndPort{
						Host: CustomVerneMQHost,
						Port: pointy.Int32(CustomVerneMQPort),
					},
				},
			},
		}

		Expect(k8sClient.Create(context.Background(), cr)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
		}, "10s", "250ms").Should(Succeed())
	})

	AfterEach(func() {
		astartes := &AstarteList{}
		Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, a := range astartes.Items {
			Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &Astarte{})
			}, "10s", "250ms").ShouldNot(Succeed())
		}

	})

	Describe("Test AstartePodPriorities.IsEnabled()", func() {
		It("should return true if AstartePodPrioritiesSpec is enabled", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{
				Enable: true,
			}

			enabled := cr.Spec.Features.AstartePodPriorities.IsEnabled()
			Expect(enabled).To(BeTrue())
		})
		It("should return false if AstartePodPrioritiesSpec is disabled", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{
				Enable: false,
			}
			disabled := cr.Spec.Features.AstartePodPriorities.IsEnabled()
			Expect(disabled).To(BeFalse())
		})
		It("should return false if AstartePodPrioritiesSpec is nil", func() {
			cr.Spec.Features.AstartePodPriorities = nil
			disabled := cr.Spec.Features.AstartePodPriorities.IsEnabled()
			Expect(disabled).To(BeFalse())
		})
	})

	Describe("Test CFSSL.GetPodLabels()", func() {
		It("should return a map with the correct pod labels", func() {
			// Set some labels to AstarteCFSSLSpec
			cr.Spec.CFSSL = AstarteCFSSLSpec{
				PodLabels: map[string]string{
					"cfssl-label": "cfssl",
				},
			}
			// Check the returned map
			labels := cr.Spec.CFSSL.GetPodLabels()
			Expect(labels).ToNot(BeNil())
			Expect(labels).To(HaveKeyWithValue("cfssl-label", "cfssl"))
		})
	})
})
