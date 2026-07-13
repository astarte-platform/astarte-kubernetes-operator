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
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
)

const (
	prometheusOperatorVersion = "v0.72.0"
	prometheusOperatorURL     = "https://github.com/prometheus-operator/prometheus-operator/" +
		"releases/download/%s/bundle.yaml"

	certmanagerVersion = "v1.16.3"
	certmanagerURLTmpl = "https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager.yaml"

	// the astarteName must match the value of the
	// test/samples/api_v2alpha1_astarte_1*.yaml files
	astarteName      = "example-astarte"
	astarteNamespace = "example-astarte-ns"

	// DefaultRetryInterval applied to all tests
	DefaultRetryInterval time.Duration = time.Second * 10
	// DefaultTimeout applied to all tests
	DefaultTimeout time.Duration = time.Second * 420
	// DefaultCleanupRetryInterval applied to all tests
	DefaultCleanupRetryInterval time.Duration = time.Second * 1
	// DefaultCleanupTimeout applied to all tests
	DefaultCleanupTimeout time.Duration = time.Second * 5

	rabbitmqClusterOperatorVersion = "v2.16.0"                                                                                //nolint:all
	rabbitmqClusterOperatorURL     = "https://github.com/rabbitmq/cluster-operator/releases/download/%s/cluster-operator.yml" //nolint:all
	rabbitmqNamespace              = "rabbitmq"

	scyllaOperatorVersion = "v1.17.1"
	scyllaOperatorURL     = "https://raw.githubusercontent.com/scylladb/scylla-operator/%s/deploy/operator.yaml"
	scyllaNamespace       = "scylla"

	rendezvousServerNamespace = "rendezvous-server"

	openbaoVersion = "0.4.0"
)

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

func verifyControllerPodRunning() error {
	cmd := exec.Command("kubectl", "get",
		"pods", "-l", "control-plane=controller-manager",
		"-o", "jsonpath={.items[?(@.status.phase=='Running')].metadata.name}",
		"-n", operatorNamespace,
	)
	output, err := Run(cmd)
	if err != nil {
		return err
	}
	podNames := GetNonEmptyLines(string(output))
	if len(podNames) != 1 {
		return fmt.Errorf("expect 1 running controller pod, got %d", len(podNames))
	}
	if !strings.Contains(podNames[0], "controller-manager") {
		return fmt.Errorf("unexpected pod name: %s", podNames[0])
	}
	return nil
}

func DeployRendezvousServer() error {
	if err := EnsureNamespaceExists(rendezvousServerNamespace); err != nil {
		return fmt.Errorf("failed to ensure namespace %s exists: %w", rendezvousServerNamespace, err)
	}

	projectDir, err := GetProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %w", err)
	}
	cmd := exec.Command("kubectl", "apply", "-n", rendezvousServerNamespace,
		"-f", filepath.Join(projectDir, "test/manifests/dependencies/rendezvous-server"))
	_, err = Run(cmd)
	return err
}

// InstallPrometheusOperator installs the prometheus Operator to be used to export the enabled metrics.
func InstallPrometheusOperator() error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "create", "-f", url)
	_, err := Run(cmd)
	return err
}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) ([]byte, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return output, nil
}

// UninstallPrometheusOperator uninstalls the prometheus
func UninstallPrometheusOperator() {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// UninstallCertManager uninstalls the cert manager
func UninstallCertManager() {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// InstallCertManager installs the cert manager bundle.
func InstallCertManager() error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)

	_, err := Run(cmd)
	return err
}

// LoadImageToKindClusterWithName loads a local docker image to the kind cluster
func LoadImageToKindClusterWithName(name string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := Run(cmd)
	return err
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.ReplaceAll(wd, "/test/e2e", "")
	return wd, nil
}

// InstallAstarte installs the Astarte CR.
func InstallAstarte(manifestPath string) error {
	err := EnsureNamespaceExists(astarteNamespace)
	if err != nil {
		return fmt.Errorf("failed to ensure namespace %s exists: %w", astarteNamespace, err)
	}

	cmd := exec.Command("kubectl", "apply", "-f", manifestPath, "--namespace", astarteNamespace)
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to install Astarte: %w", err)
	}
	return err
}

