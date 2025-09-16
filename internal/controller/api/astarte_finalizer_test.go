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
package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
)

var _ = Describe("Astarte Finalizer testing", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "test-astarte-finalizer"
		CustomAstarteNamespace = "astarte-finalizer-tests"
		AstarteVersion         = "1.3.0"
		CustomRabbitMQHost     = "custom-rabbitmq-host"
		CustomRabbitMQPort     = 5673
		CustomVerneMQHost      = "vernemq.example.com"
		CustomVerneMQPort      = 8884
	)

	var cr *v2alpha1.Astarte
	var reconciler *AstarteReconciler

	BeforeAll(func() {
		if CustomAstarteNamespace != "default" {
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: CustomAstarteNamespace,
				},
			}

			Eventually(func() error {
				err := k8sClient.Create(context.Background(), ns)
				if apierrors.IsAlreadyExists(err) {
					return nil
				}
				return err
			}, Timeout, Interval).Should(Succeed())
		}
	})

	AfterAll(func() {
		if CustomAstarteNamespace != "default" {
			astartes := &v2alpha1.AstarteList{}
			Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())

			for _, a := range astartes.Items {
				Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &v2alpha1.Astarte{})
				}, Timeout, Interval).ShouldNot(Succeed())
			}

			// Attempt namespace deletion but don't block on it in envtest
			_ = k8sClient.Delete(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}})
		}
	})

	BeforeEach(func() {
		// Initialize reconciler
		reconciler = &AstarteReconciler{
			Client: k8sClient,
		}

		// Create and initialize a basic Astarte CR
		cr = &v2alpha1.Astarte{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CustomAstarteName,
				Namespace: CustomAstarteNamespace,
			},
			Spec: v2alpha1.AstarteSpec{
				Version: AstarteVersion,
				API: v2alpha1.AstarteAPISpec{
					Host: "test.example.com",
				},
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
	})

	AfterEach(func() {
		astartes := &v2alpha1.AstarteList{}
		Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())

		for i := range astartes.Items {
			a := &astartes.Items[i]

			// Remove any finalizers to avoid blocking deletion
			a.SetFinalizers([]string{})
			Expect(k8sClient.Update(context.Background(), a)).To(Succeed())

			// Delete each Astarte resource
			Eventually(func() error {
				return k8sClient.Delete(context.Background(), &astartes.Items[i])
			}, Timeout, Interval).Should(Succeed())

			// Wait for deletion to complete
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &v2alpha1.Astarte{})
			}, Timeout, Interval).ShouldNot(Succeed())
		}

	})

	Describe("Test HandleFinalization", func() {
		It("should successfully finalize Astarte instance with finalizer", func() {
			// Add finalizer to the CR
			cr.SetFinalizers([]string{astarteFinalizer})

			Eventually(func() error {
				return k8sClient.Create(context.Background(), cr)
			}, Timeout, Interval).Should(Succeed())

			// Verify CR was created with finalizer
			Eventually(func() []string {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
				if err != nil {
					return nil
				}
				return cr.GetFinalizers()
			}, Timeout, Interval).Should(ContainElement(astarteFinalizer))

			// Mark the CR for deletion
			Eventually(func() error {
				return k8sClient.Delete(context.Background(), cr)
			}, Timeout, Interval).Should(Succeed())

			// Get a copy with deletion timestamp
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
				if err != nil {
					return false
				}
				return cr.GetDeletionTimestamp() != nil
			}, Timeout, Interval).Should(BeTrue())

			// Call handleFinalization
			result, err := reconciler.handleFinalization(cr)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify finalizer was removed from the object
			Expect(cr.GetFinalizers()).ToNot(ContainElement(astarteFinalizer))

			// Verify CR is eventually deleted (handleFinalization should have updated it)
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, &v2alpha1.Astarte{})
			}, Timeout, Interval).ShouldNot(Succeed())
		})
	})

	Describe("Test AddFinalizer", func() {
		It("should successfully add finalizer to CR", func() {
			// Createa new CR without finalizers
			crNew := cr.DeepCopy()
			crNew.Name = "test-astarte-finalizer-add"
			crNew.SetFinalizers([]string{})
			crNew.ResourceVersion = ""

			Expect(k8sClient.Create(context.Background(), crNew)).To(Succeed())

			// Verify CR was created without finalizer
			Eventually(func() []string {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: crNew.Name, Namespace: crNew.Namespace}, crNew)
				if err != nil {
					return nil
				}
				return crNew.GetFinalizers()
			}, Timeout, Interval).ShouldNot(ContainElement(astarteFinalizer))

			// Add finalizer
			err := reconciler.addFinalizer(crNew)
			Expect(err).ToNot(HaveOccurred())

			// Verify finalizer was added
			Eventually(func() []string {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: crNew.Name, Namespace: crNew.Namespace}, crNew)
				if err != nil {
					return nil
				}
				return crNew.GetFinalizers()
			}, Timeout, Interval).Should(ContainElement(astarteFinalizer))
		})
	})
})
