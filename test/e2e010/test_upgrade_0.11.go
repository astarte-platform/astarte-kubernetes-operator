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
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	operator "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/utils"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

var target011Version string = "0.11.0-rc.0"

func astarteUpgradeTo011Test(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}
	installedAstarte := &operator.Astarte{}
	// use TestCtx's helper to Get the object
	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: utils.AstarteTestResource.GetName(), Namespace: namespace}, installedAstarte); err != nil {
		return err
	}

	// "Upgrade" the object
	installedAstarte.Spec.Version = target011Version
	if err := f.Client.Update(goctx.TODO(), installedAstarte); err != nil {
		return err
	}

	// Wait until Astarte reaches green state and the new version. It might take a while.
	if err := wait.Poll(retryInterval, 10*time.Minute, func() (done bool, err error) {
		astarteObj := &operator.Astarte{}
		if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Namespace: namespace, Name: utils.AstarteTestResource.GetName()}, astarteObj); err != nil {
			return false, nil
		}
		if astarteObj.Status.Health != "green" {
			return false, nil
		}
		if astarteObj.Status.ReconciliationPhase != operator.ReconciliationPhaseReconciled {
			return false, nil
		}
		if astarteObj.Status.AstarteVersion != target011Version {
			return false, nil
		}

		return true, nil
	}); err != nil {
		return err
	}

	// Check all the StatefulSets
	if err := utils.EnsureStatefulSetReadiness(namespace, "example-astarte-cfssl", f); err != nil {
		return err
	}
	if err := utils.EnsureStatefulSetReadiness(namespace, "example-astarte-cassandra", f); err != nil {
		return err
	}
	if err := utils.EnsureStatefulSetReadiness(namespace, "example-astarte-rabbitmq", f); err != nil {
		return err
	}

	// Check if API deployments + DUP are ready. If they are, we're done.
	if err := utils.EnsureDeploymentReadiness(namespace, "example-astarte-appengine-api", f); err != nil {
		return err
	}
	if err := utils.EnsureDeploymentReadiness(namespace, "example-astarte-housekeeping-api", f); err != nil {
		return err
	}
	if err := utils.EnsureDeploymentReadiness(namespace, "example-astarte-pairing-api", f); err != nil {
		return err
	}
	if err := utils.EnsureDeploymentReadiness(namespace, "example-astarte-realm-management-api", f); err != nil {
		return err
	}
	if err := utils.EnsureDeploymentReadiness(namespace, "example-astarte-trigger-engine", f); err != nil {
		return err
	}
	if err := utils.EnsureDeploymentReadiness(namespace, "example-astarte-data-updater-plant", f); err != nil {
		return err
	}

	// Check VerneMQ last thing
	if err := utils.EnsureStatefulSetReadiness(namespace, "example-astarte-vernemq", f); err != nil {
		return err
	}

	// Print events
	if err := utils.PrintNamespaceEvents(namespace, f); err != nil {
		return err
	}

	return nil
}
