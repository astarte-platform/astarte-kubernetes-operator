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

package e2e

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Astarte Operator", func() {
	Context("CR lifecycle", func() {
		targetTestVersions := map[string]string{
			"1.4": "test/manifests/api_v2alpha1_astarte_1.4.yaml",
		}

		for k, v := range targetTestVersions {
			version := k
			manifestPath := v

			It("should create, verify health, and clean up an Astarte v"+version+" instance", func() {
				projectDir, _ := GetProjectDir()
				manifest := filepath.Join(projectDir, manifestPath)

				By("creating an instance of Astarte (CR)")
				Expect(InstallAstarte(manifest)).To(Succeed())

				By("ensuring that the Astarte health becomes green")
				EventuallyWithOffset(1,
					EnsureAstarteWithInfoDump,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())

				By("deleting an instance of Astarte (CR)")
				Expect(UninstallAstarte(manifest)).To(Succeed())

				By("ensuring that every deployment of Astarte is removed")
				EventuallyWithOffset(1,
					EnsureAstarteDeployementsAreRemoved,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())

				By("ensuring that every statefulset of Astarte is removed")
				EventuallyWithOffset(1,
					EnsureAstarteStatefulsetsAreRemoved,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())

				By("ensuring that every configmap of Astarte is removed")
				EventuallyWithOffset(1,
					EnsureAstarteConfigmapsAreRemoved,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())

				By("deleting the RabbitMq connection secret")
				EventuallyWithOffset(1,
					DeleteRabbitMQConnectionSecret,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())

				By("deleting the Scylla connection secret")
				EventuallyWithOffset(1,
					DeleteScyllaConnectionSecret,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())

				By("deleting the Vault connection secret")
				EventuallyWithOffset(1,
					DeleteVaultConnectionSecret,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())

				By("ensuring that every secret of Astarte is removed")
				EventuallyWithOffset(1,
					EnsureAstarteSecretsAreRemoved,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())

				By("ensuring that every pvc of Astarte is removed")
				EventuallyWithOffset(1,
					EnsureAstartePvcAreRemoved,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())
			})
		}
	})
})
