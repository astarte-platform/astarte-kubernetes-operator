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

package deps

import (
	"context"

	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	CustomAstarteNamespace = "astarte-controller-test"
)

var _ = Describe("Deps", Ordered, Serial, func() {
	BeforeAll(func() {
		integrationutils.CreateNamespace(k8sClient, CustomAstarteNamespace)
	})

	AfterAll(func() {
		integrationutils.TeardownResourcesInNamespace(context.Background(), k8sClient, CustomAstarteNamespace)
	})

	Context("Testing the GetDefaultVersionForCFSSL function", func() {
		// This test makes no sense as CFSSL version is hardcoded in the function.
		// We keep this here just as a reminder to update the test if we ever decide to
		// make changes to GetDefaultVersionForCFSSL.
		It("Should return the correct CFSSL version for Astarte 1.3.x", func() {
			version := GetDefaultVersionForCFSSL("foo")
			Expect(version).To(Equal("1.5.0-astarte.3"))
		})
	})
})
