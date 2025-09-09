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
	"time"

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
			if err == nil {
				// ... and then delete it only if it exists
				By("deleting the created resource")
				err = k8sClient.Delete(ctx, resource)
				Expect(err).ToNot(HaveOccurred())
			} else if !errors.IsNotFound(err) {
				// If error is something other than NotFound, it's unexpected
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not reconcile an unsupported astarte version", func() {
			By("Updating the resource to an unsupported version")
			resource := &apiv2alpha1.Astarte{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).ToNot(HaveOccurred())

			resource.Spec.Version = "4.0.1"
			Expect(k8sClient.Update(ctx, resource)).To(Succeed())

			By("Reconciling the created resource")
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).To(HaveOccurred())
			// Check that we're requeuing after a minute due to version error
			Expect(result.RequeueAfter).To(Equal(time.Minute))
		})

		It("should reconcile when in manual maintenance mode", func() {
			By("Updating the resource to be in manual maintenance mode")
			resource := &apiv2alpha1.Astarte{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).ToNot(HaveOccurred())

			resource.Spec.ManualMaintenanceMode = true
			Expect(k8sClient.Update(ctx, resource)).To(Succeed())

			By("Reconciling the created resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not requeue when the resource is not found and there are no errors", func() {
			By("Reconciling a non-existing resource")
			res, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existing",
					Namespace: "default",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Requeue).To(BeFalse())
		})

	})

	Context("Testing the Astarte finalizer", func() {
		var controllerReconciler *AstarteReconciler
		var astarteInstance *apiv2alpha1.Astarte
		var ctx context.Context

		BeforeEach(func() {
			// Create a new context for this test
			ctx = context.Background()

			// Setup a client for testing the finalizer
			astarteInstance = &apiv2alpha1.Astarte{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-finalizer",
					Namespace:         "default",
					Finalizers:        []string{astarteFinalizer},
					DeletionTimestamp: &metav1.Time{Time: metav1.Now().Time},
				},
				Spec: apiv2alpha1.AstarteSpec{
					Version: "1.3.0",
					VerneMQ: apiv2alpha1.AstarteVerneMQSpec{
						HostAndPort: apiv2alpha1.HostAndPort{
							Host: "vernemq.example.com",
							Port: pointy.Int32(1883),
						},
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
					API: apiv2alpha1.AstarteAPISpec{
						Host: "api.example.com",
					},
				},
			}

			// Create a new reconciler with the test client
			scheme := k8sClient.Scheme()
			controllerReconciler = &AstarteReconciler{
				Client:   k8sClient,
				Scheme:   scheme,
				Log:      ctrl.Log.WithName("test-finalizer-reconciler"),
				Recorder: record.NewFakeRecorder(1024),
			}

			// Create the resource with finalizer
			Expect(k8sClient.Create(ctx, astarteInstance)).To(Succeed())
		})

		AfterEach(func() {
			// Clean up
			resource := &apiv2alpha1.Astarte{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-finalizer",
				Namespace: "default",
			}, resource)

			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			} else if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("should handle finalization", func() {
			By("Reconciling a resource with deletion timestamp")
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-finalizer",
					Namespace: "default",
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})
	})

	Context("Testing the addFinalizer function", func() {
		var controllerReconciler *AstarteReconciler
		var astarteInstance *apiv2alpha1.Astarte
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()

			astarteInstance = &apiv2alpha1.Astarte{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-add-finalizer",
					Namespace: "default",
				},
				Spec: apiv2alpha1.AstarteSpec{
					Version: "1.3.0",
					VerneMQ: apiv2alpha1.AstarteVerneMQSpec{
						HostAndPort: apiv2alpha1.HostAndPort{
							Host: "vernemq.example.com",
							Port: pointy.Int32(1883),
						},
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
					API: apiv2alpha1.AstarteAPISpec{
						Host: "api.example.com",
					},
				},
			}

			scheme := k8sClient.Scheme()
			controllerReconciler = &AstarteReconciler{
				Client:   k8sClient,
				Scheme:   scheme,
				Log:      ctrl.Log.WithName("test-add-finalizer-reconciler"),
				Recorder: record.NewFakeRecorder(1024),
			}

			Expect(k8sClient.Create(ctx, astarteInstance)).To(Succeed())
		})

		AfterEach(func() {
			resource := &apiv2alpha1.Astarte{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-add-finalizer",
				Namespace: "default",
			}, resource)

			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			} else if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("should add a finalizer to an Astarte resource", func() {
			By("Adding a finalizer to the resource")
			err := controllerReconciler.addFinalizer(astarteInstance)
			Expect(err).ToNot(HaveOccurred())

			// Check that the finalizer was added
			updatedResource := &apiv2alpha1.Astarte{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-add-finalizer",
				Namespace: "default",
			}, updatedResource)

			Expect(err).ToNot(HaveOccurred())
			Expect(updatedResource.Finalizers).To(ContainElement(astarteFinalizer))
		})
	})
})

