/*
  This file is part of Astarte.

  Copyright 2020-23 SECO Mind Srl

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
	goctx "context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// DefaultRetryInterval applied to all tests
	DefaultRetryInterval time.Duration = time.Second * 10
	// DefaultTimeout applied to all tests
	DefaultTimeout time.Duration = time.Second * 420
	// DefaultCleanupRetryInterval applied to all tests
	DefaultCleanupRetryInterval time.Duration = time.Second * 1
	// DefaultCleanupTimeout applied to all tests
	DefaultCleanupTimeout time.Duration = time.Second * 5
)

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
	err := c.Get(goctx.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, deployment)
	if err != nil {
		return err
	}

	if deployment.Status.ReadyReplicas < 1 {
		return fmt.Errorf("Not ready yet: %s in %s", name, namespace)
	}

	return nil
}

// WaitForDeploymentReadiness waits until a Deployment is ready with a reasonable timeout
func WaitForDeploymentReadiness(namespace, name string, c client.Client) error {
	return wait.Poll(DefaultRetryInterval, DefaultTimeout, func() (done bool, err error) {
		deployment := &appsv1.Deployment{}
		err = c.Get(goctx.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, deployment)
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
	err := c.Get(goctx.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, statefulSet)
	if err != nil {
		return err
	}

	if statefulSet.Status.ReadyReplicas < 1 {
		return fmt.Errorf("Not ready yet: %s in %s", name, namespace)
	}

	return nil
}

// WaitForStatefulSetReadiness waits until a StatefulSet is ready with a reasonable timeout
func WaitForStatefulSetReadiness(namespace, name string, c client.Client) error {
	return wait.Poll(DefaultRetryInterval, DefaultTimeout, func() (done bool, err error) {
		statefulSet := &appsv1.StatefulSet{}
		err = c.Get(goctx.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, statefulSet)
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
	if err := c.List(goctx.TODO(), events, client.InNamespace(namespace)); err != nil {
		return err
	}

	for _, event := range events.Items {
		fmt.Printf("%s [%s]: %s: %s\n", event.InvolvedObject.Name, event.CreationTimestamp.String(), event.Reason, event.Message)
	}

	return nil
}
