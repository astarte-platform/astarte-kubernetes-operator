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

package reconcile

import (
	"context"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("Astarte PriorityClass reconcile tests", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "example-astarte-priorityclasses"
		CustomAstarteNamespace = "astarte-priorityclass-test"
	)

	var cr *apiv2alpha1.Astarte

	BeforeAll(func() {
		integrationutils.CreateNamespace(k8sClient, CustomAstarteNamespace)
	})

	AfterAll(func() {
		integrationutils.DeleteNamespace(k8sClient, CustomAstarteNamespace)
	})

	BeforeEach(func() {
		cr = baseCr.DeepCopy()
		cr.SetName(CustomAstarteName)
		cr.SetNamespace(CustomAstarteNamespace)
		cr.SetResourceVersion("")
		integrationutils.DeployAstarte(k8sClient, cr)
	})

	AfterEach(func() {
		integrationutils.TeardownResourcesInNamespace(context.Background(), k8sClient, CustomAstarteNamespace)
	})

	Describe("EnsureAstartePriorityClasses", func() {
		It("should not create PriorityClasses when feature disabled or nil", func() {
			// Feature nil
			cr.Spec.Features.AstartePodPriorities = nil
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, Timeout, Interval).Should(Succeed())
			Expect(EnsureAstartePriorityClasses(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Objects should not exist
			for _, name := range []string{AstarteHighPriorityName, AstarteMidPriorityName, AstarteLowPriorityName} {
				pc := &schedulingv1.PriorityClass{}
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: name}, pc)
				}, Timeout, Interval).ShouldNot(Succeed())
			}

			// Explicitly disable
			cr.Spec.Features.AstartePodPriorities = &apiv2alpha1.AstartePodPrioritiesSpec{Enable: false}
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, Timeout, Interval).Should(Succeed())
			Expect(EnsureAstartePriorityClasses(cr, k8sClient, scheme.Scheme)).To(Succeed())
			for _, name := range []string{AstarteHighPriorityName, AstarteMidPriorityName, AstarteLowPriorityName} {
				pc := &schedulingv1.PriorityClass{}
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: name}, pc)
				}, Timeout, Interval).ShouldNot(Succeed())
			}
		})

		It("should create all PriorityClasses with expected values when enabled", func() {
			cr.Spec.Features.AstartePodPriorities = &apiv2alpha1.AstartePodPrioritiesSpec{
				Enable:              true,
				AstarteHighPriority: pointy.Int(2000),
				AstarteMidPriority:  pointy.Int(200),
				AstarteLowPriority:  pointy.Int(20),
			}
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, Timeout, Interval).Should(Succeed())

			Expect(EnsureAstartePriorityClasses(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Check all PriorityClasses exist with expected values
			testPriorityClass := func(name string, expected int32) {
				pc := &schedulingv1.PriorityClass{}
				Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: name}, pc)).To(Succeed())
				Expect(pc.Value).To(Equal(expected))
				Expect(pc.GlobalDefault).To(BeFalse())
				Expect(pc.PreemptionPolicy).ToNot(BeNil())
				Expect(*pc.PreemptionPolicy).To(Equal(v1.PreemptNever))
				Expect(pc.Description).To(ContainSubstring("Astarte"))
			}

			testPriorityClass(AstarteHighPriorityName, int32(2000))
			testPriorityClass(AstarteMidPriorityName, int32(200))
			testPriorityClass(AstarteLowPriorityName, int32(20))
		})

		It("should reconcile and restore PriorityClass values if changed", func() {
			cr.Spec.Features.AstartePodPriorities = &apiv2alpha1.AstartePodPrioritiesSpec{
				Enable:              true,
				AstarteHighPriority: pointy.Int(1500),
				AstarteMidPriority:  pointy.Int(150),
				AstarteLowPriority:  pointy.Int(15),
			}
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, Timeout, Interval).Should(Succeed())

			Expect(EnsureAstartePriorityClasses(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Helper function to test priority class properties
			testPriorityClassProperties := func(name string, expectedValue int32, expectedDescriptionSubstring string) {
				pc := &schedulingv1.PriorityClass{}
				Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: name}, pc)).To(Succeed())
				Expect(pc.Value).To(Equal(expectedValue))
				Expect(pc.Description).To(ContainSubstring(expectedDescriptionSubstring))
				Expect(pc.GlobalDefault).To(BeFalse())
				Expect(pc.PreemptionPolicy).ToNot(BeNil())
				Expect(*pc.PreemptionPolicy).To(Equal(v1.PreemptNever))
			}

			// Ensure all priority classes are created with expected values
			testPriorityClassProperties(AstarteHighPriorityName, int32(1500), "Astarte high-priority pods")
			testPriorityClassProperties(AstarteMidPriorityName, int32(150), "Astarte mid-priority pods")
			testPriorityClassProperties(AstarteLowPriorityName, int32(15), "Astarte low-priority pods")

			// Test reconciliation by modifying mutable fields and ensuring they are restored
			for _, testCase := range []struct {
				name                         string
				expectedValue                int32
				expectedDescriptionSubstring string
			}{
				{AstarteHighPriorityName, int32(1500), "Astarte high-priority pods"},
				{AstarteMidPriorityName, int32(150), "Astarte mid-priority pods"},
				{AstarteLowPriorityName, int32(15), "Astarte low-priority pods"},
			} {
				pc := &schedulingv1.PriorityClass{}
				Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: testCase.name}, pc)).To(Succeed())

				originalValue := pc.Value
				originalDescription := pc.Description

				// Modify mutable fields (e.g. Description
				pc.Description = "modified description"
				Expect(k8sClient.Update(context.Background(), pc)).To(Succeed())

				// Verify the changes were applied
				modifiedPc := &schedulingv1.PriorityClass{}
				Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: testCase.name}, modifiedPc)).To(Succeed())
				Expect(modifiedPc.Description).To(Equal("modified description"))

				// Re-run reconciliation - should restore all fields to expected values
				Expect(EnsureAstartePriorityClasses(cr, k8sClient, scheme.Scheme)).To(Succeed())

				// Verify reconciliation restored the correct values
				restoredPc := &schedulingv1.PriorityClass{}
				Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: testCase.name}, restoredPc)).To(Succeed())
				Expect(restoredPc.Value).To(Equal(originalValue))
				Expect(restoredPc.Description).To(Equal(originalDescription))
				Expect(restoredPc.GlobalDefault).To(BeFalse())
				Expect(*restoredPc.PreemptionPolicy).To(Equal(v1.PreemptNever))

				// Try to change non-mutable field (Value) - should trigger an error on update
				Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: AstarteHighPriorityName}, pc)).To(Succeed())
				pc.Value = originalValue + 100
				Expect(k8sClient.Update(context.Background(), pc)).ToNot(Succeed())
				// Re-run ensure - should not change anything as Value cannot be changed
				Expect(EnsureAstartePriorityClasses(cr, k8sClient, scheme.Scheme)).To(Succeed())
				Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: testCase.name}, pc)).To(Succeed())
				Expect(pc.Value).To(Equal(originalValue)) // Value should remain unchanged
			}
		})
	})

	Describe("GetDefaultAstartePriorityClassNameForComponent", func() {
		It("should return expected mapping for known components", func() {
			cases := map[apiv2alpha1.AstarteComponent]string{
				apiv2alpha1.AppEngineAPI:     AstarteMidPriorityName,
				apiv2alpha1.DataUpdaterPlant: AstarteHighPriorityName,
				apiv2alpha1.FlowComponent:    AstarteMidPriorityName,
				apiv2alpha1.Housekeeping:     AstarteLowPriorityName,
				apiv2alpha1.Pairing:          AstarteMidPriorityName,
				apiv2alpha1.RealmManagement:  AstarteLowPriorityName,
				apiv2alpha1.TriggerEngine:    AstarteLowPriorityName,
				apiv2alpha1.Dashboard:        AstarteLowPriorityName,
			}
			for comp, expected := range cases {
				Expect(GetDefaultAstartePriorityClassNameForComponent(comp)).To(Equal(expected))
			}
		})

		It("should return empty string for unknown component", func() {
			var fake apiv2alpha1.AstarteComponent = "non_existing_component"
			Expect(GetDefaultAstartePriorityClassNameForComponent(fake)).To(Equal(""))
		})
	})
})