func EnsureAstarteHealthGreen() error {
	// Wait for astarte health to be green.
	cmd := exec.Command("kubectl", "wait", "astartes.v2alpha1.api.astarte-platform.org", astarteName,
		"--for", fmt.Sprintf("jsonpath={.status.health}=%s", apiv2alpha1.AstarteClusterHealthGreen),
		"--namespace", astarteNamespace,
		"--timeout", "10m",
	)

	_, err := Run(cmd)
	return err
}

func EnsureAstarteWithInfoDump() error {
	err := EnsureAstarteHealthGreen()
	if err != nil {
		DumpAstarteDebuggingInfo()
	}
	return err
}

func DeleteRabbitMQConnectionSecret() error {
	cmd := exec.Command("kubectl", "delete", "secret",
		"rabbitmq-connection-secret",
		"-n", astarteNamespace,
	)
	_, err := Run(cmd)
	return err
}

func DeleteScyllaConnectionSecret() error {
	cmd := exec.Command("kubectl", "delete", "secret",
		"scylladb-connection-secret",
		"-n", astarteNamespace,
	)
	_, err := Run(cmd)
	return err
}

func DeleteVaultConnectionSecret() error {
	cmd := exec.Command("kubectl", "delete", "secret",
		"vault-connection-string",
		"-n", astarteNamespace,
	)
	_, err := Run(cmd)
	return err
}

func EnsureAstarteDeployementsAreRemoved() error {
	// Wait for all Astarte resources to be deleted.
	cmd := exec.Command("kubectl", "get", "deployments.apps",
		"-n", astarteNamespace,
		"-l", "component=astarte",
		"-o", "go-template={{ range .items }}"+
			"{{ if not .metadata.deletionTimestamp }}"+
			"{{ .metadata.name }}"+
			"{{ \"\\n\" }}{{ end }}{{ end }}",
	)

	cmdOut, err := Run(cmd)

	if err != nil {
		return err
	}

	resources := GetNonEmptyLines(string(cmdOut))
	if len(resources) != 0 {
		return fmt.Errorf("expect 0 deployments running, but got %d", len(resources))
	}

	return nil
}

func EnsureAstarteStatefulsetsAreRemoved() error {
	// Wait for all Astarte resources to be deleted.
	cmd := exec.Command("kubectl", "get", "statefulsets.apps",
		"-n", astarteNamespace,
		"-l", "component=astarte",
		"-o", "go-template={{ range .items }}"+
			"{{ if not .metadata.deletionTimestamp }}"+
			"{{ .metadata.name }}"+
			"{{ \"\\n\" }}{{ end }}{{ end }}",
	)

	cmdOut, err := Run(cmd)

	if err != nil {
		return err
	}

	resources := GetNonEmptyLines(string(cmdOut))
	if len(resources) != 0 {
		return fmt.Errorf("expect 0 stateful sets running, but got %d", len(resources))
	}

	return nil
}

func EnsureAstarteConfigmapsAreRemoved() error {
	// Wait for all Astarte resources to be deleted.
	cmd := exec.Command("kubectl", "get", "configmaps",
		"-n", astarteNamespace,
		"-o", "go-template={{ range .items }}"+
			"{{ .metadata.name }}{{ end }}")

	cmdOut, err := Run(cmd)
	if err != nil {
		return err
	}

	resources := GetNonEmptyLines(string(cmdOut))

	// From Kubernetes 1.20+, a configmap named "kube-root-ca.crt" is created by default
	// in every namespace. If it's the only item left, let it be.
	if len(resources) > 1 || (len(resources) == 1 && resources[0] != "kube-root-ca.crt") {
		// Do not account for kube-root-ca.crt when returning the error.
		return fmt.Errorf("some configmaps are still present")
	}

	return nil
}

func EnsureAstarteSecretsAreRemoved() error {
	cmd := exec.Command("kubectl", "get", "secrets",
		"-n", astarteNamespace,
		"-o", "go-template={{ range .items }}"+
			"{{ .metadata.name }}{{ end }}")

	cmdOut, err := Run(cmd)

	if err != nil {
		return err
	}

	resources := GetNonEmptyLines(string(cmdOut))

	if len(resources) != 0 {
		return fmt.Errorf("expect 0 secrets but got %d", len(resources))
	}

	return nil
}

