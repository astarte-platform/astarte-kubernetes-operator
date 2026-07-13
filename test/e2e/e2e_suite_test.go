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
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	"github.com/astarte-platform/astarte-kubernetes-operator/internal/version"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(5 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting astarte-kubernetes-operator suite\n")
	RunSpecs(t, "Astarte Operator e2e suite")
}

var _ = BeforeSuite(func() {
	By("installing prometheus operator")
	Expect(InstallPrometheusOperator()).To(Succeed())

	By("installing the cert-manager")
	Expect(InstallCertManager()).To(Succeed())

	By("installing rabbitmq cluster operator")
	Expect(InstallRabbitMQClusterOperator()).To(Succeed())

	By("installing openbao cluster helm chart")
	Expect(DeployOpenBao()).To(Succeed())

	By("deploying the rendezvous server")
	Expect(DeployRendezvousServer()).To(Succeed())

	By("deploying the RabbitMQ cluster")
	Expect(DeployRabbitMQCluster()).To(Succeed())

	By("creating the RabbitMq connection secret")
	Expect(CreateRabbitMQConnectionSecret()).To(Succeed())

	By("installing scylla operator")
	Expect(InstallScyllaOperator()).To(Succeed())

	By("deploying the Scylla cluster")
	Expect(DeployScyllaCluster()).To(Succeed())

	By("creating the Scylla connection secret")
	Expect(CreateScyllaConnectionSecret()).To(Succeed())

	By("creating astarte operator namespace")
	cmd := exec.Command("kubectl", "create", "ns", operatorNamespace)
	_, _ = Run(cmd)

	projectimage := fmt.Sprintf("local-registry/astarte-kubernetes-operator:%s", version.Version)

	By("building the manager(Operator) image")
	cmd = exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
	_, err := Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("loading the manager(Operator) image on Kind")
	err = LoadImageToKindClusterWithName(projectimage)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("deploying the controller-manager")
	cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectimage))
	_, err = Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("validating that the controller-manager pod is running as expected")
	EventuallyWithOffset(1, verifyControllerPodRunning, DefaultTimeout, DefaultRetryInterval).Should(Succeed())

	By("waiting for the controller deployment to become available")
	cmd = exec.Command("kubectl", "wait", "deployment",
		"--for", "condition=Available",
		"astarte-kubernetes-operator-controller-manager",
		"-n", operatorNamespace,
		"--timeout", "5m",
	)
	_, err = Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("dump Astarte Operator info and logs for debugging")
	DumpAstarteOperatorDebuggingInfo(operatorNamespace)

	By("undeploying the controller-manager")
	cmd := exec.Command("make", "undeploy")
	_, _ = Run(cmd)

	By("validating that the controller-manager pod is not running")
	Eventually(verifyControllerPodNotRunning, DefaultTimeout, DefaultRetryInterval).Should(Succeed())

	By("uninstalling CRDs")
	cmd = exec.Command("make", "uninstall")
	_, _ = Run(cmd)

	By("uninstalling the Prometheus manager bundle")
	UninstallPrometheusOperator()

	By("uninstalling rabbitmq cluster")
	UninstallRabbitMQCluster()

	By("uninstalling rabbitmq cluster operator")
	UninstallRabbitMQClusterOperator()

	By("uninstalling scylla operator")
	UninstallScyllaOperator()

	By("uninstalling the cert-manager bundle")
	UninstallCertManager()

	By("uninstalling OpenBao")
	cmd = exec.Command("helm", "uninstall", "openbao", "--namespace", "openbao")
	_, _ = Run(cmd)
	cmd = exec.Command("kubectl", "delete", "namespace", "openbao")
	_, _ = Run(cmd)

	By("uninstalling the rendezvous server")
	cmd = exec.Command("kubectl", "delete", "namespace", rendezvousServerNamespace)
	_, _ = Run(cmd)

	By("removing astarte operator namespace")
	cmd = exec.Command("kubectl", "delete", "ns", operatorNamespace)
	_, _ = Run(cmd)
})
