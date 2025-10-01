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
	"fmt"
	"time"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	CustomAstarteNamespace = "astarte-controller-test"
)

var _ = Describe("Astarte Controller", Ordered, Serial, func() {
	BeforeAll(func() {
		integrationutils.CreateNamespace(k8sClient, CustomAstarteNamespace)
	})

	AfterAll(func() {
		// Do not delete the namespace here to avoid 'NamespaceTerminating' flakiness in subsequent specs
		integrationutils.TeardownResourcesInNamespace(context.Background(), k8sClient, CustomAstarteNamespace)
	})

	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		var controllerReconciler *AstarteReconciler
		var cr *apiv2alpha1.Astarte
		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: CustomAstarteNamespace,
		}

		BeforeEach(func() {
			By("Initializing the controller reconciler")
			controllerReconciler = &AstarteReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Log:      ctrl.Log.WithName("test-reconciler"),
				Recorder: record.NewFakeRecorder(1024),
			}

			By("creating the custom resource for the Kind Astarte")
			cr = baseCr.DeepCopy()
			cr.SetName(resourceName)
			cr.SetNamespace(CustomAstarteNamespace)
			cr.SetResourceVersion("")
			integrationutils.DeployAstarte(k8sClient, cr)
		})

		AfterEach(func() {
			integrationutils.TeardownResourcesInNamespace(ctx, k8sClient, CustomAstarteNamespace)
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
					Namespace: CustomAstarteNamespace,
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Requeue).To(BeFalse())
		})
	})

	Context("Testing the Astarte finalizer", func() {
		const finalizerTestName = "test-finalizer"
		var controllerReconciler *AstarteReconciler
		var cr *apiv2alpha1.Astarte
		ctx := context.Background()

		BeforeEach(func() {
			controllerReconciler = &AstarteReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Log:      ctrl.Log.WithName("test-finalizer-reconciler"),
				Recorder: record.NewFakeRecorder(1024),
			}

			// Create the resource with finalizer and deletion timestamp
			cr = baseCr.DeepCopy()
			cr.SetName(finalizerTestName)
			cr.SetNamespace(CustomAstarteNamespace)
			cr.SetResourceVersion("")
			cr.Finalizers = []string{"astarte-operator.astarte-platform.org/finalizer"}
			cr.DeletionTimestamp = &metav1.Time{Time: time.Now()}

			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		})

		AfterEach(func() {
			integrationutils.TeardownResourcesInNamespace(ctx, k8sClient, CustomAstarteNamespace)
		})

		It("should handle finalization", func() {
			By("Reconciling a resource with deletion timestamp")
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      finalizerTestName,
					Namespace: CustomAstarteNamespace,
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})
	})

	Context("Testing the addFinalizer function", func() {
		const addFinalizerTestName = "test-add-finalizer"
		var controllerReconciler *AstarteReconciler
		var cr *apiv2alpha1.Astarte
		ctx := context.Background()

		BeforeEach(func() {
			controllerReconciler = &AstarteReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Log:      ctrl.Log.WithName("test-add-finalizer-reconciler"),
				Recorder: record.NewFakeRecorder(1024),
			}

			cr = baseCr.DeepCopy()
			cr.SetName(addFinalizerTestName)
			cr.SetNamespace(CustomAstarteNamespace)
			cr.SetResourceVersion("")
			integrationutils.DeployAstarte(k8sClient, cr)
		})

		AfterEach(func() {
			integrationutils.TeardownResourcesInNamespace(ctx, k8sClient, CustomAstarteNamespace)
		})

		It("should add a finalizer to an Astarte resource", func() {
			By("Getting the resource")
			astarteInstance := &apiv2alpha1.Astarte{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: addFinalizerTestName, Namespace: CustomAstarteNamespace}, astarteInstance)
			Expect(err).ToNot(HaveOccurred())

			By("Adding a finalizer to the resource")
			err = controllerReconciler.addFinalizer(astarteInstance)
			Expect(err).ToNot(HaveOccurred())

			By("Checking that the finalizer was added")
			updatedResource := &apiv2alpha1.Astarte{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: addFinalizerTestName, Namespace: CustomAstarteNamespace}, updatedResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedResource.Finalizers).To(ContainElement("astarte.astarte-platform.org/finalizer"))
		})
	})
})

var _ = Describe("Standalone Tests", func() {
	Context("Testing with k8sClient directly", func() {
		It("should handle non-existent resources", func() {
			ctx := context.Background()

			// Ensure the test namespace exists and isn't terminating
			Eventually(func() error {
				ns := &v1.Namespace{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: CustomAstarteNamespace}, ns); err != nil {
					// Try to create if it's missing
					cErr := k8sClient.Create(ctx, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}})
					if apierrors.IsAlreadyExists(cErr) {
						return nil
					}
					return cErr
				}
				// If it's terminating, return an error to retry
				if ns.Status.Phase == v1.NamespaceTerminating {
					return fmt.Errorf("namespace terminating")
				}
				return nil
			}, "20s", Interval).Should(Succeed())

			// Create a test resource
			cr := baseCr.DeepCopy()
			cr.SetName("test-direct-reconcile")
			cr.SetNamespace(CustomAstarteNamespace)
			cr.SetResourceVersion("")
			integrationutils.DeployAstarte(k8sClient, cr)

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
					Namespace: CustomAstarteNamespace,
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			// Cleanup is handled by the test framework
		})
	})
})
