/*
This file is part of Astarte.

Copyright 2020-25 SECO Mind Srl.

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

//nolint:goconst
package controllerutils

import (
	"context"

	"github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/builder"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/integrationutils"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("controllerutils tests", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "my-astarte"
		CustomAstarteNamespace = "astarte-controllerutils-tests"
	)

	var cr *v2alpha1.Astarte
	var b *builder.TestAstarteBuilder

	BeforeAll(func() {
		integrationutils.CreateNamespace(k8sClient, CustomAstarteNamespace)
	})

	AfterAll(func() {
		integrationutils.TeardownNamespace(k8sClient, CustomAstarteNamespace)
	})

	BeforeEach(func() {
		b = builder.NewTestAstarteBuilder(CustomAstarteName, CustomAstarteNamespace)
		cr = b.Build()
		integrationutils.DeployAstarte(k8sClient, CustomAstarteName, CustomAstarteNamespace, cr)
	})

	AfterEach(func() {
		integrationutils.TeardownResources(context.Background(), k8sClient, CustomAstarteNamespace)
	})

	Describe("TestFunction", func() {

	})
})
