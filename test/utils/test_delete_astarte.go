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

	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	operator "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
)

// AstarteDeleteTest deletes an Astarte instance and tests whether it was deleted and cleaned up
// nolint
func AstarteDeleteTest(c client.Client, namespace string) error {
	installedAstarte := &operator.Astarte{}
	// use Context's helper to Get the object
	if err := c.Get(context.TODO(), types.NamespacedName{Name: AstarteTestResource.GetName(), Namespace: namespace}, installedAstarte); err != nil {
		return err
	}

	// Delete the object
	if err := c.Delete(context.TODO(), installedAstarte); err != nil {
		return err
	}

	// Wait until everything in the namespace is erased. Finalizers should do the job.
	if err := wait.Poll(DefaultRetryInterval, DefaultTimeout, func() (done bool, err error) {
		deployments := &appsv1.DeploymentList{}
		if err = c.List(context.TODO(), deployments, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if len(deployments.Items) > 0 {
			return false, nil
		}

		statefulSets := &appsv1.StatefulSetList{}
		if err = c.List(context.TODO(), statefulSets, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if len(statefulSets.Items) > 0 {
			return false, nil
		}

		configMaps := &v1.ConfigMapList{}
		if err = c.List(context.TODO(), configMaps, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if l := len(configMaps.Items); l > 1 {
			return false, nil
		} else if l == 1 {
			// From Kubernetes 1.20+, a configmap named "kube-root-ca.crt" is created by default
			// in every namespace. If it's the only item left, let it be.
			if configMaps.Items[0].Name != "kube-root-ca.crt" {
				return false, nil
			}
		}

		secrets := &v1.SecretList{}
		if err = c.List(context.TODO(), secrets, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		// The Default Token is acceptable.
		if len(secrets.Items) > 1 {
			return false, nil
		}

		pvcs := &v1.PersistentVolumeClaimList{}
		if err = c.List(context.TODO(), pvcs, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if len(pvcs.Items) > 0 {
			return false, nil
		}

		return true, nil
	}); err != nil {
		return err
	}

	return nil
}
