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
	// "context"

	// apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	// . "github.com/onsi/gomega"
	// "go.openly.dev/pointy"
	// "k8s.io/apimachinery/pkg/api/errors"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/types"
	// "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Astarte Controller", func() {
	// Context("When reconciling a resource", func() {
	// 	const resourceName = "test-resource"

	// 	ctx := context.Background()

	// 	typeNamespacedName := types.NamespacedName{
	// 		Name:      resourceName,
	// 		Namespace: "default",
	// 	}
	// 	astarte := &apiv2alpha1.Astarte{}

	// 	BeforeEach(func() {
	// 		By("creating the custom resource for the Kind Astarte")
	// 		err := k8sClient.Get(ctx, typeNamespacedName, astarte)
	// 		if err != nil && errors.IsNotFound(err) {
	// 			resource := &apiv2alpha1.Astarte{
	// 				ObjectMeta: metav1.ObjectMeta{
	// 					Name:      resourceName,
	// 					Namespace: "default",
	// 				},
	// 				Spec: apiv2alpha1.AstarteSpec{
	// 					Version: "1.3.0",
	// 					API: apiv2alpha1.AstarteAPISpec{
	// 						Host: "api.example.com",
	// 					},
	// 					RabbitMQ: apiv2alpha1.AstarteRabbitMQSpec{
	// 						Connection: &apiv2alpha1.AstarteRabbitMQConnectionSpec{
	// 							HostAndPort: apiv2alpha1.HostAndPort{
	// 								Host: "rabbitmq.example.com",
	// 								Port: pointy.Int32(5672),
	// 							},
	// 							GenericConnectionSpec: apiv2alpha1.GenericConnectionSpec{
	// 								CredentialsSecret: &apiv2alpha1.LoginCredentialsSecret{
	// 									Name:        "rabbitmq-credentials",
	// 									UsernameKey: "username",
	// 									PasswordKey: "password",
	// 								},
	// 							},
	// 						},
	// 					},
	// 					VerneMQ: apiv2alpha1.AstarteVerneMQSpec{
	// 						HostAndPort: apiv2alpha1.HostAndPort{
	// 							Host: "vernemq.example.com",
	// 							Port: pointy.Int32(1883),
	// 						},
	// 						AstarteGenericClusteredResource: apiv2alpha1.AstarteGenericClusteredResource{
	// 							Image: "docker.io/astarte/vernemq:1.3-snapshot",
	// 						},
	// 					},
	// 				},
	// 			}
	// 			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
	// 		}
	// 	})

	// 	AfterEach(func() {
	// 		resource := &apiv2alpha1.Astarte{}
	// 		err := k8sClient.Get(ctx, typeNamespacedName, resource)
	// 		Expect(err).NotTo(HaveOccurred())

	// 		By("Cleanup the specific resource instance Astarte")
	// 		Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
	// 	})
	// 	It("should successfully reconcile the resource", func() {
	// 		By("Reconciling the created resource")
	// 		controllerReconciler := &AstarteReconciler{
	// 			Client: k8sClient,
	// 			Scheme: k8sClient.Scheme(),
	// 		}

	// 		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
	// 			NamespacedName: typeNamespacedName,
	// 		})
	// 		Expect(err).NotTo(HaveOccurred())
	// 	})
	// })
})
