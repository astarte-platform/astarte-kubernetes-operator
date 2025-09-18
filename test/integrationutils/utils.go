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

//nolint:lll
package integrationutils

import (
	"context"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const Timeout string = "30s"
const Interval string = "1s"

const AstarteHighPriorityName string = "astarte-high-priority-non-preemptive"
const AstarteMidPriorityName string = "astarte-mid-priority-non-preemptive"
const AstarteLowPriorityName string = "astarte-low-priority-non-preemptive"

func TeardownResources(ctx context.Context, k8sClient client.Client, namespace string) {
	teardownAstarte(ctx, k8sClient, namespace)
	teardownDeployments(ctx, k8sClient, namespace)
	teardownStatefulSets(ctx, k8sClient, namespace)
	teardownConfigMaps(ctx, k8sClient, namespace)
	teardownSecrets(ctx, k8sClient, namespace)
	teardownPVCs(ctx, k8sClient, namespace)
	teardownServices(ctx, k8sClient, namespace)
	teardownRBAC(k8sClient, namespace)
	teardownPriorityClasses(k8sClient)
}

func teardownAstarte(ctx context.Context, k8sClient client.Client, namespace string) {
	// Since we cannot import the api package here due to circular dependencies, we use unstructured.UnstructuredList
	// instead of: &v2alpha1.AstarteList{}
	astartes := &unstructured.UnstructuredList{}
	astartes.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "api.astarte-platform.org",
		Version: "v2alpha1",
		Kind:    "AstarteList",
	})

	Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: namespace})).To(Succeed())

	for _, a := range astartes.Items {
		// Remove finalizer to avoid issues
		a.SetFinalizers([]string{})
		Eventually(func() error {
			return k8sClient.Update(ctx, &a)
		}, Timeout, Interval).Should(Succeed())

		// Delete the Astarte resource
		Eventually(func() error {
			return k8sClient.Delete(ctx, &a)
		}, Timeout, Interval).Should(Succeed())

		// Ensure the Astarte resource is deleted
		Eventually(func() error {
			astarte := &unstructured.Unstructured{}
			astarte.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "api.astarte-platform.org",
				Version: "v2alpha1",
				Kind:    "Astarte",
			})
			return k8sClient.Get(context.Background(), client.ObjectKey{Name: a.GetName(), Namespace: a.GetNamespace()}, astarte)
		}, Timeout, Interval).ShouldNot(Succeed())
	}
}

func teardownDeployments(ctx context.Context, k8sClient client.Client, namespace string) {
	deployments := &appsv1.DeploymentList{}
	Expect(k8sClient.List(context.Background(), deployments, &client.ListOptions{Namespace: namespace})).To(Succeed())

	for _, d := range deployments.Items {
		Eventually(func() error {
			return k8sClient.Delete(ctx, &d)
		}, Timeout, Interval).Should(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.Background(), client.ObjectKey{Name: d.Name, Namespace: d.Namespace}, &appsv1.Deployment{})
		}, Timeout, Interval).ShouldNot(Succeed())
	}
}

func teardownStatefulSets(ctx context.Context, k8sClient client.Client, namespace string) {
	statefulsets := &appsv1.StatefulSetList{}
	Expect(k8sClient.List(context.Background(), statefulsets, &client.ListOptions{Namespace: namespace})).To(Succeed())

	for _, s := range statefulsets.Items {
		Eventually(func() error {
			return k8sClient.Delete(ctx, &s)
		}, Timeout, Interval).Should(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.Background(), client.ObjectKey{Name: s.Name, Namespace: s.Namespace}, &appsv1.StatefulSet{})
		}, Timeout, Interval).ShouldNot(Succeed())
	}
}

func teardownConfigMaps(ctx context.Context, k8sClient client.Client, namespace string) {
	configMaps := &v1.ConfigMapList{}
	Expect(k8sClient.List(context.Background(), configMaps, &client.ListOptions{Namespace: namespace})).To(Succeed())

	for _, c := range configMaps.Items {
		Eventually(func() error {
			return k8sClient.Delete(ctx, &c)
		}, Timeout, Interval).Should(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.Background(), client.ObjectKey{Name: c.Name, Namespace: c.Namespace}, &v1.ConfigMap{})
		}, Timeout, Interval).ShouldNot(Succeed())
	}
}

func teardownSecrets(ctx context.Context, k8sClient client.Client, namespace string) {
	secrets := &v1.SecretList{}

	Expect(k8sClient.List(context.Background(), secrets, &client.ListOptions{Namespace: namespace})).To(Succeed())

	for _, s := range secrets.Items {
		Eventually(func() error {
			return k8sClient.Delete(ctx, &s)
		}, Timeout, Interval).Should(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.Background(), client.ObjectKey{Name: s.Name, Namespace: s.Namespace}, &v1.Secret{})
		}, Timeout, Interval).ShouldNot(Succeed())
	}
}

func teardownPVCs(ctx context.Context, k8sClient client.Client, namespace string) {
	pvcs := &v1.PersistentVolumeClaimList{}
	Expect(k8sClient.List(context.Background(), pvcs, &client.ListOptions{Namespace: namespace})).To(Succeed())

	for _, p := range pvcs.Items {
		Eventually(func() error {
			return k8sClient.Delete(ctx, &p)
		}, Timeout, Interval).Should(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.Background(), client.ObjectKey{Name: p.Name, Namespace: p.Namespace}, &v1.PersistentVolumeClaim{})
		}, Timeout, Interval).ShouldNot(Succeed())
	}
}

