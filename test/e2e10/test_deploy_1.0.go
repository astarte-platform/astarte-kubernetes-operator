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

package e2e10

import (
	goctx "context"
	"fmt"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	operator "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/utils"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

var target10Version string = "1.0-snapshot"

func astarteDeploy10Test(t *testing.T, f *framework.Framework, ctx *framework.Context) error {
	namespace, err := ctx.GetWatchNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}
	exampleAstarte := utils.AstarteTestResource.DeepCopy()
	exampleAstarte.ObjectMeta.Namespace = namespace
	exampleAstarte.Spec.Version = target10Version

	// use Context's create helper to create the object, and do not cleanup.
	if err := f.Client.Create(goctx.TODO(), exampleAstarte, nil); err != nil {
		return err
	}
	// wait for example-astarte-housekeeping to reach 1 replica
	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "example-astarte-housekeeping", 1, utils.DefaultRetryInterval, utils.DefaultTimeout); err != nil {
		return err
	}

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
	if err := utils.EnsureAstarteServicesReadinessUpTo10(namespace, f); err != nil {
		return err
	}

	// Print events
	if err := utils.PrintNamespaceEvents(namespace, f); err != nil {
		return err
	}

	return nil
}
