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
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	operator "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/utils"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

var target011Version string = "0.11.2"

func astarteDeploy011Test(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}
	exampleAstarte := utils.AstarteTestResource.DeepCopy()
	exampleAstarte.ObjectMeta.Namespace = namespace
	exampleAstarte.Spec.Version = target011Version

	// use TestCtx's create helper to create the object, and do not cleanup.
	if err := f.Client.Create(goctx.TODO(), exampleAstarte, nil); err != nil {
		return err
	}
	// wait for example-astarte-housekeeping-api to reach 1 replica
	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "example-astarte-housekeeping-api", 1, retryInterval, timeout); err != nil {
		return err
	}

	if err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		astarteObj := &operator.Astarte{}
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Namespace: namespace, Name: utils.AstarteTestResource.GetName()}, astarteObj)
		if err != nil {
			return false, err
		}
		if astarteObj.Status.Health != "green" {
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