func teardownServices(ctx context.Context, k8sClient client.Client, namespace string) {
	services := &v1.ServiceList{}
	Expect(k8sClient.List(context.Background(), services, &client.ListOptions{Namespace: namespace})).To(Succeed())
	for _, s := range services.Items {
		if s.Name != "kubernetes" {
			Eventually(func() error {
				return k8sClient.Delete(ctx, &s)
			}, Timeout, Interval).Should(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), client.ObjectKey{Name: s.Name, Namespace: s.Namespace}, &v1.Service{})
			}, Timeout, Interval).ShouldNot(Succeed())
		}
	}
}

func teardownRBAC(k8sClient client.Client, namespace string) {
	serviceAccounts := &v1.ServiceAccountList{}
	Expect(k8sClient.List(context.Background(), serviceAccounts, &client.ListOptions{Namespace: namespace})).To(Succeed())
	for _, sa := range serviceAccounts.Items {
		serviceAccount := sa // Capture loop variable
		if err := k8sClient.Delete(context.Background(), &serviceAccount); err != nil && !apierrors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	}
	// Wait for all service accounts to be deleted
	Eventually(func() int {
		saList := &v1.ServiceAccountList{}
		if err := k8sClient.List(context.Background(), saList, &client.ListOptions{Namespace: namespace}); err != nil {
			return -1
		}
		return len(saList.Items)
	}, Timeout, Interval).Should(Equal(0))

	roles := &rbacv1.RoleList{}
	Expect(k8sClient.List(context.Background(), roles, &client.ListOptions{Namespace: namespace})).To(Succeed())
	for _, r := range roles.Items {
		role := r // Capture loop variable
		if err := k8sClient.Delete(context.Background(), &role); err != nil && !apierrors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	}
	// Wait for all roles to be deleted
	Eventually(func() int {
		roleList := &rbacv1.RoleList{}
		if err := k8sClient.List(context.Background(), roleList, &client.ListOptions{Namespace: namespace}); err != nil {
			return -1
		}
		return len(roleList.Items)
	}, Timeout, Interval).Should(Equal(0))

	roleBindings := &rbacv1.RoleBindingList{}
	Expect(k8sClient.List(context.Background(), roleBindings, &client.ListOptions{Namespace: namespace})).To(Succeed())
	for _, rb := range roleBindings.Items {
		roleBinding := rb // Capture loop variable
		if err := k8sClient.Delete(context.Background(), &roleBinding); err != nil && !apierrors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	}
	// Wait for all role bindings to be deleted
	Eventually(func() int {
		rbList := &rbacv1.RoleBindingList{}
		if err := k8sClient.List(context.Background(), rbList, &client.ListOptions{Namespace: namespace}); err != nil {
			return -1
		}
		return len(rbList.Items)
	}, Timeout, Interval).Should(Equal(0))
}

func teardownPriorityClasses(k8sClient client.Client) {
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
		}, Timeout, Interval).Should(BeTrue())
	}
}

func TeardownNamespace(k8sClient client.Client, namespace string) {
	if namespace != "default" {
		// Just give the namespace deletion a try, do not repeat on timeout
		// as it would return "namespace terminating" errors
		ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
		_ = k8sClient.Delete(context.Background(), ns)
	}
}

func CreateNamespace(k8sClient client.Client, namespace string) {
	if namespace != "default" {
		ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
		Eventually(func() error {
			err := k8sClient.Create(context.Background(), ns)
			if apierrors.IsAlreadyExists(err) {
				return nil
			}
			return err
		}, Timeout, Interval).Should(Succeed())
	}
}

// DeployCustomResource creates any custom resource and waits for it to be available
// This works with any Kubernetes custom resource type, including Astarte CRs
func DeployCustomResource(k8sClient client.Client, cr client.Object) {
	Eventually(func() error {
		return k8sClient.Create(context.Background(), cr)
	}, Timeout, Interval).Should(Succeed())

	Eventually(func() error {
		return k8sClient.Get(context.Background(), client.ObjectKeyFromObject(cr), cr)
	}, Timeout, Interval).Should(Succeed())
}

func DeleteCustomResource(ctx context.Context, k8sClient client.Client, cr client.Object) {
	// Remove finalizers to avoid deletion issues
	cr.SetFinalizers([]string{})
	Eventually(func() error {
		return k8sClient.Update(ctx, cr)
	}, Timeout, Interval).Should(Succeed())

	// Delete the resource
	Eventually(func() error {
		return k8sClient.Delete(ctx, cr)
	}, Timeout, Interval).Should(Succeed())

	// Ensure the resource is deleted
	Eventually(func() error {
		return k8sClient.Get(ctx, client.ObjectKeyFromObject(cr), cr)
	}, Timeout, Interval).ShouldNot(Succeed())
}

// Wrapper around DeployCustomResource for Astarte resources
func DeployAstarte(k8sClient client.Client, cr client.Object, namespace string) {
	CreateNamespace(k8sClient, namespace)
	cr.SetNamespace(namespace)
	cr.SetResourceVersion("")
	DeployCustomResource(k8sClient, cr)
}
