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

package controller

import (
	"context"
	"testing"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Astarte Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		var controllerReconciler *AstarteReconciler
		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		astarte := &apiv2alpha1.Astarte{}

		BeforeEach(func() {
			By("Initializing the controller reconciler")
			controllerReconciler = &AstarteReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Log:      ctrl.Log.WithName("test-reconciler"),
				Recorder: record.NewFakeRecorder(1024),
			}

			By("creating the custom resource for the Kind Astarte")
			err := k8sClient.Get(ctx, typeNamespacedName, astarte)
			if err != nil && errors.IsNotFound(err) {
				resource := &apiv2alpha1.Astarte{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: apiv2alpha1.AstarteSpec{
						Version: "1.3.0",
						API: apiv2alpha1.AstarteAPISpec{
							Host: "api.example.com",
						},
						RabbitMQ: apiv2alpha1.AstarteRabbitMQSpec{
							Connection: &apiv2alpha1.AstarteRabbitMQConnectionSpec{
								HostAndPort: apiv2alpha1.HostAndPort{
									Host: "rabbitmq.example.com",
									Port: pointy.Int32(5672),
								},
								GenericConnectionSpec: apiv2alpha1.GenericConnectionSpec{
									CredentialsSecret: &apiv2alpha1.LoginCredentialsSecret{
										Name:        "rabbitmq-credentials",
										UsernameKey: "username",
										PasswordKey: "password",
									},
								},
							},
						},
						VerneMQ: apiv2alpha1.AstarteVerneMQSpec{
							HostAndPort: apiv2alpha1.HostAndPort{
								Host: "vernemq.example.com",
								Port: pointy.Int32(1883),
							},
							AstarteGenericClusteredResource: apiv2alpha1.AstarteGenericClusteredResource{
								Image: "docker.io/astarte/vernemq:1.3-snapshot",
							},
						},
						Cassandra: apiv2alpha1.AstarteCassandraSpec{
							Connection: &apiv2alpha1.AstarteCassandraConnectionSpec{
								Nodes: []apiv2alpha1.HostAndPort{
									{
										Host: "cassandra1.example.com",
										Port: pointy.Int32(9042),
									},
								},
								GenericConnectionSpec: apiv2alpha1.GenericConnectionSpec{
									CredentialsSecret: &apiv2alpha1.LoginCredentialsSecret{
										Name:        "cassandra-credentials",
										UsernameKey: "username",
										PasswordKey: "password",
									},
								},
							},
						},
					},
				}

				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// After each test, let's make sure the resource exists
			By("getting the created resource")
			resource := &apiv2alpha1.Astarte{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			// ... and then delete it
			By("deleting the created resource")
			err = k8sClient.Delete(ctx, resource)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not reconcile an unsupported astarte version", func() {
			By("Updating the resource to an unsupported version")
			resource := &apiv2alpha1.Astarte{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			resource.Spec.Version = "4.0.1"
			Expect(k8sClient.Update(ctx, resource)).To(Succeed())

			By("Reconciling the created resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).To(HaveOccurred())
		})

		It("should reconcile when in manual maintenance mode", func() {
			By("Updating the resource to be in manual maintenance mode")
			resource := &apiv2alpha1.Astarte{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			resource.Spec.ManualMaintenanceMode = true
			Expect(k8sClient.Update(ctx, resource)).To(Succeed())

			By("Reconciling the created resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not requeue when the resource is not found and there are no errors", func() {
			By("Reconciling a non-existing resource")
			res, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existing",
					Namespace: "default",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Requeue).To(BeFalse())
		})

	})
})

func TestContains(t *testing.T) {
	testCases := []struct {
		description string
		list        []string
		s           string
		expected    bool
	}{
		{
			description: "string is in the list",
			list:        []string{"a", "b", "c"},
			s:           "b",
			expected:    true,
		},
		{
			description: "string is not in the list",
			list:        []string{"a", "b", "c"},
			s:           "d",
			expected:    false,
		},
		{
			description: "empty list",
			list:        []string{},
			s:           "a",
			expected:    false,
		},
		{
			description: "empty string is in the list",
			list:        []string{"", "a", "b"},
			s:           "",
			expected:    true,
		},
	}

	g := NewWithT(t)
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := contains(tc.list, tc.s)
			g.Expect(result).To(Equal(tc.expected))
		})
	}
}

func TestRemove(t *testing.T) {
	testCases := []struct {
		description string
		list        []string
		s           string
		expected    []string
	}{
		{
			description: "string is in the list",
			list:        []string{"a", "b", "c"},
			s:           "b",
			expected:    []string{"a", "c"},
		},
		{
			description: "string is not in the list",
			list:        []string{"a", "b", "c"},
			s:           "d",
			expected:    []string{"a", "b", "c"},
		},
		{
			description: "empty list",
			list:        []string{},
			s:           "a",
			expected:    []string{},
		},
		{
			description: "empty string is in the list",
			list:        []string{"", "a", "b"},
			s:           "",
			expected:    []string{"a", "b"},
		},
		{
			description: "empty string is not in the list",
			list:        []string{"a", "b"},
			s:           "",
			expected:    []string{"a", "b"},
		},
	}

	g := NewWithT(t)
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := remove(tc.list, tc.s)
			g.Expect(result).To(Equal(tc.expected))
		})
	}
}
