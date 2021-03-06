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

package oldoperator

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/astarte-platform/astarte-kubernetes-operator/apis/api/commontypes"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/utils"
)

var _ = Describe("Astarte controller", func() {
	Context("When deploying a new-style Astarte Operator in a cluster with an existing Astarte Resource", func() {
		It("Should reconcile the resource automatically", func() {
			By("By converting it and turning the cluster green")

			Eventually(func() (commontypes.AstarteClusterHealth, error) {
				return utils.EnsureAstarteBecomesGreen(utils.AstarteTestResource.Name, namespace, k8sClient)
			}, utils.DefaultTimeout, utils.DefaultRetryInterval).Should(BeEquivalentTo(commontypes.AstarteClusterHealthGreen))

			By("By ensuring all Astarte services are up and running")
			Expect(utils.EnsureAstarteServicesReadinessUpTo011(namespace, k8sClient, true)).Should(Succeed())
		})
		It("Should clean up the cluster after Astarte deletion", func() {
			By("By deleting the Astarte Resource and waiting for services to go down")
			Expect(utils.AstarteDeleteTest(k8sClient, namespace)).Should(Succeed())
		})
	})
})
