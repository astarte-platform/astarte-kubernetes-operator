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
package controllerutils

import (
	"context"

	"github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("controllerutils tests", Ordered, Serial, func() {
	const (
		CustomSecretName       = "custom-secret"
		CustomUsernameKey      = "usr"
		CustomPasswordKey      = "pwd"
		CustomAstarteName      = "my-astarte"
		CustomAstarteNamespace = "astarte-controllerutils-tests"
		CustomRabbitMQHost     = "custom-rabbitmq-host"
		CustomRabbitMQPort     = 5673
		CustomVerneMQHost      = "vernemq.example.com"
		CustomVerneMQPort      = 8884
		AstarteVersion         = "1.3.0"
	)

	var cr *v2alpha1.Astarte
	var log logr.Logger

	BeforeAll(func() {
		log = logr.Discard()
		log.Info("Starting controllerutils tests")
		if CustomAstarteNamespace != "default" {
			ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}}
			Eventually(func() error {
				err := k8sClient.Create(context.Background(), ns)
				if apierrors.IsAlreadyExists(err) {
					return nil
				}
				return err
			}, "10s", "250ms").Should(Succeed())
		}
	})

	AfterAll(func() {
		if CustomAstarteNamespace != "default" {
			astartes := &v2alpha1.AstarteList{}
			Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
			for _, a := range astartes.Items {
				_ = k8sClient.Delete(context.Background(), &a)
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &v2alpha1.Astarte{})
				}, "10s", "250ms").ShouldNot(Succeed())
			}
			// Do not delete the namespace here to avoid 'NamespaceTerminating' flakiness in subsequent specs
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

			// If finalizers block deletion in envtest, remove them and retry
			nn := types.NamespacedName{Name: a.Name, Namespace: a.Namespace}
			Eventually(func() error {
				current := &v2alpha1.Astarte{}
				if err := k8sClient.Get(context.Background(), nn, current); err != nil {
					if errors.IsNotFound(err) {
						return nil
					}
					return err
				}
				if len(current.Finalizers) == 0 {
					return nil
				}
				current.Finalizers = nil
				if err := k8sClient.Update(context.Background(), current); err != nil {
					if errors.IsNotFound(err) {
						return nil
					}
					return err
				}
				return nil
			}, "10s", "250ms").Should(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), nn, &v2alpha1.Astarte{})
			}, "10s", "250ms").ShouldNot(Succeed())
		}
		Eventually(func() int {
			list := &v2alpha1.AstarteList{}
			if err := k8sClient.List(context.Background(), list, &client.ListOptions{Namespace: CustomAstarteNamespace}); err != nil {
				return -1
			}
			return len(list.Items)
		}, "10s", "250ms").Should(Equal(0))
	})

	Describe("TestFunction", func() {

	})
})
