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

package controllerutils

import (
	"context"

	"github.com/astarte-platform/astarte-kubernetes-operator/internal/reconcile"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	scheduling "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// FinalizeAstarte: not tested here because envtest doesn't run kube-controller-manager,
// so resources with finalizers (e.g., PVCs) aren't actually removed. Delete() returns nil
// but objects remain terminating, making assertions unreliable in this environment.
var _ = Describe("FinalizeAstarte", Serial, func() {
	// Intentionally left without executable tests; see note above.
})

var _ = Describe("finalizePriorityClasses", Ordered, Serial, func() {
	var logger logr.Logger

	BeforeAll(func() {
		logger = logr.Discard()
	})

	AfterEach(func() {
		// Ensure we don't leak the unrelated class between tests
		_ = k8sClient.Delete(context.Background(), &scheduling.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: "unrelated-priority"}})
		// Also try to remove Astarte* classes in case a test failed midway
		_ = k8sClient.Delete(context.Background(), &scheduling.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: reconcile.AstarteHighPriorityName}})
		_ = k8sClient.Delete(context.Background(), &scheduling.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: reconcile.AstarteMidPriorityName}})
		_ = k8sClient.Delete(context.Background(), &scheduling.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: reconcile.AstarteLowPriorityName}})
	})

	It("deletes Astarte PriorityClasses and leaves others", func() {
		preempt := v1.PreemptNever
		// Create three Astarte priority classes and one foreign class with required fields
		high := &scheduling.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: reconcile.AstarteHighPriorityName}, Value: 1000000, GlobalDefault: false, PreemptionPolicy: &preempt}
		mid := &scheduling.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: reconcile.AstarteMidPriorityName}, Value: 500000, GlobalDefault: false, PreemptionPolicy: &preempt}
		low := &scheduling.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: reconcile.AstarteLowPriorityName}, Value: 100000, GlobalDefault: false, PreemptionPolicy: &preempt}
		other := &scheduling.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: "unrelated-priority"}, Value: 1, GlobalDefault: false, PreemptionPolicy: &preempt}

		Expect(k8sClient.Create(context.Background(), high)).To(Succeed())
		Expect(k8sClient.Create(context.Background(), mid)).To(Succeed())
		Expect(k8sClient.Create(context.Background(), low)).To(Succeed())
		Expect(k8sClient.Create(context.Background(), other)).To(Succeed())

		// Sanity check they exist
		pcList := &scheduling.PriorityClassList{}
		Expect(k8sClient.List(context.Background(), pcList)).To(Succeed())
		found := map[string]bool{}
		for _, pc := range pcList.Items {
			found[pc.Name] = true
		}
		Expect(found[reconcile.AstarteHighPriorityName]).To(BeTrue())
		Expect(found[reconcile.AstarteMidPriorityName]).To(BeTrue())
		Expect(found[reconcile.AstarteLowPriorityName]).To(BeTrue())
		Expect(found["unrelated-priority"]).To(BeTrue())

		// Call finalize
		Expect(finalizePriorityClasses(k8sClient, logger)).To(Succeed())

		// Astarte classes should be gone
		exists := &scheduling.PriorityClass{}
		Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: reconcile.AstarteHighPriorityName}, exists)).NotTo(Succeed())
		Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: reconcile.AstarteMidPriorityName}, exists)).NotTo(Succeed())
		Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: reconcile.AstarteLowPriorityName}, exists)).NotTo(Succeed())

		// Other should still exist
		Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: "unrelated-priority"}, &scheduling.PriorityClass{})).To(Succeed())
	})

	It("is a no-op when no Astarte PriorityClasses exist", func() {
		// Ensure none exist
		_ = k8sClient.Delete(context.Background(), &scheduling.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: reconcile.AstarteHighPriorityName}})
		_ = k8sClient.Delete(context.Background(), &scheduling.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: reconcile.AstarteMidPriorityName}})
		_ = k8sClient.Delete(context.Background(), &scheduling.PriorityClass{ObjectMeta: metav1.ObjectMeta{Name: reconcile.AstarteLowPriorityName}})

		Expect(finalizePriorityClasses(k8sClient, logger)).To(Succeed())
	})
})
