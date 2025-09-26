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

	"go.openly.dev/pointy"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("CFSSL testing", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "example-astarte"
		CustomAstarteNamespace = "cfssl-test"
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
		integrationutils.DeployAstarte(k8sClient, cr)
	})

	AfterEach(func() {
		integrationutils.TeardownResources(context.Background(), k8sClient, CustomAstarteNamespace)
	})

	Describe("Test EnsureCFSSL", func() {
		It("should create/update the CFSSL pod", func() {
			deploymentName := CustomAstarteName + "-cfssl"
			cr.Spec.CFSSL.Deploy = pointy.Bool(true)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())

			// First reconciliation
			Expect(EnsureCFSSL(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Check that the deployment is created
			cfsslDeployment := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: CustomAstarteNamespace}, cfsslDeployment)
			}, Timeout, Interval).Should(Succeed())

			// Store the checksum
			initialChecksum := cfsslDeployment.Spec.Template.Annotations["checksum/config"]
			Expect(initialChecksum).ToNot(BeEmpty())

			// Update the Astarte CR
			cr.Spec.CFSSL.CaExpiry = "1h"
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())

			// Second reconciliation
			Expect(EnsureCFSSL(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Check that the deployment is updated
			Eventually(func() string {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: CustomAstarteNamespace}, cfsslDeployment)
				if err != nil {
					return ""
				}
				return cfsslDeployment.Spec.Template.Annotations["checksum/config"]
			}, Timeout, Interval).ShouldNot(Equal(initialChecksum))
		})
	})
})