func EnsureAstartePvcAreRemoved() error {
	cmd := exec.Command("kubectl", "get", "pvc",
		"-n", astarteNamespace,
		"-o", "go-template={{ range .items }}"+
			"{{ .metadata.name }}{{ end }}")

	cmdOut, err := Run(cmd)

	if err != nil {
		return err
	}

	resources := GetNonEmptyLines(string(cmdOut))

	if len(resources) != 0 {
		return fmt.Errorf("expect 0 pvc but got %d", len(resources))
	}

	return nil
}

func UninstallAstarte(manifestPath string) error {
	cmd := exec.Command("kubectl", "delete", "-f", manifestPath, "--namespace", astarteNamespace)
	_, err := Run(cmd)
	return err
}

func DeployRabbitMQCluster() error {
	prj, err := GetProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %w", err)
	}

	// Create RabbitMQ cluster namespace
	err = EnsureNamespaceExists(rabbitmqNamespace)
	if err != nil {
		return fmt.Errorf("failed to ensure namespace %s exists: %w", rabbitmqNamespace, err)
	}

	// Check if manifest file exists
	manifestPath := fmt.Sprintf("%s/test/manifests/dependencies/rabbitmq-cluster.yaml", prj)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest file %s does not exist", manifestPath)
	}

	// Deploy RabbitMQ cluster with the manifest
	cmd := exec.Command("kubectl", "apply", "-f", manifestPath, "-n", rabbitmqNamespace)
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to deploy RabbitMQ cluster: %w", err)
	}

	cmd = exec.Command("kubectl", "wait",
		"--for=create",
		"statefulset",
		"rabbitmq-server",
		"-n", rabbitmqNamespace,
		"--timeout", "5m",
	)

	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to wait for RabbitMQ cluster pods to be created: %w", err)
	}

	// Wait for RabbitMQ cluster to be ready
	cmd = exec.Command("kubectl", "wait",
		"--for", "condition=ready",
		"pod",
		"-l", "app.kubernetes.io/component=rabbitmq",
		"-n", rabbitmqNamespace,
		"--timeout", "5m",
	)

	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to wait for RabbitMQ cluster pods to be ready: %w", err)
	}

	return nil
}

func DeployOpenBao() error {
	cmd := exec.Command("helm", "repo", "add", "openbao", "https://openbao.github.io/openbao-helm")
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to add OpenBao Helm repository: %w", err)
	}

	openbaoNamespace := "openbao"

	cmd = exec.Command("helm", "install", "openbao", "openbao/openbao",
		"--namespace", openbaoNamespace, "--create-namespace",
		"--version", openbaoVersion,
		"--set", "server.dev.enabled=true",
	)
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to install OpenBao: %w", err)
	}

	secretData := map[string]string{
		"connection-string": "root",
	}

	if err := CreateSecret("vault-connection-string", astarteNamespace, secretData); err != nil {
		return fmt.Errorf("failed to create Vault connection secret: %w", err)
	}

	cmd = exec.Command("kubectl", "wait", "--for=condition=Ready", "pod",
		"-l", "app.kubernetes.io/instance=openbao", "-n", openbaoNamespace, "--timeout", "5m")
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to wait for OpenBao pod to be ready: %w", err)
	}

	return nil
}

