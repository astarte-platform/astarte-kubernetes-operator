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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("Astarte Probes testing", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "test-astarte-probes"
		CustomAstarteNamespace = "astarte-probes-test"
	)

	var cr *apiv2alpha1.Astarte
	var emptyRes apiv2alpha1.AstarteGenericClusteredResource
	var customProbe *v1.Probe

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

		emptyRes = apiv2alpha1.AstarteGenericClusteredResource{}
		customProbe = &v1.Probe{
			ProbeHandler: v1.ProbeHandler{
				HTTPGet: &v1.HTTPGetAction{Path: "/custom", Port: intstr.FromString("http")},
			},
			InitialDelaySeconds: 3,
			TimeoutSeconds:      2,
			PeriodSeconds:       11,
			FailureThreshold:    7,
		}
	})

	AfterEach(func() {
		integrationutils.TeardownResourcesInNamespace(context.Background(), k8sClient, CustomAstarteNamespace)
	})

	// Default probes (no custom overrides)
	Describe("Test getAstarteReadinessProbe", func() {
		Context("with default probe", func() {
			It("returns the default readiness probe", func() {
				readiness := getAstarteReadinessProbe(apiv2alpha1.AppEngineAPI, emptyRes)
				Expect(readiness).ToNot(BeNil())
				Expect(readiness.HTTPGet).ToNot(BeNil())
				Expect(readiness.HTTPGet.Path).To(Equal("/health"))
				Expect(readiness.HTTPGet.Port.String()).To(Equal("http"))
				Expect(readiness.FailureThreshold).To(Equal(int32(5)))
			})
		})

		Context("with custom probes", func() {
			It("returns the custom readiness probe", func() {
				res := apiv2alpha1.AstarteGenericClusteredResource{ReadinessProbe: customProbe}
				readinessAPI := getAstarteReadinessProbe(apiv2alpha1.AppEngineAPI, res)
				Expect(readinessAPI).ToNot(BeNil())
				Expect(readinessAPI.HTTPGet).ToNot(BeNil())
				Expect(readinessAPI.HTTPGet.Path).To(Equal("/custom"))
				Expect(readinessAPI.HTTPGet.Port.String()).To(Equal("http"))
				Expect(readinessAPI.FailureThreshold).To(Equal(int32(7)))
			})
		})
	})

	Describe("Test getAstarteLivenessProbe", func() {
		Context("with default probe", func() {
			It("returns the default liveness probe", func() {
				liveness := getAstarteLivenessProbe(apiv2alpha1.AppEngineAPI, emptyRes)
				Expect(liveness).ToNot(BeNil())
				Expect(liveness.HTTPGet).ToNot(BeNil())
				Expect(liveness.HTTPGet.Path).To(Equal("/health"))
				Expect(liveness.HTTPGet.Port.String()).To(Equal("http"))
				Expect(liveness.FailureThreshold).To(Equal(int32(5)))
			})
		})

		Context("with custom probes", func() {
			It("returns the custom liveness probe", func() {
				res := apiv2alpha1.AstarteGenericClusteredResource{LivenessProbe: customProbe}
				livenessAPI := getAstarteLivenessProbe(apiv2alpha1.AppEngineAPI, res)
				Expect(livenessAPI).ToNot(BeNil())
				Expect(livenessAPI.HTTPGet).ToNot(BeNil())
				Expect(livenessAPI.HTTPGet.Path).To(Equal("/custom"))
				Expect(livenessAPI.HTTPGet.Port.String()).To(Equal("http"))
				Expect(livenessAPI.FailureThreshold).To(Equal(int32(7)))
			})
		})
	})

	Describe("Test getAstarteStartupProbe", func() {
		Context("with default probe", func() {
			It("returns nil when no startup probe is configured", func() {
				startup := getAstarteStartupProbe(emptyRes)
				Expect(startup).To(BeNil())
			})
		})

		Context("with custom probes", func() {
			It("returns the custom startup probe", func() {
				res := apiv2alpha1.AstarteGenericClusteredResource{StartupProbe: customProbe}
				startupAPI := getAstarteStartupProbe(res)
				Expect(startupAPI).ToNot(BeNil())
				Expect(startupAPI.HTTPGet).ToNot(BeNil())
				Expect(startupAPI.HTTPGet.Path).To(Equal("/custom"))
				Expect(startupAPI.HTTPGet.Port.String()).To(Equal("http"))
				Expect(startupAPI.FailureThreshold).To(Equal(int32(7)))
			})
		})
	})

	// Test that probes (default and custom) are correctly applied on deployed resources
	Describe("Test probes injection on reconcile", func() {
		DescribeTable("should inject default probes when no custom probes are set",
			func(component apiv2alpha1.AstarteComponent) {
				Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
				}, Timeout, Interval).Should(Succeed())

				switch component {
				case apiv2alpha1.Housekeeping:
					Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.Housekeeping, component, k8sClient, scheme.Scheme)).To(Succeed())
				case apiv2alpha1.RealmManagement:
					Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.RealmManagement, component, k8sClient, scheme.Scheme)).To(Succeed())
				case apiv2alpha1.Pairing:
					Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.Pairing, component, k8sClient, scheme.Scheme)).To(Succeed())
				case apiv2alpha1.AppEngineAPI:
					Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec, component, k8sClient, scheme.Scheme)).To(Succeed())
				case apiv2alpha1.DataUpdaterPlant:
					Expect(EnsureAstarteGenericBackend(cr, cr.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource, component, k8sClient, scheme.Scheme)).To(Succeed())
				case apiv2alpha1.TriggerEngine:
					Expect(EnsureAstarteGenericBackend(cr, cr.Spec.Components.TriggerEngine.AstarteGenericClusteredResource, component, k8sClient, scheme.Scheme)).To(Succeed())
				}

				// Verify deployment was created and get it
				deploymentName := cr.Name + "-" + component.DashedString()
				dep := &appsv1.Deployment{}
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      deploymentName,
						Namespace: CustomAstarteNamespace,
					}, dep)
				}, Timeout, Interval).Should(Succeed())

				Expect(dep.Spec.Template.Spec.Containers).To(HaveLen(1))
				container := dep.Spec.Template.Spec.Containers[0]

				// Verify readiness and liveness probes exist with default values
				Expect(container.ReadinessProbe).ToNot(BeNil())
				Expect(container.LivenessProbe).ToNot(BeNil())
				Expect(container.ReadinessProbe.HTTPGet).ToNot(BeNil())
				Expect(container.LivenessProbe.HTTPGet).ToNot(BeNil())
				Expect(container.ReadinessProbe.HTTPGet.Path).To(Equal("/health"))
				Expect(container.LivenessProbe.HTTPGet.Path).To(Equal("/health"))
				Expect(container.ReadinessProbe.HTTPGet.Port.String()).To(Equal("http"))
				Expect(container.LivenessProbe.HTTPGet.Port.String()).To(Equal("http"))

				// Verify startup probe is nil by default
				Expect(container.StartupProbe).To(BeNil())

				// Housekeeping has longer failure thresholds
				if component == apiv2alpha1.Housekeeping {
					Expect(container.ReadinessProbe.FailureThreshold).To(Equal(int32(15)))
					Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(15)))
				} else {
					Expect(container.ReadinessProbe.FailureThreshold).To(Equal(int32(5)))
					Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(5)))
				}
			},
			Entry("Housekeeping", apiv2alpha1.Housekeeping),
			Entry("RealmManagement", apiv2alpha1.RealmManagement),
			Entry("Pairing", apiv2alpha1.Pairing),
			Entry("DataUpdaterPlant", apiv2alpha1.DataUpdaterPlant),
			Entry("AppEngineAPI", apiv2alpha1.AppEngineAPI),
			Entry("TriggerEngine", apiv2alpha1.TriggerEngine),
		)

		DescribeTable("should inject custom probes on API components",
			func(component apiv2alpha1.AstarteComponent) {
				switch component {
				case apiv2alpha1.Housekeeping:
					cr.Spec.Components.Housekeeping.LivenessProbe = customProbe
					cr.Spec.Components.Housekeeping.ReadinessProbe = customProbe
					cr.Spec.Components.Housekeeping.StartupProbe = customProbe
				case apiv2alpha1.RealmManagement:
					cr.Spec.Components.RealmManagement.LivenessProbe = customProbe
					cr.Spec.Components.RealmManagement.ReadinessProbe = customProbe
					cr.Spec.Components.RealmManagement.StartupProbe = customProbe
				case apiv2alpha1.Pairing:
					cr.Spec.Components.Pairing.LivenessProbe = customProbe
					cr.Spec.Components.Pairing.ReadinessProbe = customProbe
					cr.Spec.Components.Pairing.StartupProbe = customProbe
				case apiv2alpha1.AppEngineAPI:
					cr.Spec.Components.AppengineAPI.LivenessProbe = customProbe
					cr.Spec.Components.AppengineAPI.ReadinessProbe = customProbe
					cr.Spec.Components.AppengineAPI.StartupProbe = customProbe
				case apiv2alpha1.DataUpdaterPlant:
					cr.Spec.Components.DataUpdaterPlant.LivenessProbe = customProbe
					cr.Spec.Components.DataUpdaterPlant.ReadinessProbe = customProbe
					cr.Spec.Components.DataUpdaterPlant.StartupProbe = customProbe
				case apiv2alpha1.TriggerEngine:
					cr.Spec.Components.TriggerEngine.LivenessProbe = customProbe
					cr.Spec.Components.TriggerEngine.ReadinessProbe = customProbe
					cr.Spec.Components.TriggerEngine.StartupProbe = customProbe
				}

				Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
				}, Timeout, Interval).Should(Succeed())

				switch component {
				case apiv2alpha1.Housekeeping:
					Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.Housekeeping, component, k8sClient, scheme.Scheme)).To(Succeed())
				case apiv2alpha1.RealmManagement:
					Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.RealmManagement, component, k8sClient, scheme.Scheme)).To(Succeed())
				case apiv2alpha1.Pairing:
					Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.Pairing, component, k8sClient, scheme.Scheme)).To(Succeed())
				case apiv2alpha1.AppEngineAPI:
					Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec, component, k8sClient, scheme.Scheme)).To(Succeed())
				case apiv2alpha1.DataUpdaterPlant:
					Expect(EnsureAstarteGenericBackend(cr, cr.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource, component, k8sClient, scheme.Scheme)).To(Succeed())
				case apiv2alpha1.TriggerEngine:
					Expect(EnsureAstarteGenericBackend(cr, cr.Spec.Components.TriggerEngine.AstarteGenericClusteredResource, component, k8sClient, scheme.Scheme)).To(Succeed())
				}
				// Verify deployment was created and get it
				deploymentName := cr.Name + "-" + component.DashedString()
				dep := &appsv1.Deployment{}
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      deploymentName,
						Namespace: CustomAstarteNamespace,
					}, dep)
				}, Timeout, Interval).Should(Succeed())

				Expect(dep.Spec.Template.Spec.Containers).To(HaveLen(1))
				container := dep.Spec.Template.Spec.Containers[0]

				// Verify all custom probes are applied
				Expect(container.ReadinessProbe).ToNot(BeNil())
				Expect(container.LivenessProbe).ToNot(BeNil())
				Expect(container.StartupProbe).ToNot(BeNil())

				// Verify custom readiness probe
				Expect(container.ReadinessProbe.HTTPGet).ToNot(BeNil())
				Expect(container.ReadinessProbe.HTTPGet.Path).To(Equal("/custom"))
				Expect(container.ReadinessProbe.HTTPGet.Port.String()).To(Equal("http"))
				Expect(container.ReadinessProbe.InitialDelaySeconds).To(Equal(int32(3)))
				Expect(container.ReadinessProbe.TimeoutSeconds).To(Equal(int32(2)))
				Expect(container.ReadinessProbe.PeriodSeconds).To(Equal(int32(11)))
				Expect(container.ReadinessProbe.FailureThreshold).To(Equal(int32(7)))

				// Verify custom liveness probe
				Expect(container.LivenessProbe.HTTPGet).ToNot(BeNil())
				Expect(container.LivenessProbe.HTTPGet.Path).To(Equal("/custom"))
				Expect(container.LivenessProbe.HTTPGet.Port.String()).To(Equal("http"))
				Expect(container.LivenessProbe.InitialDelaySeconds).To(Equal(int32(3)))
				Expect(container.LivenessProbe.TimeoutSeconds).To(Equal(int32(2)))
				Expect(container.LivenessProbe.PeriodSeconds).To(Equal(int32(11)))
				Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(7)))

				// Verify custom startup probe
				Expect(container.StartupProbe.HTTPGet).ToNot(BeNil())
				Expect(container.StartupProbe.HTTPGet.Path).To(Equal("/custom"))
				Expect(container.StartupProbe.HTTPGet.Port.String()).To(Equal("http"))
				Expect(container.StartupProbe.InitialDelaySeconds).To(Equal(int32(3)))
				Expect(container.StartupProbe.TimeoutSeconds).To(Equal(int32(2)))
				Expect(container.StartupProbe.PeriodSeconds).To(Equal(int32(11)))
				Expect(container.StartupProbe.FailureThreshold).To(Equal(int32(7)))
			},
			Entry("Housekeeping", apiv2alpha1.Housekeeping),
			Entry("RealmManagement", apiv2alpha1.RealmManagement),
			Entry("Pairing", apiv2alpha1.Pairing),
			Entry("DataUpdaterPlant", apiv2alpha1.DataUpdaterPlant),
			Entry("AppEngineAPI", apiv2alpha1.AppEngineAPI),
			Entry("TriggerEngine", apiv2alpha1.TriggerEngine),
		)
	})
})
