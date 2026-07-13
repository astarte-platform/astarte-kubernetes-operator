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
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Astarte Operator", func() {
	Context("CR lifecycle", func() {
		It("should create, verify health, and clean up an Astarte instance", func() {
			projectDir, _ := GetProjectDir()

			targetTestVersions := map[string]string{
				"1.4": filepath.Join(projectDir, "/test/manifests/api_v2alpha1_astarte_1.4.yaml"),
			}

			for k, v := range targetTestVersions {
				By(fmt.Sprintf("creating an instance of Astarte (CR), version: %s", k))
				Expect(InstallAstarte(v)).To(Succeed())

				By(fmt.Sprintf("ensuring that the Astarte v%s health becomes green", k))
				EventuallyWithOffset(1,
					EnsureAstarteWithInfoDump,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())

				By(fmt.Sprintf("deleting an instance of Astarte (CR), version: %s", k))
				Expect(UninstallAstarte(v)).To(Succeed())

				By(fmt.Sprintf("ensuring that every deployment of Astarte v%s is removed", k))
				EventuallyWithOffset(1,
					EnsureAstarteDeployementsAreRemoved,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())

				By(fmt.Sprintf("ensuring that every statefulset of Astarte v%s is removed", k))
				EventuallyWithOffset(1,
					EnsureAstarteStatefulsetsAreRemoved,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())

				By(fmt.Sprintf("ensuring that every configmap of Astarte v%s is removed", k))
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

				By(fmt.Sprintf("ensuring that every secret of Astarte v%s is removed", k))
				EventuallyWithOffset(1,
					EnsureAstarteSecretsAreRemoved,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())

				By(fmt.Sprintf("ensuring that every pvc of Astarte v%s is removed", k))
				EventuallyWithOffset(1,
					EnsureAstartePvcAreRemoved,
					DefaultTimeout,
					DefaultRetryInterval,
				).Should(Succeed())
			}
		})
	})
})
