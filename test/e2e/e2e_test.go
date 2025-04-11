/*
This file is part of Astarte.

Copyright 2024 SECO Mind Srl.

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
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/astarte-platform/astarte-kubernetes-operator/internal/version"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/utils"
)

const (
	namespace = "astarte-kubernetes-operator-system"
)

var _ = Describe("controller", Ordered, func() {
	BeforeAll(func() {
		By("installing prometheus operator")
		Expect(utils.InstallPrometheusOperator()).To(Succeed())

		By("installing the cert-manager")
		Expect(utils.InstallCertManager()).To(Succeed())

		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	AfterAll(func() {
		By("uninstalling the Prometheus manager bundle")
		utils.UninstallPrometheusOperator()

		By("uninstalling the cert-manager bundle")
		utils.UninstallCertManager()

		By("removing manager namespace")
		cmd := exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	Context("Astarte Operator", func() {
		It("should run successfully", func() {
			var controllerPodName string
			var err error
			projectDir, _ := utils.GetProjectDir()

			// projectimage stores the name of the image used in the example
			var projectimage = fmt.Sprintf("local-registry/astarte-kubernetes-operator:%s", version.Version)

			By("building the manager(Operator) image")
			cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("loading the the manager(Operator) image on Kind")
			err = utils.LoadImageToKindClusterWithName(projectimage)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("deploying the controller-manager")
			cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectimage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func() error {
				// Get pod name
				cmd = exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(string(podOutput))
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				status, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if string(status) != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, verifyControllerUp, utils.DefaultTimeout, utils.DefaultRetryInterval).Should(Succeed())

			targetTestVersions := map[string]string{
				"1.1": filepath.Join(projectDir, "/test/manifests/api_v1alpha2_astarte_1.1.yaml"),
				"1.2": filepath.Join(projectDir, "/test/manifests/api_v1alpha2_astarte_1.2.yaml"),
			}

			for k, v := range targetTestVersions {
				// let's wait a few seconds to ensure the webhook becomes available
				time.Sleep(4 * time.Second)

				By(fmt.Sprintf("creating an instance of Astarte (CR), version: %s", k))
				EventuallyWithOffset(1,
					utils.InstallAstarte,
					utils.DefaultTimeout,
					utils.DefaultRetryInterval,
				).WithArguments(v).Should(Succeed())

				By(fmt.Sprintf("ensuring that the Astarte v%s health becomes green", k))
				EventuallyWithOffset(1,
					utils.EnsureAstarteHealthGreen,
					utils.DefaultTimeout,
					utils.DefaultRetryInterval,
				).Should(Succeed())

				By(fmt.Sprintf("deleting an instance of Astarte (CR), version: %s", k))
				EventuallyWithOffset(1,
					utils.UninstallAstarte,
					utils.DefaultTimeout,
					utils.DefaultRetryInterval,
				).WithArguments(v).Should(Succeed())
			}

			By("undeploying the controller-manager")
			cmd = exec.Command("make", "undeploy")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the controller-manager pod is not running")
			verifyControllerDown := func() error {
				// Get pod name
				cmd = exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(string(podOutput))
				if len(podNames) != 0 {
					return fmt.Errorf("expect 0 controller pods running, but got %d", len(podNames))
				}

				return nil
			}
			EventuallyWithOffset(1, verifyControllerDown, utils.DefaultTimeout, utils.DefaultRetryInterval).Should(Succeed())

			By("uninstalling CRDs")
			cmd = exec.Command("make", "uninstall")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
		})
	})
})