func CreateRabbitMQConnectionSecret() error {
	// Get RabbitMQ credentials
	// kubectl get secret -n rabbitmq rabbitmq-default-user -o jsonpath='{.data.username}' | base64 -d
	// kubectl get secret -n rabbitmq rabbitmq-default-user -o jsonpath='{.data.password}' | base64 -d

	cmd := exec.Command("kubectl", "get", "secret", "-n", rabbitmqNamespace, "rabbitmq-default-user",
		"-o", "jsonpath={.data.username}")
	usrB64, err := Run(cmd)
	if err != nil {
		return fmt.Errorf("failed to get RabbitMQ username: %w", err)
	}

	cmd = exec.Command("kubectl", "get", "secret", "-n", rabbitmqNamespace, "rabbitmq-default-user",
		"-o", "jsonpath={.data.password}")
	pwdB64, err := Run(cmd)
	if err != nil {
		return fmt.Errorf("failed to get RabbitMQ password: %w", err)
	}

	usr, err := base64.StdEncoding.DecodeString(string(usrB64))
	if err != nil {
		return fmt.Errorf("failed to decode rabbitmq username: %w", err)
	}
	pwd, err := base64.StdEncoding.DecodeString(string(pwdB64))
	if err != nil {
		return fmt.Errorf("failed to decode rabbitmq password: %w", err)
	}

	secretData := map[string]string{
		"username": strings.TrimSpace(string(usr)),
		"password": strings.TrimSpace(string(pwd)),
	}

	if err := CreateSecret("rabbitmq-connection-secret", astarteNamespace, secretData); err != nil {
		return fmt.Errorf("failed to create RabbitMQ connection secret: %w", err)
	}

	return nil
}

func InstallRabbitMQClusterOperator() error {
	url := fmt.Sprintf(rabbitmqClusterOperatorURL, rabbitmqClusterOperatorVersion)
	cmd := exec.Command("kubectl", "create", "-f", url)

	if _, err := Run(cmd); err != nil {
		return err
	}

	// Wait for CRDs
	crd := "rabbitmqclusters.rabbitmq.com"
	cmd = exec.Command("kubectl", "wait",
		"--for", "condition=established",
		fmt.Sprintf("crd/%s", crd),
		"--timeout", "5m",
	)

	if _, err := Run(cmd); err != nil {
		return err
	}

	// Wait for operator deployment
	cmd = exec.Command("kubectl",
		"-n", "rabbitmq-system",
		"rollout", "status",
		"deployment/rabbitmq-cluster-operator",
		"--timeout", "10m",
	)

	if _, err := Run(cmd); err != nil {
		return err
	}

	// Verify operator pod is running
	cmd = exec.Command("kubectl", "wait",
		"--for", "condition=ready",
		"pod",
		"-l", "app.kubernetes.io/name=rabbitmq-cluster-operator",
		"-n", "rabbitmq-system",
		"--timeout", "5m",
	)

	if _, err := Run(cmd); err != nil {
		return err
	}

	return nil
}

// DumpAstarteOperatorDebuggingInfo dumps Astarte Operator info and logs for debugging purposes.
func DumpAstarteOperatorDebuggingInfo(operatorNamespace string) {
	_, _ = fmt.Fprintf(GinkgoWriter, "---- Dumping Astarte Operator info for debugging ----\n")

	// Dump Astarte Operator pods
	cmd := exec.Command("kubectl", "get", "pods", "-n", operatorNamespace)
	output, err := Run(cmd)
	if err != nil {
		warnError(fmt.Errorf("failed to dump pods: %w", err))
	}
	_, _ = fmt.Fprintf(GinkgoWriter, "\n\nAstarte Operator Pods:\n%s\n", string(output))

	// Describe Astarte Operator deployment
	cmd = exec.Command("kubectl", "describe", "deployment/astarte-kubernetes-operator-controller-manager",
		"-n", operatorNamespace)
	output, err = Run(cmd)
	if err != nil {
		warnError(fmt.Errorf("failed to describe Astarte Operator deployment: %w", err))
	}
	_, _ = fmt.Fprintf(GinkgoWriter, "\n\nAstarte Operator Deployment Description:\n%s\n", string(output))

	// Dump YAML for Astarte Operator deployment
	cmd = exec.Command("kubectl", "get", "deployment/astarte-kubernetes-operator-controller-manager",
		"-n", operatorNamespace, "-o", "yaml")
	output, err = Run(cmd)
	if err != nil {
		warnError(fmt.Errorf("failed to get Astarte Operator deployment YAML: %w", err))
	}
	_, _ = fmt.Fprintf(GinkgoWriter, "\n\nAstarte Operator Deployment YAML:\n%s\n", string(output))

	// Dump logs for Astarte Operator controller-manager
	cmd = exec.Command("kubectl", "logs", "-n", operatorNamespace,
		"deployment/astarte-kubernetes-operator-controller-manager")
	output, err = Run(cmd)
	if err != nil {
		warnError(fmt.Errorf("failed to get logs for Astarte Operator controller-manager: %w", err))
	}
	_, _ = fmt.Fprintf(GinkgoWriter, "\n\nLogs for Astarte Operator controller-manager:\n%s\n", string(output))
}

