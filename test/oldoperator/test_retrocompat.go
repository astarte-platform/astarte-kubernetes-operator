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

package oldoperator

import (
	goctx "context"
	"fmt"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	operator "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/utils"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func astarteEnsureRetrocompatTest(f *framework.Framework, ctx *framework.Context) error {
	namespace, err := ctx.GetWatchNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	// Wait until the Deployment turns green.
	if err := wait.Poll(utils.DefaultRetryInterval, utils.DefaultTimeout, func() (done bool, err error) {
		astarteObj := &operator.Astarte{}
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Namespace: namespace, Name: utils.AstarteTestResource.GetName()}, astarteObj)
		if err != nil {
			return false, err
		}
		if astarteObj.Status.Health != operator.AstarteClusterHealthGreen {
			return false, nil
		}
		return true, nil
	}); err != nil {
		return err
	}

	// Check all the Astarte Services
	if err := utils.EnsureAstarteServicesReadinessUpTo011(namespace, f); err != nil {
		return err
	}

	// Print events
	if err := utils.PrintNamespaceEvents(namespace, f); err != nil {
		return err
	}

	return nil
}
