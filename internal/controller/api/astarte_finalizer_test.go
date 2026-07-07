/*
This file is part of Astarte.

Copyright 2020-26 SECO Mind Srl.

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
	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
)

var _ = Describe("Astarte Finalizer testing", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "test-astarte-finalizer"
		CustomAstarteNamespace = "astarte-finalizer-tests"
	)

	var reconciler *AstarteReconciler
	var cr *apiv2alpha1.Astarte

	BeforeAll(func() {
		integrationutils.CreateNamespace(k8sClient, CustomAstarteNamespace)
		DeferCleanup(func() {
			integrationutils.DeleteNamespace(k8sClient, CustomAstarteNamespace)
		})
	})

	BeforeEach(func() {
		reconciler = &AstarteReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
		cr = baseCr.DeepCopy()
		cr.SetName(CustomAstarteName)
		cr.SetNamespace(CustomAstarteNamespace)
		cr.SetResourceVersion("")
		integrationutils.DeployAstarte(k8sClient, cr)
		DeferCleanup(func() {
			integrationutils.TeardownResourcesInNamespace(ctx, k8sClient, CustomAstarteNamespace)
		})
	})

	Describe("Test HandleFinalization", func() {
		It("should successfully finalize Astarte instance with finalizer", func() {
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}).Should(Succeed())

			cr.SetFinalizers([]string{astarteFinalizer})

			Eventually(func() error {
				return k8sClient.Update(ctx, cr)
			}).Should(Succeed())

			Eventually(func() []string {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
				if err != nil {
					return nil
				}
				return cr.GetFinalizers()
			}).Should(ContainElement(astarteFinalizer))

			Eventually(func() error {
				return k8sClient.Delete(ctx, cr)
			}).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
				if err != nil {
					return false
				}
				return cr.GetDeletionTimestamp() != nil
			}).Should(BeTrue())

			result, err := reconciler.handleFinalization(cr)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			Expect(cr.GetFinalizers()).ToNot(ContainElement(astarteFinalizer))

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, &apiv2alpha1.Astarte{})
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())
		})
	})

	Describe("Test AddFinalizer", func() {
		It("should successfully add finalizer to CR", func() {
			crNew := cr.DeepCopy()
			crNew.Name = "test-astarte-finalizer-add"
			crNew.SetFinalizers([]string{})
			crNew.ResourceVersion = ""

			Expect(k8sClient.Create(ctx, crNew)).To(Succeed())

			Eventually(func() []string {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: crNew.Name, Namespace: crNew.Namespace}, crNew)
				if err != nil {
					return nil
				}
				return crNew.GetFinalizers()
			}).ShouldNot(ContainElement(astarteFinalizer))

			err := reconciler.addFinalizer(crNew)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() []string {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: crNew.Name, Namespace: crNew.Namespace}, crNew)
				if err != nil {
					return nil
				}
				return crNew.GetFinalizers()
			}).Should(ContainElement(astarteFinalizer))
		})
	})
})