// DumpAstarteDebuggingInfo dumps Astarte info and logs for debugging purposes.
func DumpAstarteDebuggingInfo() {
	_, _ = fmt.Fprintf(GinkgoWriter, "---- Dumping Astarte info for debugging ----\n")

	// Dump Astarte pods
	cmd := exec.Command("kubectl", "get", "pods", "-n", astarteNamespace)
	output, err := Run(cmd)
	if err != nil {
		warnError(fmt.Errorf("failed to dump Astarte pods: %w", err))
	}
	_, _ = fmt.Fprintf(GinkgoWriter, "\n\nAstarte Pods:\n%s\n", string(output))

	// Dump Astarte statefulsets
	cmd = exec.Command("kubectl", "get", "statefulsets", "-n", astarteNamespace)
	output, err = Run(cmd)
	if err != nil {
		warnError(fmt.Errorf("failed to dump Astarte statefulsets: %w", err))
	}
	_, _ = fmt.Fprintf(GinkgoWriter, "\n\nAstarte StatefulSets:\n%s\n", string(output))

	// Dump Astarte services
	cmd = exec.Command("kubectl", "get", "services", "-n", astarteNamespace)
	output, err = Run(cmd)
	if err != nil {
		warnError(fmt.Errorf("failed to dump Astarte services: %w", err))
	}
	_, _ = fmt.Fprintf(GinkgoWriter, "\n\nAstarte Services:\n%s\n", string(output))

	// Dump Astarte CR
	cmd = exec.Command("kubectl", "get", "astartes.v2alpha1.api.astarte-platform.org",
		astarteName, "-n", astarteNamespace, "-o", "yaml")
	output, err = Run(cmd)
	if err != nil {
		warnError(fmt.Errorf("failed to get Astarte CR YAML: %w", err))
	}
	_, _ = fmt.Fprintf(GinkgoWriter, "\n\nAstarte CR YAML:\n%s\n", string(output))

	// Possibly we could also dump the logs for Astarte pods, describe Astarte services, etc.
	// For now, we focus on the most relevant resources, but we can expand this function as needed.
}

func UninstallRabbitMQClusterOperator() {
	url := fmt.Sprintf(rabbitmqClusterOperatorURL, rabbitmqClusterOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url, "-n", "rabbitmq-system")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

func UninstallRabbitMQCluster() {
	cmd := exec.Command("kubectl", "delete", "rabbitmqcluster", "--all", "-n", rabbitmqNamespace)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}

	// Delete the RabbitMQ namespace
	cmd = exec.Command("kubectl", "delete", "namespace", rabbitmqNamespace)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

func EnsureNamespaceExists(namespace string) error {
	if namespace == "default" || namespace == "kube-system" {
		// Skip creating default and kube-system namespaces
		return nil
	}

	cmd := exec.Command("kubectl", "create", "namespace", namespace)
	if _, err := Run(cmd); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
		}
		_, _ = fmt.Fprintf(GinkgoWriter, "Namespace %s already exists, skipping creation.\n", namespace)
	}

	// Wait for the namespace to be ready
	cmd = exec.Command("kubectl", "wait", "--for=create",
		"namespace", namespace,
		"--timeout", "30s",
	)

	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to wait for namespace %s to be ready: %w", namespace, err)
	}

	return nil
}

