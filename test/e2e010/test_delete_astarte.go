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

package e2e010

import (
	goctx "context"
	"fmt"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operator "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func astarteDeleteTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetWatchNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}
	installedAstarte := &operator.Astarte{}
	// use TestCtx's helper to Get the object
	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: utils.AstarteTestResource.GetName(), Namespace: namespace}, installedAstarte); err != nil {
		return err
	}

	// Delete the object
	if err := f.Client.Delete(goctx.TODO(), installedAstarte); err != nil {
		return err
	}

	// Wait until everything in the namespace is erased. Finalizers should do the job.
	if err := wait.Poll(utils.DefaultRetryInterval, utils.DefaultTimeout, func() (done bool, err error) {
		deployments := &appsv1.DeploymentList{}
		if err = f.Client.List(goctx.TODO(), deployments, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if len(deployments.Items) > 0 {
			return false, nil
		}

		statefulSets := &appsv1.StatefulSetList{}
		if err = f.Client.List(goctx.TODO(), statefulSets, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if len(statefulSets.Items) > 0 {
			return false, nil
		}

		configMaps := &v1.ConfigMapList{}
		if err = f.Client.List(goctx.TODO(), configMaps, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if len(configMaps.Items) > 0 {
			return false, nil
		}

		secrets := &v1.SecretList{}
		if err = f.Client.List(goctx.TODO(), secrets, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		// The Default Token is acceptable.
		if len(secrets.Items) > 1 {
			return false, nil
		}

		pvcs := &v1.PersistentVolumeClaimList{}
		if err = f.Client.List(goctx.TODO(), pvcs, client.InNamespace(namespace)); err != nil {
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
