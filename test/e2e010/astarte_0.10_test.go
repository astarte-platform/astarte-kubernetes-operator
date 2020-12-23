/*
  This file is part of Astarte.

  Copyright 2020 Ispirata Srl

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

package e2e010

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	operator "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/utils"
)

var (
	target010Version string = "0.10.2"
	target011Version string = "0.11.1"
)

var _ = Describe("Astarte controller", func() {
	Context("When deploying an Astarte resource", func() {
		It("Should become Green after a new Deployment", func() {
			By("By creating a new Astarte")
			exampleAstarte := utils.AstarteTestResource.DeepCopy()
			exampleAstarte.ObjectMeta.Namespace = namespace
			exampleAstarte.Spec.Version = target010Version
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, exampleAstarte)).Should(Succeed())

			By("By ensuring housekeeping API reaches readiness")
			Eventually(func() error {
				return utils.EnsureDeploymentReadiness(namespace, "example-astarte-housekeeping-api", k8sClient)
			}, utils.DefaultTimeout, utils.DefaultRetryInterval).Should(Succeed())

			By("By ensuring that the Astarte resource Status becomes green")
			Eventually(func() (operator.AstarteClusterHealth, error) {
				return utils.EnsureAstarteBecomesGreen(exampleAstarte.Name, namespace, k8sClient)
			}, utils.DefaultTimeout, utils.DefaultRetryInterval).Should(BeEquivalentTo(operator.AstarteClusterHealthGreen))

			By("By ensuring all Astarte services are up and running")
			Expect(utils.EnsureAstarteServicesReadinessUpTo011(namespace, k8sClient, true)).Should(Succeed())
		})
		It("Should upgrade automatically to Astarte 0.11", func() {
			By("By updating the Astarte Resource Version to a target 0.11 Version")
			ctx := context.Background()
			astarteLookupKey := types.NamespacedName{Name: utils.AstarteTestResource.Name, Namespace: namespace}
			installedAstarte := &operator.Astarte{}

			Expect(k8sClient.Get(ctx, astarteLookupKey, installedAstarte)).Should(Succeed())

			// "Upgrade" the object
			installedAstarte.Spec.Version = target011Version
			Expect(k8sClient.Update(ctx, installedAstarte)).Should(Succeed())

			By("By ensuring that the cluster status advertises the new version")
			Eventually(func() (string, error) {
				return utils.GetAstarteStatusVersion(utils.AstarteTestResource.Name, namespace, k8sClient)
			}, utils.DefaultTimeout, utils.DefaultRetryInterval).Should(BeEquivalentTo(target011Version))

			By("By ensuring that the Astarte Resource has been reconciled at least once after upgrade")
			Eventually(func() (operator.ReconciliationPhase, error) {
				return utils.GetAstarteReconciliationPhase(utils.AstarteTestResource.Name, namespace, k8sClient)
			}, utils.DefaultTimeout, utils.DefaultRetryInterval).Should(BeEquivalentTo(operator.ReconciliationPhaseReconciled))

			By("By ensuring that the Astarte resource Status becomes green")
			Eventually(func() (operator.AstarteClusterHealth, error) {
				return utils.EnsureAstarteBecomesGreen(utils.AstarteTestResource.Name, namespace, k8sClient)
			}, utils.DefaultTimeout, utils.DefaultRetryInterval).Should(BeEquivalentTo(operator.AstarteClusterHealthGreen))

			By("By ensuring all Astarte services are up and running")
			Expect(utils.EnsureAstarteServicesReadinessUpTo011(namespace, k8sClient, true)).Should(Succeed())
		})
		It("Should clean up the cluster after Astarte deletion", func() {
			By("By deleting the Astarte Resource and waiting for services to go down")
			Expect(utils.AstarteDeleteTest(k8sClient, namespace)).Should(Succeed())
		})
	})
})