func DeployScyllaCluster() error {
	prj, err := GetProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %w", err)
	}

	// Create Scylla configmap
	manifestPath := fmt.Sprintf("%s/test/manifests/dependencies/scylla-config.yaml", prj)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest file %s does not exist", manifestPath)
	}

	cmd := exec.Command("kubectl", "apply", "-f", manifestPath, "-n", scyllaNamespace)
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to create Scylla configmap: %w", err)
	}

	// Create Scylla cluster
	manifestPath = fmt.Sprintf("%s/test/manifests/dependencies/scylla-cluster.yaml", prj)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest file %s does not exist", manifestPath)
	}

	cmd = exec.Command("kubectl", "apply", "-f", manifestPath, "-n", scyllaNamespace)
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to deploy Scylla cluster: %w", err)
	}

	// Wait for Scylla cluster pods to be created
	cmd = exec.Command("kubectl", "wait",
		"--for=create",
		"statefulset",
		"scylla-local-kind-1-local-kind-1a",
		"-n", scyllaNamespace,
		"--timeout", "5m",
	)

	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to wait for Scylla cluster pods to be created: %w", err)
	}

	// Wait for Scylla cluster pods to be ready
	cmd = exec.Command("kubectl", "wait",
		"--for", "condition=ready",
		"pod",
		"-l", "app=scylla",
		"-n", scyllaNamespace,
		"--timeout", "15m",
	)

	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to wait for Scylla cluster pods to be ready: %w", err)
	}

	return nil
}

func CreateScyllaConnectionSecret() error {
	secretData := map[string]string{
		"username": "cassandra",
		"password": "cassandra",
	}

	if err := CreateSecret("scylladb-connection-secret", astarteNamespace, secretData); err != nil {
		return fmt.Errorf("failed to create scylla connection secret: %w", err)
	}

	return nil
}

func InstallScyllaOperator() error {
	url := fmt.Sprintf(scyllaOperatorURL, scyllaOperatorVersion)
	cmd := exec.Command("kubectl", "create", "-f", url)
	_, err := Run(cmd)
	if err != nil {
		return err
	}

	// Create Scylla cluster namespace
	err = EnsureNamespaceExists(scyllaNamespace)
	if err != nil {
		return fmt.Errorf("failed to ensure namespace %s exists: %w", scyllaNamespace, err)
	}

	// Wait for all CRDs to be established
	crds := []string{
		"scyllaclusters.scylla.scylladb.com",
		"nodeconfigs.scylla.scylladb.com",
		"scyllaoperatorconfigs.scylla.scylladb.com",
	}

	for _, crd := range crds {
		cmd := exec.Command("kubectl", "wait",
			"--for", "condition=established",
			fmt.Sprintf("crd/%s", crd),
			"--timeout", "5m",
		)
		if _, err := Run(cmd); err != nil {
			return fmt.Errorf("failed to wait for CRD %s: %w", crd, err)
		}
	}

	// Wait for all deployments to be ready
	deployments := []string{
		"scylla-operator",
		"webhook-server",
	}

	for _, deployment := range deployments {
		cmd := exec.Command("kubectl",
			"-n", "scylla-operator",
			"rollout", "status",
			fmt.Sprintf("deployment.apps/%s", deployment),
			"--timeout", "10m",
		)
		if _, err := Run(cmd); err != nil {
			return fmt.Errorf("failed to wait for deployment %s: %w", deployment, err)
		}
	}

	return nil
}

func UninstallScyllaOperator() {
	url := fmt.Sprintf(scyllaOperatorURL, scyllaOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// CreateSecret takes a secret name, namespace, and a map of key-value pairs
// to create a generic Kubernetes secret using kubectl.
func CreateSecret(secretName string, secretNamespace string, data map[string]string) error {
	args := []string{
		"create",
		"secret",
		"generic",
		secretName,
		"--namespace",
		secretNamespace,
	}

	for key, value := range data {
		args = append(args, fmt.Sprintf("--from-literal=%s=%s", key, value))
	}

	if err := EnsureNamespaceExists(secretNamespace); err != nil {
		return fmt.Errorf("failed to ensure namespace %s exists: %w", secretNamespace, err)
	}

	cmd := exec.Command("kubectl", args...)

	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to create secret %s: %w", secretName, err)
	}

	// Wait for the secret to be created
	waitCmd := exec.Command("kubectl", "wait", "--for=create",
		"secret", secretName,
		"--namespace", secretNamespace,
		"--timeout", "30s",
	)

	if _, err := Run(waitCmd); err != nil {
		return fmt.Errorf("failed to wait for secret %s to be created: %w", secretName, err)
	}

	return nil
}
