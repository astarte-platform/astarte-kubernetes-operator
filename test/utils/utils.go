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

package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2" //nolint:golint,revive
)

const (
	prometheusOperatorVersion = "v0.72.0"
	prometheusOperatorURL     = "https://github.com/prometheus-operator/prometheus-operator/" +
		"releases/download/%s/bundle.yaml"

	certmanagerVersion = "v1.14.4"
	certmanagerURLTmpl = "https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager.yaml"

	// DefaultRetryInterval applied to all tests
	DefaultRetryInterval time.Duration = time.Second * 10
	// DefaultTimeout applied to all tests
	DefaultTimeout time.Duration = time.Second * 420
	// DefaultCleanupRetryInterval applied to all tests
	DefaultCleanupRetryInterval time.Duration = time.Second * 1
	// DefaultCleanupTimeout applied to all tests
	DefaultCleanupTimeout time.Duration = time.Second * 5
)

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
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

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %s\n", err)
	}

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
	wd = strings.Replace(wd, "/test/e2e", "", -1)
	return wd, nil
}

// EnsureAstarteServicesReadinessUpTo12 ensures all existing Astarte components up to 1.2
func EnsureAstarteServicesReadinessUpTo12(namespace string, c client.Client) error {
	// No changes in components deployment, just check the previous stuff
	return EnsureAstarteServicesReadinessUpTo11(namespace, c)
}

// EnsureAstarteServicesReadinessUpTo11 ensures all existing Astarte components up to 1.1
func EnsureAstarteServicesReadinessUpTo11(namespace string, c client.Client) error {
	// No changes in components deployment, just check the previous stuff
	return EnsureAstarteServicesReadinessUpTo10(namespace, c)
}

// EnsureAstarteServicesReadinessUpTo10 ensures all existing Astarte components up to 1.0
func EnsureAstarteServicesReadinessUpTo10(namespace string, c client.Client) error {
	if err := EnsureStatefulSetReadiness(namespace, "example-astarte-cassandra", c); err != nil {
		return err
	}
	if err := EnsureStatefulSetReadiness(namespace, "example-astarte-rabbitmq", c); err != nil {
		return err
	}

	// Check if API deployments + DUP are ready. If they are, we're done.
	if err := EnsureDeploymentReadiness(namespace, "example-astarte-appengine-api", c); err != nil {
		return err
	}
	if err := EnsureDeploymentReadiness(namespace, "example-astarte-housekeeping", c); err != nil {
		return err
	}
	if err := EnsureDeploymentReadiness(namespace, "example-astarte-housekeeping-api", c); err != nil {
		return err
	}
	if err := EnsureDeploymentReadiness(namespace, "example-astarte-pairing", c); err != nil {
		return err
	}
	if err := EnsureDeploymentReadiness(namespace, "example-astarte-pairing-api", c); err != nil {
		return err
	}
	if err := EnsureDeploymentReadiness(namespace, "example-astarte-realm-management", c); err != nil {
		return err
	}
	if err := EnsureDeploymentReadiness(namespace, "example-astarte-realm-management-api", c); err != nil {
		return err
	}
	if err := EnsureDeploymentReadiness(namespace, "example-astarte-trigger-engine", c); err != nil {
		return err
	}
	if err := EnsureDeploymentReadiness(namespace, "example-astarte-data-updater-plant", c); err != nil {
		return err
	}

	if err := EnsureStatefulSetReadiness(namespace, "example-astarte-vernemq", c); err != nil {
		return err
	}

	if err := EnsureDeploymentReadiness(namespace, "example-astarte-cfssl", c); err != nil {
		return err
	}

	if err := EnsureDeploymentReadiness(namespace, "example-astarte-flow", c); err != nil {
		return err
	}

	// Done
	return nil
}

// EnsureDeploymentReadiness ensures a Deployment is ready by the time the function is called
func EnsureDeploymentReadiness(namespace, name string, c client.Client) error {
	deployment := &appsv1.Deployment{}
	err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, deployment)
	if err != nil {
		return err
	}

	if deployment.Status.ReadyReplicas < 1 {
		return fmt.Errorf("not ready yet: %s in %s", name, namespace)
	}

	return nil
}

// WaitForDeploymentReadiness waits until a Deployment is ready with a reasonable timeout
func WaitForDeploymentReadiness(namespace, name string, c client.Client) error {
	return wait.Poll(DefaultRetryInterval, DefaultTimeout, func() (done bool, err error) {
		deployment := &appsv1.Deployment{}
		err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, deployment)
		if err != nil {
			return false, err
		}

		if deployment.Status.ReadyReplicas < 1 {
			return false, nil
		}

		return true, nil
	})
}

// EnsureStatefulSetReadiness ensures a StatefulSet is ready by the time the function is called
func EnsureStatefulSetReadiness(namespace, name string, c client.Client) error {
	statefulSet := &appsv1.StatefulSet{}
	err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, statefulSet)
	if err != nil {
		return err
	}

	if statefulSet.Status.ReadyReplicas < 1 {
		return fmt.Errorf("not ready yet: %s in %s", name, namespace)
	}

	return nil
}

// WaitForStatefulSetReadiness waits until a StatefulSet is ready with a reasonable timeout
func WaitForStatefulSetReadiness(namespace, name string, c client.Client) error {
	return wait.Poll(DefaultRetryInterval, DefaultTimeout, func() (done bool, err error) {
		statefulSet := &appsv1.StatefulSet{}
		err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, statefulSet)
		if err != nil {
			return false, err
		}

		if statefulSet.Status.ReadyReplicas < 1 {
			return false, nil
		}

		return true, nil
	})
}

// PrintNamespaceEvents prints to fmt all namespace events
func PrintNamespaceEvents(namespace string, c client.Client) error {
	events := &v1.EventList{}
	if err := c.List(context.TODO(), events, client.InNamespace(namespace)); err != nil {
		return err
	}

	for _, event := range events.Items {
		fmt.Printf("%s [%s]: %s: %s\n", event.InvolvedObject.Name,
			event.CreationTimestamp.String(), event.Reason, event.Message)
	}

	return nil
}