var _ = Describe("Standalone Tests", func() {
	Context("Testing with k8sClient directly", func() {
		It("should handle non-existent resources", func() {
			ctx := context.Background()

			// Create a test resource
			astarte := &apiv2alpha1.Astarte{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-direct-reconcile",
					Namespace: "default",
				},
				Spec: apiv2alpha1.AstarteSpec{
					Version: "1.3.0",
					VerneMQ: apiv2alpha1.AstarteVerneMQSpec{
						HostAndPort: apiv2alpha1.HostAndPort{
							Host: "vernemq.example.com",
							Port: pointy.Int32(1883),
						},
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
					API: apiv2alpha1.AstarteAPISpec{
						Host: "api.example.com",
					},
				},
			}

			// Create the resource in the test environment
			Expect(k8sClient.Create(ctx, astarte)).To(Succeed())

			// Create the reconciler with the test client
			reconciler := &AstarteReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Log:      ctrl.Log.WithName("test-direct-reconciler"),
				Recorder: record.NewFakeRecorder(1024),
			}

			// Test reconciling a non-existent resource
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent",
					Namespace: "default",
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			// Clean up
			Expect(k8sClient.Delete(ctx, astarte)).To(Succeed())
		})
	})

	Context("Testing utility functions", func() {
		Context("contains function", func() {
			It("should correctly identify if a string is in a list", func() {
				Expect(contains([]string{"a", "b", "c"}, "b")).To(BeTrue())
				Expect(contains([]string{"a", "b", "c"}, "d")).To(BeFalse())
				Expect(contains([]string{}, "a")).To(BeFalse())
				Expect(contains([]string{"", "a", "b"}, "")).To(BeTrue())
				Expect(contains(nil, "a")).To(BeFalse())
				Expect(contains([]string{"A", "B", "C"}, "a")).To(BeFalse(), "should be case sensitive")
			})
		})

		Context("remove function", func() {
			It("should correctly remove a string from a list", func() {
				Expect(remove([]string{"a", "b", "c"}, "b")).To(Equal([]string{"a", "c"}))
				Expect(remove([]string{"a", "b", "c"}, "d")).To(Equal([]string{"a", "b", "c"}))
				Expect(remove([]string{}, "a")).To(Equal([]string{}))
				Expect(remove([]string{"", "a", "b"}, "")).To(Equal([]string{"a", "b"}))
				Expect(remove([]string{"a", "b"}, "")).To(Equal([]string{"a", "b"}))
			})

			It("should have an issue with multiple occurrences", func() {
				// This test is explicitly marked as "known issue"
				originalList := []string{"a", "b", "b", "c"}
				result := remove(originalList, "b")

				// Current implementation removes only the first occurrence
				// This is not what we want, but it's the current behavior
				Expect(result).NotTo(Equal([]string{"a", "c"}))

				// For documentation purposes, show the actual current behavior
				Expect(len(result)).To(BeNumerically(">", 2), "Current implementation doesn't handle multiple occurrences correctly")
			})
		})
	})
})
