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

package e2e011

import (
	goctx "context"
	"fmt"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	operator "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/utils"
	"github.com/astarte-platform/astarte-kubernetes-operator/version"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

var target10Version string = version.SnapshotVersion

func astarteUpgradeTo10Test(f *framework.Framework, ctx *framework.Context) error {
	namespace, err := ctx.GetWatchNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}
	installedAstarte := &operator.Astarte{}
	// use Context's helper to Get the object
	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: utils.AstarteTestResource.GetName(), Namespace: namespace}, installedAstarte); err != nil {
		return err
	}

	// "Upgrade" the object
	installedAstarte.Spec.Version = target10Version
	if err := f.Client.Update(goctx.TODO(), installedAstarte); err != nil {
		return err
	}

	// Wait until Astarte reaches green state and the new version. It might take a while.
	if err := wait.Poll(utils.DefaultRetryInterval, 10*time.Minute, func() (done bool, err error) {
		astarteObj := &operator.Astarte{}
		if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Namespace: namespace, Name: utils.AstarteTestResource.GetName()}, astarteObj); err != nil {
			return false, nil
		}
		if astarteObj.Status.Health != operator.AstarteClusterHealthGreen {
			return false, nil
		}
		if astarteObj.Status.ReconciliationPhase != operator.ReconciliationPhaseReconciled {
			return false, nil
		}
		if astarteObj.Status.AstarteVersion != target10Version {
			return false, nil
		}

		return true, nil
	}); err != nil {
		return err
	}

	// Check all the Astarte Services
	if err := utils.EnsureAstarteServicesReadinessUpTo10(namespace, f); err != nil {
		return err
	}

	// Print events
	if err := utils.PrintNamespaceEvents(namespace, f); err != nil {
		return err
	}

	return nil
}
