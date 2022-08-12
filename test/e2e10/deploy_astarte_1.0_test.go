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

package e2e10

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/utils"
)

var target10Version string = "1.0.3"

var _ = Describe("Astarte controller", func() {
	Context("When deploying an Astarte resource", func() {
		It("Should become Green after a new Deployment", func() {
			By("By creating a new Astarte")
			exampleAstarte := utils.AstarteTestResource.DeepCopy()
			exampleAstarte.ObjectMeta.Namespace = namespace
			exampleAstarte.Spec.Version = target10Version
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, exampleAstarte)).Should(Succeed())

			By("By ensuring housekeeping reaches readiness")
			Eventually(func() error {
				return utils.EnsureDeploymentReadiness(namespace, "example-astarte-housekeeping", k8sClient)
			}, utils.DefaultTimeout, utils.DefaultRetryInterval).Should(Succeed())

			By("By ensuring that the Astarte resource Status becomes green")
			Eventually(func() (v1alpha1.AstarteClusterHealth, error) {
				return utils.EnsureAstarteBecomesGreen(utils.AstarteTestResource.Name, namespace, k8sClient)
			}, utils.DefaultTimeout, utils.DefaultRetryInterval).Should(BeEquivalentTo(v1alpha1.AstarteClusterHealthGreen))

			By("By ensuring all Astarte services are up and running")
			Expect(utils.EnsureAstarteServicesReadinessUpTo10(namespace, k8sClient)).Should(Succeed())
		})
		It("Should clean up the cluster after Astarte deletion", func() {
			By("By deleting the Astarte Resource and waiting for services to go down")
			Expect(utils.AstarteDeleteTest(k8sClient, namespace)).Should(Succeed())
		})
	})

})
