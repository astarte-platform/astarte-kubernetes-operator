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
package ingress

import (
	"context"
	"log"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
	ingressv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v1alpha1"
	v1alpha2mocks "github.com/astarte-platform/astarte-kubernetes-operator/test/mocks/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ADIName     = "adi-test"
	AstarteName = "astarte-test"
	Namespace   = "default"
)

var _ = Describe("AstarteDefaultIngress Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		astarteNamespacedName := types.NamespacedName{
			Name:      AstarteName,
			Namespace: Namespace,
		}

		adiNamespacedName := types.NamespacedName{
			Name:      ADIName,
			Namespace: Namespace,
		}

		adi := &ingressv1alpha1.AstarteDefaultIngress{}
		astarte := &apiv1alpha2.Astarte{}

		BeforeEach(func() {
			By("creating the instance for the Kind Astarte")
			err := k8sClient.Get(ctx, astarteNamespacedName, astarte)

			if err != nil && errors.IsNotFound(err) {
				resource := v1alpha2mocks.GetAstarteMock(AstarteName, Namespace)
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			By("creating the instance for the Kind AstarteDefaultIngress")
			err = k8sClient.Get(ctx, adiNamespacedName, adi)
			if err != nil && errors.IsNotFound(err) {
				resource := &ingressv1alpha1.AstarteDefaultIngress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ADIName,
						Namespace: Namespace,
					},
					Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
						Astarte:      AstarteName,
						IngressClass: "astarte-nginx",
						API: ingressv1alpha1.AstarteDefaultIngressAPISpec{
							Deploy: pointy.Bool(true),
						},
						Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
							Deploy: pointy.Bool(true),
						},
						Broker: ingressv1alpha1.AstarteDefaultIngressBrokerSpec{
							Deploy:      pointy.Bool(true),
							ServiceType: v1.ServiceTypeNodePort,
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			By("Cleanup the specific resource instance AstarteDefaultIngress")
			err := k8sClient.Get(ctx, adiNamespacedName, adi)
			if err == nil {
				Expect(k8sClient.Delete(ctx, adi)).To(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, adiNamespacedName, adi)
					return errors.IsNotFound(err)
				}).Should(BeTrue())
			}

			By("Cleanup the specific resource instance Astarte")
			err = k8sClient.Get(ctx, astarteNamespacedName, astarte)
			if err == nil {
				Expect(k8sClient.Delete(ctx, astarte)).To(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, astarteNamespacedName, astarte)
					return errors.IsNotFound(err)
				}).Should(BeTrue())
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")

			controllerReconciler := &AstarteDefaultIngressReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: adiNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the AstarteDefaultIngress resource exists")
			createdADI := &ingressv1alpha1.AstarteDefaultIngress{}
			err = k8sClient.Get(ctx, adiNamespacedName, createdADI)
			if err != nil {
				log.Println("Error:", err)
			}
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource has the expected spec values")
			Expect(createdADI.Spec.Astarte).To(Equal(AstarteName))
			Expect(createdADI.Spec.IngressClass).To(Equal("astarte-nginx"))
			Expect(pointy.BoolValue(createdADI.Spec.API.Deploy, false)).To(BeTrue())
			Expect(pointy.BoolValue(createdADI.Spec.Dashboard.Deploy, false)).To(BeTrue())
			Expect(pointy.BoolValue(createdADI.Spec.Broker.Deploy, false)).To(BeTrue())
			Expect(createdADI.Spec.Broker.ServiceType).To(Equal(v1.ServiceTypeNodePort))
		})
	})
})
