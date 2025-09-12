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

// Why is there a no-lint here? Well, golangci-lint is failing on this whole file
// and it is not giving any useful information on why. Probably a bug in golangci-lint
// that is outdated. Disabling linting for this file for now, until we can upgrade golangci-lint.
// nolint
package reconcile

import (
	"context"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Astarte PriorityClass reconcile tests", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "my-astarte-priorityclasses"
		CustomAstarteNamespace = "astarte-priorityclass-test"
		AstarteVersion         = "1.3.0"
		CustomRabbitMQHost     = "rabbitmq.example.com"
		CustomRabbitMQPort     = 5672
		CustomVerneMQHost      = "vernemq.example.com"
		CustomVerneMQPort      = 8883
	)

	var cr *apiv2alpha1.Astarte

	BeforeAll(func() {
		// Since priorityclasses are cluster-wide, we need to ensure no other tests left them behind
		// Cleanup of priorityclasses that might remain from previous test runs
		for _, name := range []string{AstarteHighPriorityName, AstarteMidPriorityName, AstarteLowPriorityName} {
			pc := &schedulingv1.PriorityClass{}
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: name}, pc)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), pc)
			}

			// Ensure they are gone
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: name}, pc)
				return apierrors.IsNotFound(err)
			}, "10s", "250ms").Should(BeTrue())
		}

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
			astartes := &apiv2alpha1.AstarteList{}
			Expect(k8sClient.List(context.Background(), astartes, client.InNamespace(CustomAstarteNamespace))).To(Succeed())
			for _, a := range astartes.Items {
				_ = k8sClient.Delete(context.Background(), &a)
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &apiv2alpha1.Astarte{})
				}, "10s", "250ms").ShouldNot(Succeed())
			}
			// Do not delete the namespace here to avoid 'NamespaceTerminating' flakiness in subsequent specs
			//_ = k8sClient.Delete(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}})
		}
	})

	BeforeEach(func() {
		cr = &apiv2alpha1.Astarte{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CustomAstarteName,
				Namespace: CustomAstarteNamespace,
			},
			Spec: apiv2alpha1.AstarteSpec{
				Version:    AstarteVersion,
				RabbitMQ:   apiv2alpha1.AstarteRabbitMQSpec{Connection: &apiv2alpha1.AstarteRabbitMQConnectionSpec{HostAndPort: apiv2alpha1.HostAndPort{Host: CustomRabbitMQHost, Port: pointy.Int32(CustomRabbitMQPort)}}},
				VerneMQ:    apiv2alpha1.AstarteVerneMQSpec{HostAndPort: apiv2alpha1.HostAndPort{Host: CustomVerneMQHost, Port: pointy.Int32(CustomVerneMQPort)}},
				Cassandra:  apiv2alpha1.AstarteCassandraSpec{Connection: &apiv2alpha1.AstarteCassandraConnectionSpec{Nodes: []apiv2alpha1.HostAndPort{{Host: "cassandra.example.com", Port: pointy.Int32(9042)}}}},
				Components: apiv2alpha1.AstarteComponentsSpec{},
			},
		}

		Expect(k8sClient.Create(context.Background(), cr)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
		}, "10s", "250ms").Should(Succeed())
	})

	AfterEach(func() {
		astartes := &apiv2alpha1.AstarteList{}
		Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, a := range astartes.Items {
			Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &apiv2alpha1.Astarte{})
			}, "10s", "250ms").ShouldNot(Succeed())
		}

		// Cleanup of priorityclasses that might remain from tests
		for _, name := range []string{AstarteHighPriorityName, AstarteMidPriorityName, AstarteLowPriorityName} {
			pc := &schedulingv1.PriorityClass{}
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: name}, pc)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), pc)
			}

			// Ensure they are gone
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: name}, pc)
				return apierrors.IsNotFound(err)
			}, "10s", "250ms").Should(BeTrue())
		}

		Eventually(func() bool {
			// Ensure no CRs left
			list := &apiv2alpha1.AstarteList{}
			if err := k8sClient.List(context.Background(), list, &client.ListOptions{Namespace: CustomAstarteNamespace}); err != nil {
				return false
			}
			return len(list.Items) == 0
		}, "10s", "250ms").Should(BeTrue())
	})

	Describe("EnsureAstartePriorityClasses", func() {
		It("should not create PriorityClasses when feature disabled or nil", func() {
			// Feature nil
			cr.Spec.Features.AstartePodPriorities = nil
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, "10s", "250ms").Should(Succeed())
			Expect(EnsureAstartePriorityClasses(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Objects should not exist
			for _, name := range []string{AstarteHighPriorityName, AstarteMidPriorityName, AstarteLowPriorityName} {
				pc := &schedulingv1.PriorityClass{}
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: name}, pc)
				}, "10s", "250ms").ShouldNot(Succeed())
			}

			// Explicitly disable
			cr.Spec.Features.AstartePodPriorities = &apiv2alpha1.AstartePodPrioritiesSpec{Enable: false}
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, "10s", "250ms").Should(Succeed())
			Expect(EnsureAstartePriorityClasses(cr, k8sClient, scheme.Scheme)).To(Succeed())
			for _, name := range []string{AstarteHighPriorityName, AstarteMidPriorityName, AstarteLowPriorityName} {
				pc := &schedulingv1.PriorityClass{}
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: name}, pc)
				}, "10s", "250ms").ShouldNot(Succeed())
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
			}, "10s", "250ms").Should(Succeed())

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
			}, "10s", "250ms").Should(Succeed())

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
