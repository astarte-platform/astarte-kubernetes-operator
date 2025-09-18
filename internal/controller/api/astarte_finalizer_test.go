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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/builder"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/integrationutils"
)

var _ = Describe("Astarte Finalizer testing", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "test-astarte-finalizer"
		CustomAstarteNamespace = "astarte-finalizer-tests"
	)

	var cr *apiv2alpha1.Astarte
	var b *builder.TestAstarteBuilder
	var reconciler *AstarteReconciler

	BeforeAll(func() {
		integrationutils.CreateNamespace(k8sClient, CustomAstarteNamespace)
	})

	AfterAll(func() {
		integrationutils.TeardownNamespace(k8sClient, CustomAstarteNamespace)
	})

	BeforeEach(func() {
		// Initialize reconciler
		reconciler = &AstarteReconciler{
			Client: k8sClient,
		}

		b = builder.NewTestAstarteBuilder(CustomAstarteName, CustomAstarteNamespace)
		cr = b.Build()
	})

	AfterEach(func() {
		integrationutils.TeardownResources(context.Background(), k8sClient, CustomAstarteNamespace)
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
