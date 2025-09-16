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
package reconcile

import (
	"context"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	builder "github.com/astarte-platform/astarte-kubernetes-operator/test/builder"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/integrationutils"
	"go.openly.dev/pointy"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("VerneMQ testing", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "my-astarte"
		CustomAstarteNamespace = "vernemq-test"
		CustomRabbitMQHost     = "custom-rabbitmq-host"
		CustomRabbitMQPort     = 5673
		CustomVerneMQHost      = "vernemq.example.com"
		CustomVerneMQPort      = 8884
		AstarteVersion         = "1.3.0"
	)

	var cr *apiv2alpha1.Astarte
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

	Describe("Test EnsureVerneMQ", func() {
		It("should create/update the VerneMQ StatefulSet", func() {
			Expect(EnsureVerneMQ(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Check that the VerneMQ statefulSet is created
			statefulSetName := GetVerneMQStatefulSetName(cr)
			Expect(statefulSetName).To(Equal(CustomAstarteName + "-vernemq"))
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: statefulSetName, Namespace: cr.Namespace}, &appsv1.StatefulSet{})
			}, Timeout, Interval).Should(Succeed())

			// Update the VerneMQ spec to use a different image
			originalStatefulSet := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: statefulSetName, Namespace: cr.Namespace}, originalStatefulSet)).To(Succeed())
			cr.Spec.VerneMQ.Image = "vernemq/vernemq:0.12.3"
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())

			// update astarte cr.vernemq.deploy to true to force reconciliation
			cr.Spec.VerneMQ.Deploy = pointy.Bool(true)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())

			// Reconcile again
			Expect(EnsureVerneMQ(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Check that the VerneMQ statefulSet is updated
			updatedStatefulSet := &appsv1.StatefulSet{}
			Eventually(func() string {
				if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: statefulSetName, Namespace: cr.Namespace}, updatedStatefulSet); err != nil {
					return ""
				}
				if len(updatedStatefulSet.Spec.Template.Spec.Containers) == 0 {
					return ""
				}
				return updatedStatefulSet.Spec.Template.Spec.Containers[0].Image
			}, Timeout, Interval).Should(Equal("vernemq/vernemq:0.12.3"))

			// Now, set cr.spec.vernemq.deploy to false and ensure the statefulSet is deleted
			cr.Spec.VerneMQ.Deploy = pointy.Bool(false)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Expect(EnsureVerneMQ(cr, k8sClient, scheme.Scheme)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: statefulSetName, Namespace: cr.Namespace}, &appsv1.StatefulSet{})
			}, Timeout, Interval).ShouldNot(Succeed())

		})
	})
})
