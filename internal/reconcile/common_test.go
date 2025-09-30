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

package reconcile

import (
	"context"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("Common reconcile testing", Ordered, func() {
	const (
		CustomAstarteName      = "example-astarte"
		CustomAstarteNamespace = "common-test"
	)

	var cr *apiv2alpha1.Astarte

	BeforeAll(func() {
		integrationutils.CreateNamespace(k8sClient, CustomAstarteNamespace)
	})

	AfterAll(func() {
		integrationutils.DeleteNamespace(k8sClient, CustomAstarteNamespace)
	})

	BeforeEach(func() {
		cr = baseCr.DeepCopy()
		cr.SetName(CustomAstarteName)
		cr.SetNamespace(CustomAstarteNamespace)
		cr.SetResourceVersion("")
		integrationutils.DeployAstarte(k8sClient, cr)
	})

	AfterEach(func() {
		integrationutils.TeardownResourcesInNamespace(context.Background(), k8sClient, CustomAstarteNamespace)
	})

	Describe("Test EnsureHousekeepingKey", func() {
		It("Should create a valid Housekeeping keypair", func() {
			// Ensure the housekeeping key is created
			Expect(EnsureHousekeepingKey(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Check that the housekeeping keypair secrets are present
			secret_private := &v1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      CustomAstarteName + "-housekeeping-private-key",
					Namespace: CustomAstarteNamespace,
				}, secret_private)
			}, Timeout, Interval).Should(Succeed())
			Expect(secret_private.Data).To(HaveKey("private-key"))

			secret_public := &v1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      CustomAstarteName + "-housekeeping-public-key",
					Namespace: CustomAstarteNamespace,
				}, secret_public)
			}, Timeout, Interval).Should(Succeed())
			Expect(secret_public.Data).To(HaveKey("public-key"))
		})

		Describe("Test EnsureGenericErlangConfiguration", func() {
			It("should create the Generic Erlang Configuration ConfigMap", func() {
				Expect(EnsureGenericErlangConfiguration(cr, k8sClient, scheme.Scheme)).To(Succeed())

				cm := &v1.ConfigMap{}
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      CustomAstarteName + "-generic-erlang-configuration",
						Namespace: CustomAstarteNamespace,
					}, cm)
				}, Timeout, Interval).Should(Succeed())
				Expect(cm.Data["vm.args"]).To(ContainSubstring("-name ${RELEASE_NAME}@${MY_POD_IP}"))
			})
		})

		Describe("Test EnsureErlangClusteringCookie", func() {
			It("should create the Erlang Clustering Cookie secret", func() {
				Expect(EnsureErlangClusteringCookie(cr, k8sClient, scheme.Scheme)).To(Succeed())

				secret := &v1.Secret{}
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      CustomAstarteName + "-erlang-clustering-cookie",
						Namespace: CustomAstarteNamespace,
					}, secret)
				}, Timeout, Interval).Should(Succeed())
				Expect(secret.Data["erlang-cookie"]).ToNot(BeEmpty())
			})
		})

		Describe("Test GetAstarteClusteredServicePolicyRules", func() {
			It("should return the correct PolicyRules for a clustered Astarte service", func() {
				rules := GetAstarteClusteredServicePolicyRules()
				Expect(rules).ToNot(BeNil())
				Expect(rules).To(HaveLen(1))
				Expect(rules[0].APIGroups).To(Equal([]string{""}))
				Expect(rules[0].Resources).To(Equal([]string{"pods", "endpoints"}))
				Expect(rules[0].Verbs).To(Equal([]string{"list", "get"}))
			})
		})
	})
})
