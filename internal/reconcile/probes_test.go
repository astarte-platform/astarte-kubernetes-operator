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
	Describe("Test getAstarteComponentReadinessProbe", func() {
		Context("with default probe", func() {
			It("returns the default readiness probe", func() {
				readiness := getAstarteComponentReadinessProbe(apiv2alpha1.AppEngineAPI, emptyRes)
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
				readinessAPI := getAstarteComponentReadinessProbe(apiv2alpha1.AppEngineAPI, res)
				Expect(readinessAPI).ToNot(BeNil())
				Expect(readinessAPI.HTTPGet).ToNot(BeNil())
				Expect(readinessAPI.HTTPGet.Path).To(Equal("/custom"))
				Expect(readinessAPI.HTTPGet.Port.String()).To(Equal("http"))
				Expect(readinessAPI.FailureThreshold).To(Equal(int32(7)))
			})
		})
	})

	Describe("Test getAstarteComponentLivenessProbe", func() {
		Context("with default probe", func() {
			It("returns the default liveness probe", func() {
				liveness := getAstarteComponentLivenessProbe(apiv2alpha1.AppEngineAPI, emptyRes)
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
				livenessAPI := getAstarteComponentLivenessProbe(apiv2alpha1.AppEngineAPI, res)
				Expect(livenessAPI).ToNot(BeNil())
				Expect(livenessAPI.HTTPGet).ToNot(BeNil())
				Expect(livenessAPI.HTTPGet.Path).To(Equal("/custom"))
				Expect(livenessAPI.HTTPGet.Port.String()).To(Equal("http"))
				Expect(livenessAPI.FailureThreshold).To(Equal(int32(7)))
			})
		})
	})

	Describe("Test getAstarteComponentStartupProbe", func() {
		Context("with default probe", func() {
			It("returns nil when no startup probe is configured", func() {
				startup := getAstarteComponentStartupProbe(emptyRes)
				Expect(startup).To(BeNil())
			})
		})

		Context("with custom probes", func() {
			It("returns the custom startup probe", func() {
				res := apiv2alpha1.AstarteGenericClusteredResource{StartupProbe: customProbe}
				startupAPI := getAstarteComponentStartupProbe(res)
				Expect(startupAPI).ToNot(BeNil())
				Expect(startupAPI.HTTPGet).ToNot(BeNil())
				Expect(startupAPI.HTTPGet.Path).To(Equal("/custom"))
				Expect(startupAPI.HTTPGet.Port.String()).To(Equal("http"))
				Expect(startupAPI.FailureThreshold).To(Equal(int32(7)))
			})
		})
	})

	// Test that probes (default and custom) are correctly applied on deployed resources
	Describe("Test probes injection on reconcile (Astarte Components)", func() {
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

	// Test CFSSL probes
	Describe("Test CFSSL Probes", func() {
		Describe("Test getCFSSLReadinessProbe", func() {
			Context("with default probe", func() {
				It("returns the default readiness probe", func() {
					readiness := getCFSSLReadinessProbe(cr)
					Expect(readiness).ToNot(BeNil())
					Expect(readiness.HTTPGet).ToNot(BeNil())
					Expect(readiness.HTTPGet.Path).To(Equal("/api/v1/cfssl/health"))
					Expect(readiness.HTTPGet.Port.String()).To(Equal("http"))
					Expect(readiness.InitialDelaySeconds).To(Equal(int32(10)))
					Expect(readiness.TimeoutSeconds).To(Equal(int32(5)))
					Expect(readiness.PeriodSeconds).To(Equal(int32(20)))
					Expect(readiness.FailureThreshold).To(Equal(int32(3)))
				})
			})

			Context("with custom probes", func() {
				It("returns the custom readiness probe", func() {
					cr.Spec.CFSSL.ReadinessProbe = customProbe
					readinessCFSSL := getCFSSLReadinessProbe(cr)
					Expect(readinessCFSSL).ToNot(BeNil())
					Expect(readinessCFSSL.HTTPGet).ToNot(BeNil())
					Expect(readinessCFSSL.HTTPGet.Path).To(Equal("/custom"))
					Expect(readinessCFSSL.HTTPGet.Port.String()).To(Equal("http"))
					Expect(readinessCFSSL.InitialDelaySeconds).To(Equal(int32(3)))
					Expect(readinessCFSSL.TimeoutSeconds).To(Equal(int32(2)))
					Expect(readinessCFSSL.PeriodSeconds).To(Equal(int32(11)))
					Expect(readinessCFSSL.FailureThreshold).To(Equal(int32(7)))
				})
			})
		})

		Describe("Test getCFSSLLivenessProbe", func() {
			Context("with default probe", func() {
				It("returns the default liveness probe", func() {
					liveness := getCFSSLLivenessProbe(cr)
					Expect(liveness).ToNot(BeNil())
					Expect(liveness.HTTPGet).ToNot(BeNil())
					Expect(liveness.HTTPGet.Path).To(Equal("/api/v1/cfssl/health"))
					Expect(liveness.HTTPGet.Port.String()).To(Equal("http"))
					Expect(liveness.InitialDelaySeconds).To(Equal(int32(10)))
					Expect(liveness.TimeoutSeconds).To(Equal(int32(5)))
					Expect(liveness.PeriodSeconds).To(Equal(int32(20)))
					Expect(liveness.FailureThreshold).To(Equal(int32(3)))
				})
			})

			Context("with custom probes", func() {
				It("returns the custom liveness probe", func() {
					cr.Spec.CFSSL.LivenessProbe = customProbe
					livenessCFSSL := getCFSSLLivenessProbe(cr)
					Expect(livenessCFSSL).ToNot(BeNil())
					Expect(livenessCFSSL.HTTPGet).ToNot(BeNil())
					Expect(livenessCFSSL.HTTPGet.Path).To(Equal("/custom"))
					Expect(livenessCFSSL.HTTPGet.Port.String()).To(Equal("http"))
					Expect(livenessCFSSL.InitialDelaySeconds).To(Equal(int32(3)))
					Expect(livenessCFSSL.TimeoutSeconds).To(Equal(int32(2)))
					Expect(livenessCFSSL.PeriodSeconds).To(Equal(int32(11)))
					Expect(livenessCFSSL.FailureThreshold).To(Equal(int32(7)))
				})
			})
		})

		Describe("Test getCFSSLStartupProbe", func() {
			Context("with default probe", func() {
				It("returns nil when no startup probe is configured", func() {
					startup := getCFSSLStartupProbe(cr)
					Expect(startup).To(BeNil())
				})
			})

			Context("with custom probes", func() {
				It("returns the custom startup probe", func() {
					cr.Spec.CFSSL.StartupProbe = customProbe
					startupCFSSL := getCFSSLStartupProbe(cr)
					Expect(startupCFSSL).ToNot(BeNil())
					Expect(startupCFSSL.HTTPGet).ToNot(BeNil())
					Expect(startupCFSSL.HTTPGet.Path).To(Equal("/custom"))
					Expect(startupCFSSL.HTTPGet.Port.String()).To(Equal("http"))
					Expect(startupCFSSL.InitialDelaySeconds).To(Equal(int32(3)))
					Expect(startupCFSSL.TimeoutSeconds).To(Equal(int32(2)))
					Expect(startupCFSSL.PeriodSeconds).To(Equal(int32(11)))
					Expect(startupCFSSL.FailureThreshold).To(Equal(int32(7)))
				})
			})
		})
	})

	// Test VerneMQ probes
	Describe("Test VerneMQ Probes", func() {
		Describe("Test getVerneMQReadinessProbe", func() {
			Context("with default probe", func() {
				It("returns the default readiness probe", func() {
					readiness := getVerneMQReadinessProbe(cr)
					Expect(readiness).ToNot(BeNil())
					Expect(readiness.HTTPGet).ToNot(BeNil())
					Expect(readiness.HTTPGet.Path).To(Equal("/metrics"))
					Expect(readiness.HTTPGet.Port.IntVal).To(Equal(int32(8888)))
					Expect(readiness.InitialDelaySeconds).To(Equal(int32(60)))
					Expect(readiness.TimeoutSeconds).To(Equal(int32(10)))
					Expect(readiness.PeriodSeconds).To(Equal(int32(20)))
					Expect(readiness.FailureThreshold).To(Equal(int32(3)))
				})
			})

			Context("with custom probes", func() {
				It("returns the custom readiness probe", func() {
					cr.Spec.VerneMQ.ReadinessProbe = customProbe
					readinessVerneMQ := getVerneMQReadinessProbe(cr)
					Expect(readinessVerneMQ).ToNot(BeNil())
					Expect(readinessVerneMQ.HTTPGet).ToNot(BeNil())
					Expect(readinessVerneMQ.HTTPGet.Path).To(Equal("/custom"))
					Expect(readinessVerneMQ.HTTPGet.Port.String()).To(Equal("http"))
					Expect(readinessVerneMQ.InitialDelaySeconds).To(Equal(int32(3)))
					Expect(readinessVerneMQ.TimeoutSeconds).To(Equal(int32(2)))
					Expect(readinessVerneMQ.PeriodSeconds).To(Equal(int32(11)))
					Expect(readinessVerneMQ.FailureThreshold).To(Equal(int32(7)))
				})
			})
		})

		Describe("Test getVerneMQLivenessProbe", func() {
			Context("with default probe", func() {
				It("returns the default liveness probe", func() {
					liveness := getVerneMQLivenessProbe(cr)
					Expect(liveness).ToNot(BeNil())
					Expect(liveness.HTTPGet).ToNot(BeNil())
					Expect(liveness.HTTPGet.Path).To(Equal("/metrics"))
					Expect(liveness.HTTPGet.Port.IntVal).To(Equal(int32(8888)))
					Expect(liveness.InitialDelaySeconds).To(Equal(int32(60)))
					Expect(liveness.TimeoutSeconds).To(Equal(int32(10)))
					Expect(liveness.PeriodSeconds).To(Equal(int32(20)))
					Expect(liveness.FailureThreshold).To(Equal(int32(3)))
				})
			})

			Context("with custom probes", func() {
				It("returns the custom liveness probe", func() {
					cr.Spec.VerneMQ.LivenessProbe = customProbe
					livenessVerneMQ := getVerneMQLivenessProbe(cr)
					Expect(livenessVerneMQ).ToNot(BeNil())
					Expect(livenessVerneMQ.HTTPGet).ToNot(BeNil())
					Expect(livenessVerneMQ.HTTPGet.Path).To(Equal("/custom"))
					Expect(livenessVerneMQ.HTTPGet.Port.String()).To(Equal("http"))
					Expect(livenessVerneMQ.InitialDelaySeconds).To(Equal(int32(3)))
					Expect(livenessVerneMQ.TimeoutSeconds).To(Equal(int32(2)))
					Expect(livenessVerneMQ.PeriodSeconds).To(Equal(int32(11)))
					Expect(livenessVerneMQ.FailureThreshold).To(Equal(int32(7)))
				})
			})
		})

		Describe("Test getVerneMQStartupProbe", func() {
			Context("with default probe", func() {
				It("returns nil when no startup probe is configured", func() {
					startup := getVerneMQStartupProbe(cr)
					Expect(startup).To(BeNil())
				})
			})

			Context("with custom probes", func() {
				It("returns the custom startup probe", func() {
					cr.Spec.VerneMQ.StartupProbe = customProbe
					startupVerneMQ := getVerneMQStartupProbe(cr)
					Expect(startupVerneMQ).ToNot(BeNil())
					Expect(startupVerneMQ.HTTPGet).ToNot(BeNil())
					Expect(startupVerneMQ.HTTPGet.Path).To(Equal("/custom"))
					Expect(startupVerneMQ.HTTPGet.Port.String()).To(Equal("http"))
					Expect(startupVerneMQ.InitialDelaySeconds).To(Equal(int32(3)))
					Expect(startupVerneMQ.TimeoutSeconds).To(Equal(int32(2)))
					Expect(startupVerneMQ.PeriodSeconds).To(Equal(int32(11)))
					Expect(startupVerneMQ.FailureThreshold).To(Equal(int32(7)))
				})
			})
		})
	})

	// Test probes injection on reconcile for CFSSL and VerneMQ
	Describe("Test probes injection on reconcile (CFSSL)", func() {
		It("should inject default probes when no custom probes are set", func() {
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, Timeout, Interval).Should(Succeed())

			Expect(EnsureCFSSL(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Verify deployment was created and get it
			deploymentName := cr.Name + "-cfssl"
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
			Expect(container.ReadinessProbe.HTTPGet.Path).To(Equal("/api/v1/cfssl/health"))
			Expect(container.LivenessProbe.HTTPGet.Path).To(Equal("/api/v1/cfssl/health"))
			Expect(container.ReadinessProbe.HTTPGet.Port.String()).To(Equal("http"))
			Expect(container.LivenessProbe.HTTPGet.Port.String()).To(Equal("http"))
			Expect(container.ReadinessProbe.InitialDelaySeconds).To(Equal(int32(10)))
			Expect(container.LivenessProbe.InitialDelaySeconds).To(Equal(int32(10)))
			Expect(container.ReadinessProbe.FailureThreshold).To(Equal(int32(3)))
			Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(3)))

			// Verify startup probe is nil by default
			Expect(container.StartupProbe).To(BeNil())
		})

		It("should inject custom probes on CFSSL", func() {
			cr.Spec.CFSSL.LivenessProbe = customProbe
			cr.Spec.CFSSL.ReadinessProbe = customProbe
			cr.Spec.CFSSL.StartupProbe = customProbe

			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, Timeout, Interval).Should(Succeed())

			Expect(EnsureCFSSL(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Verify deployment was created and get it
			deploymentName := cr.Name + "-cfssl"
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
		})
	})

	Describe("Test probes injection on reconcile (VerneMQ)", func() {
		It("should inject default probes when no custom probes are set", func() {
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, Timeout, Interval).Should(Succeed())

			Expect(EnsureVerneMQ(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Verify statefulset was created and get it
			statefulSetName := cr.Name + "-vernemq"
			sts := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      statefulSetName,
					Namespace: CustomAstarteNamespace,
				}, sts)
			}, Timeout, Interval).Should(Succeed())

			Expect(sts.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := sts.Spec.Template.Spec.Containers[0]

			// Verify readiness and liveness probes exist with default values
			Expect(container.ReadinessProbe).ToNot(BeNil())
			Expect(container.LivenessProbe).ToNot(BeNil())
			Expect(container.ReadinessProbe.HTTPGet).ToNot(BeNil())
			Expect(container.LivenessProbe.HTTPGet).ToNot(BeNil())
			Expect(container.ReadinessProbe.HTTPGet.Path).To(Equal("/metrics"))
			Expect(container.LivenessProbe.HTTPGet.Path).To(Equal("/metrics"))
			Expect(container.ReadinessProbe.HTTPGet.Port.IntVal).To(Equal(int32(8888)))
			Expect(container.LivenessProbe.HTTPGet.Port.IntVal).To(Equal(int32(8888)))
			Expect(container.ReadinessProbe.InitialDelaySeconds).To(Equal(int32(60)))
			Expect(container.LivenessProbe.InitialDelaySeconds).To(Equal(int32(60)))
			Expect(container.ReadinessProbe.FailureThreshold).To(Equal(int32(3)))
			Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(3)))

			// Verify startup probe is nil by default
			Expect(container.StartupProbe).To(BeNil())
		})

		It("should inject custom probes on VerneMQ", func() {
			cr.Spec.VerneMQ.LivenessProbe = customProbe
			cr.Spec.VerneMQ.ReadinessProbe = customProbe
			cr.Spec.VerneMQ.StartupProbe = customProbe

			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, Timeout, Interval).Should(Succeed())

			Expect(EnsureVerneMQ(cr, k8sClient, scheme.Scheme)).To(Succeed())

			// Verify statefulset was created and get it
			statefulSetName := cr.Name + "-vernemq"
			sts := &appsv1.StatefulSet{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      statefulSetName,
					Namespace: CustomAstarteNamespace,
				}, sts)
			}, Timeout, Interval).Should(Succeed())

			Expect(sts.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := sts.Spec.Template.Spec.Containers[0]

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
		})
	})
})
