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
	"testing"

	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis"
	operator "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/utils"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
)

func TestAstarte(t *testing.T) {
	astarteList := &operator.AstarteList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, astarteList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("astarte-group", func(t *testing.T) {
		t.Run("Cluster", AstarteCluster)
	})
}

func AstarteCluster(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{
		TestContext:   ctx,
		Timeout:       utils.DefaultCleanupTimeout,
		RetryInterval: utils.DefaultCleanupRetryInterval,
	})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetOperatorNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	// wait for astarte-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "astarte-operator", 1, utils.DefaultRetryInterval, utils.DefaultTimeout)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Starting Deployment Test")
	if err = astarteDeploy011Test(t, f, ctx); err != nil {
		t.Fatal(err)
	}

	t.Log("Starting Deletion Test")
	if err = astarteDeleteTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}
