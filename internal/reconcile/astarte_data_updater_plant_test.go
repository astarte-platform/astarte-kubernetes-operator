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

//nolint:goconst,dupl
package reconcile

import (
	"context"
	"strconv"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"

	"go.openly.dev/pointy"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("Misc utils testing", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "example-astarte"
		CustomAstarteNamespace = "astarte-data-updater-plant-test"
	)

	var cr *apiv2alpha1.Astarte

	BeforeAll(func() {
		integrationutils.CreateNamespace(k8sClient, CustomAstarteNamespace)
	})

	AfterAll(func() {
		integrationutils.TeardownNamespace(k8sClient, CustomAstarteNamespace)
	})

	BeforeEach(func() {
		cr = baseCr.DeepCopy()
		cr.SetName(CustomAstarteName)
		cr.SetNamespace(CustomAstarteNamespace)
		cr.SetResourceVersion("")
		integrationutils.DeployAstarte(k8sClient, cr, CustomAstarteNamespace)
	})

	AfterEach(func() {
		integrationutils.TeardownResources(context.Background(), k8sClient, CustomAstarteNamespace)
	})
	Describe("Test EnsureAstarteDataUpdaterPlant", func() {
		It("should return error if it is not possible to list current DUP Deployments", func() {
			// To make this fail, we will use a non existing namespace
			brokenCR := cr.DeepCopy()
			brokenCR.Namespace = "non-existing-namespace"

			dup := apiv2alpha1.AstarteDataUpdaterPlantSpec{
				AstarteGenericClusteredResource: apiv2alpha1.AstarteGenericClusteredResource{
					Deploy:   pointy.Bool(true),
					Replicas: pointy.Int32(2),
				},
			}

			Expect(EnsureAstarteDataUpdaterPlant(brokenCR, dup, k8sClient, scheme.Scheme)).ToNot(Succeed())
		})

		It("should return nil if the component is not enabled", func() {
			dup := apiv2alpha1.AstarteDataUpdaterPlantSpec{
				AstarteGenericClusteredResource: apiv2alpha1.AstarteGenericClusteredResource{
					Deploy: pointy.Bool(false),
				},
			}

			Expect(EnsureAstarteDataUpdaterPlant(cr, dup, k8sClient, scheme.Scheme)).To(Succeed())
		})

		It("should create the right number of DUP deployments", func() {
			// To test this we create 2 DUP deployments, then update the Astarte CR to have only 1 DUP replica and check
			// that only 1 is left
			cr1 := cr.DeepCopy()
			cr1.ResourceVersion = ""
			cr1.Name = "two-replicas-dup"
			cr1.Spec.Components.DataUpdaterPlant.Deploy = pointy.Bool(true)
			cr1.Spec.Components.DataUpdaterPlant.Replicas = pointy.Int32(2)
			dups := &appsv1.DeploymentList{}

			// We should have 2 deployments now
			Expect(k8sClient.Create(context.Background(), cr1)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr1.Name, Namespace: cr1.Namespace}, cr1)
			}, Timeout, Interval).Should(Succeed())

			Expect(EnsureAstarteDataUpdaterPlant(cr1, cr1.Spec.Components.DataUpdaterPlant, k8sClient, scheme.Scheme)).To(Succeed())
			Expect(k8sClient.List(context.Background(), dups, client.InNamespace(cr1.Namespace),
				client.MatchingLabels{"astarte-component": "data-updater-plant"})).To(Succeed())
			Expect(dups.Items).To(HaveLen(2))

			// Update the CR to have only 1 DUP replica
			cr1.Spec.Components.DataUpdaterPlant.Replicas = pointy.Int32(1)
			Expect(k8sClient.Update(context.Background(), cr1)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr1.Name, Namespace: cr1.Namespace}, cr1)
			}, Timeout, Interval).Should(Succeed())

			// We should have only 1 deployment now
			Expect(EnsureAstarteDataUpdaterPlant(cr1, cr1.Spec.Components.DataUpdaterPlant, k8sClient, scheme.Scheme)).To(Succeed())

			Expect(k8sClient.List(context.Background(), dups, client.InNamespace(cr1.Namespace),
				client.MatchingLabels{"astarte-component": "data-updater-plant"})).To(Succeed())
			Expect(dups.Items).To(HaveLen(1))

			// Cleanup
			Expect(k8sClient.Delete(context.Background(), cr1)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr1.Name, Namespace: cr1.Namespace}, &apiv2alpha1.Astarte{})
			}, Timeout, Interval).ShouldNot(Succeed())
		})
	})

	Describe("Test createIndexedDataUpdaterPlantDeployment", func() {
		It("should create the requested number of deployments with the right labels", func() {
			cr.Spec.Components.DataUpdaterPlant.Deploy = pointy.Bool(true)
			cr.Spec.Components.DataUpdaterPlant.Replicas = pointy.Int32(3)

			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, Timeout, Interval).Should(Succeed())

			Expect(createIndexedDataUpdaterPlantDeployment(0, 3, cr, cr.Spec.Components.DataUpdaterPlant, k8sClient, scheme.Scheme)).To(Succeed())
			Expect(createIndexedDataUpdaterPlantDeployment(1, 3, cr, cr.Spec.Components.DataUpdaterPlant, k8sClient, scheme.Scheme)).To(Succeed())
			Expect(createIndexedDataUpdaterPlantDeployment(2, 3, cr, cr.Spec.Components.DataUpdaterPlant, k8sClient, scheme.Scheme)).To(Succeed())

			dups := &appsv1.DeploymentList{}
			Expect(k8sClient.List(context.Background(), dups, client.InNamespace(cr.Namespace),
				client.MatchingLabels{"astarte-component": "data-updater-plant"})).To(Succeed())

			Expect(dups.Items).To((HaveLen(3)))
			// Check that the deployments have the right names and labels
			Expect(dups.Items[0].Name).To(Equal(CustomAstarteName + "-data-updater-plant"))
			Expect(dups.Items[1].Name).To(Equal(CustomAstarteName + "-data-updater-plant-1"))
			Expect(dups.Items[2].Name).To(Equal(CustomAstarteName + "-data-updater-plant-2"))

			for i, d := range dups.Items {
				Expect(d.Labels).ToNot(BeNil())
				if i == 0 {
					Expect(d.Labels["app"]).To(Equal(CustomAstarteName + "-data-updater-plant"))
				} else {
					Expect(d.Labels["app"]).To(Equal(CustomAstarteName + "-data-updater-plant-" + strconv.Itoa(i)))
				}
				Expect(d.Labels["component"]).To(Equal("astarte"))
				Expect(d.Labels["astarte-component"]).To(Equal("data-updater-plant"))
				Expect(d.Labels["astarte-instance-name"]).To(Equal(CustomAstarteName))
				// Check that each deployment has exactly one replica
				Expect(*d.Spec.Replicas).To(Equal(int32(1)))
			}
		})
	})
})
