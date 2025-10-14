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

	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"
	"go.openly.dev/pointy"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
)

var _ = Describe("Astarte Generic Backend testing", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "test-astarte-backend"
		CustomAstarteNamespace = "astarte-generic-backend-test"
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

	Describe("Test EnsureAstarteGenericBackend", func() {
		Context("with AppEngine API backend", func() {
			It("should create deployment and service for enabled AppEngine API", func() {
				backend := apiv2alpha1.AstarteGenericClusteredResource{
					Deploy:   pointy.Bool(true),
					Replicas: pointy.Int32(2),
				}

				component := apiv2alpha1.AppEngineAPI
				Expect(EnsureAstarteGenericBackend(cr, backend, component, k8sClient, scheme.Scheme)).To(Succeed())

				// Verify deployment was created
				deployment := &appsv1.Deployment{}
				deploymentName := cr.Name + "-" + component.DashedString()
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      deploymentName,
						Namespace: CustomAstarteNamespace,
					}, deployment)
				}, Timeout, Interval).Should(Succeed())

				Expect(*deployment.Spec.Replicas).To(Equal(int32(2)))
				Expect(deployment.Labels["astarte-component"]).To(Equal(component.DashedString()))
				Expect(deployment.Labels["app"]).To(Equal(deploymentName))
				Expect(deployment.Labels["component"]).To(Equal("astarte"))

				// Verify pod spec details
				container := deployment.Spec.Template.Spec.Containers[0]
				Expect(container.Ports).ToNot(BeEmpty())
				Expect(container.Ports[0].Name).To(Equal("http"))
				Expect(container.Ports[0].ContainerPort).To(Equal(astarteServicesPort))
				Expect(deployment.Spec.Template.Spec.ServiceAccountName).To(Equal(deploymentName))
				// Default probes on /health
				Expect(container.LivenessProbe).ToNot(BeNil())
				Expect(container.ReadinessProbe).ToNot(BeNil())
				Expect(container.LivenessProbe.HTTPGet.Path).To(Equal("/health"))
				Expect(container.ReadinessProbe.HTTPGet.Path).To(Equal("/health"))

				// Verify service was created
				service := &v1.Service{}
				serviceName := cr.Name + "-" + component.ServiceName()
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      serviceName,
						Namespace: CustomAstarteNamespace,
					}, service)
				}, Timeout, Interval).Should(Succeed())

				Expect(service.Spec.Selector["app"]).To(Equal(deploymentName))
				// Verify service spec details
				Expect(service.Spec.Type).To(Equal(v1.ServiceTypeClusterIP))
				Expect(service.Spec.Ports).To(HaveLen(1))
				Expect(service.Spec.Ports[0].Name).To(Equal("http"))
				Expect(service.Spec.Ports[0].Port).To(Equal(astarteServicesPort))
				Expect(service.Spec.Ports[0].TargetPort).To(Equal(intstr.FromString("http")))
			})

			It("should skip deployment when deploy is false", func() {
				backend := apiv2alpha1.AstarteGenericClusteredResource{
					Deploy: pointy.Bool(false),
				}

				component := apiv2alpha1.AppEngineAPI
				Expect(EnsureAstarteGenericBackend(cr, backend, component, k8sClient, scheme.Scheme)).To(Succeed())

				// Verify deployment was not created
				deployment := &appsv1.Deployment{}
				deploymentName := cr.Name + "-" + component.DashedString()
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      deploymentName,
						Namespace: CustomAstarteNamespace,
					}, deployment)
				}, "2s", Interval).ShouldNot(Succeed())
			})

			It("should delete existing deployment when deploy is changed to false", func() {
				backend := apiv2alpha1.AstarteGenericClusteredResource{
					Deploy:   pointy.Bool(true),
					Replicas: pointy.Int32(1),
				}

				component := apiv2alpha1.AppEngineAPI
				// First create the deployment
				Expect(EnsureAstarteGenericBackend(cr, backend, component, k8sClient, scheme.Scheme)).To(Succeed())

				deployment := &appsv1.Deployment{}
				deploymentName := cr.Name + "-" + component.DashedString()
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      deploymentName,
						Namespace: CustomAstarteNamespace,
					}, deployment)
				}, Timeout, Interval).Should(Succeed())

				// Now disable deployment
				backend.Deploy = pointy.Bool(false)
				Expect(EnsureAstarteGenericBackend(cr, backend, component, k8sClient, scheme.Scheme)).To(Succeed())

				// Verify deployment was deleted
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      deploymentName,
						Namespace: CustomAstarteNamespace,
					}, deployment)
				}, Timeout, Interval).ShouldNot(Succeed())
			})
		})

		Context("with TriggerEngine backend", func() {
			It("should create deployment with proper environment variables", func() {
				// Shall we test all variables?
			})
		})

		Context("with DataUpdaterPlant backend", func() {
			It("should create deployment with data queue configuration", func() {
				backend := apiv2alpha1.AstarteGenericClusteredResource{
					Deploy:   pointy.Bool(true),
					Replicas: pointy.Int32(3),
				}

				cr.Spec.Components.DataUpdaterPlant.PrefetchCount = pointy.Int(500)
				cr.Spec.RabbitMQ.DataQueuesPrefix = "custom_prefix"
				cr.Spec.VerneMQ.DeviceHeartbeatSeconds = 60

				component := apiv2alpha1.DataUpdaterPlant
				Expect(EnsureAstarteGenericBackend(cr, backend, component, k8sClient, scheme.Scheme)).To(Succeed())

				deployment := &appsv1.Deployment{}
				deploymentName := cr.Name + "-" + component.DashedString()
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      deploymentName,
						Namespace: CustomAstarteNamespace,
					}, deployment)
				}, Timeout, Interval).Should(Succeed())
			})
		})

		Context("with Dashboard backend", func() {
			It("should create deployment for Dashboard component", func() {
				backend := apiv2alpha1.AstarteGenericClusteredResource{
					Deploy:   pointy.Bool(true),
					Replicas: pointy.Int32(1),
				}

				component := apiv2alpha1.Dashboard
				Expect(EnsureAstarteGenericBackend(cr, backend, component, k8sClient, scheme.Scheme)).To(Succeed())

				deployment := &appsv1.Deployment{}
				deploymentName := cr.Name + "-" + component.DashedString()
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      deploymentName,
						Namespace: CustomAstarteNamespace,
					}, deployment)
				}, Timeout, Interval).Should(Succeed())

				Expect(deployment.Labels["astarte-component"]).To(Equal(component.DashedString()))
			})
		})

		Context("with custom probe", func() {
			It("should create deployment with custom probe", func() {
				backend := apiv2alpha1.AstarteGenericClusteredResource{
					Deploy:   pointy.Bool(true),
					Replicas: pointy.Int32(1),
				}

				customProbe := &v1.Probe{
					ProbeHandler: v1.ProbeHandler{
						HTTPGet: &v1.HTTPGetAction{
							Path: "/custom-health",
							Port: intstr.FromString("http"),
						},
					},
					InitialDelaySeconds: 30,
					TimeoutSeconds:      10,
					PeriodSeconds:       45,
					FailureThreshold:    3,
				}

				component := apiv2alpha1.AppEngineAPI
				Expect(EnsureAstarteGenericBackendWithCustomProbe(cr, backend, component, k8sClient, scheme.Scheme, customProbe)).To(Succeed())

				deployment := &appsv1.Deployment{}
				deploymentName := cr.Name + "-" + component.DashedString()
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      deploymentName,
						Namespace: CustomAstarteNamespace,
					}, deployment)
				}, Timeout, Interval).Should(Succeed())

				container := deployment.Spec.Template.Spec.Containers[0]
				Expect(container.LivenessProbe).ToNot(BeNil())
				Expect(container.ReadinessProbe).ToNot(BeNil())
				Expect(container.LivenessProbe.HTTPGet.Path).To(Equal("/custom-health"))
				Expect(container.LivenessProbe.InitialDelaySeconds).To(Equal(int32(30)))
				Expect(container.ReadinessProbe.HTTPGet.Path).To(Equal("/custom-health"))
				Expect(container.ReadinessProbe.InitialDelaySeconds).To(Equal(int32(30)))
			})
		})

		Context("with resource requirements and additional configuration", func() {
			It("should create deployment with custom resources and environment variables", func() {
				backend := apiv2alpha1.AstarteGenericClusteredResource{
					Deploy:   pointy.Bool(true),
					Replicas: pointy.Int32(1),
					Resources: &v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("100m"),
							v1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("500m"),
							v1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
					AdditionalEnv: []v1.EnvVar{
						{Name: "CUSTOM_ENV_VAR", Value: "custom_value"},
						{Name: "ANOTHER_VAR", Value: "another_value"},
					},
					PodLabels: map[string]string{
						"custom-label": "custom-value",
						"env":          "test",
					},
				}

				component := apiv2alpha1.AppEngineAPI
				Expect(EnsureAstarteGenericBackend(cr, backend, component, k8sClient, scheme.Scheme)).To(Succeed())

				deployment := &appsv1.Deployment{}
				deploymentName := cr.Name + "-" + component.DashedString()
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{
						Name:      deploymentName,
						Namespace: CustomAstarteNamespace,
					}, deployment)
				}, Timeout, Interval).Should(Succeed())

				container := deployment.Spec.Template.Spec.Containers[0]

				// Check resource requirements
				Expect(container.Resources.Requests[v1.ResourceCPU]).To(Equal(resource.MustParse("100m")))
				Expect(container.Resources.Requests[v1.ResourceMemory]).To(Equal(resource.MustParse("128Mi")))
				Expect(container.Resources.Limits[v1.ResourceCPU]).To(Equal(resource.MustParse("500m")))
				Expect(container.Resources.Limits[v1.ResourceMemory]).To(Equal(resource.MustParse("512Mi")))

				// Check pod labels
				Expect(deployment.Spec.Template.ObjectMeta.Labels["custom-label"]).To(Equal("custom-value"))
				Expect(deployment.Spec.Template.ObjectMeta.Labels["env"]).To(Equal("test"))
			})
		})
	})

	Describe("Test getAstarteGenericBackendEnvVars", func() {
		// This function is trivial, it just returns a list of env vars. No need to test it.
	})

	Describe("Test getAstarteDataUpdaterPlantQueuesEnvVars", func() {
		// This function is trivial, it just returns a list of env vars. No need to test it.
	})

	Describe("Test getAstarteBackendProbe", func() {
		It("should return custom probe when provided", func() {
			customProbe := &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/custom",
						Port: intstr.FromString("http"),
					},
				},
				InitialDelaySeconds: 20,
			}

			result := getAstarteBackendProbe(apiv2alpha1.AppEngineAPI, customProbe)
			Expect(result).To(Equal(customProbe))
		})

		It("should return housekeeping probe with longer threshold", func() {
			result := getAstarteBackendProbe(apiv2alpha1.Housekeeping, nil)

			Expect(result.HTTPGet.Path).To(Equal("/health"))
			Expect(result.FailureThreshold).To(Equal(int32(15)))
			Expect(result.InitialDelaySeconds).To(Equal(int32(10)))
		})

		It("should return generic probe for other components", func() {
			result := getAstarteBackendProbe(apiv2alpha1.AppEngineAPI, nil)

			Expect(result.HTTPGet.Path).To(Equal("/health"))
			Expect(result.FailureThreshold).To(Equal(int32(5)))
			Expect(result.InitialDelaySeconds).To(Equal(int32(10)))
		})
	})

	Describe("Test error handling", func() {
		It("should handle invalid namespace gracefully", func() {
			brokenCR := cr.DeepCopy()
			brokenCR.Namespace = "non-existing-namespace"

			backend := apiv2alpha1.AstarteGenericClusteredResource{
				Deploy:   pointy.Bool(true),
				Replicas: pointy.Int32(1),
			}

			err := EnsureAstarteGenericBackend(brokenCR, backend, apiv2alpha1.AppEngineAPI, k8sClient, scheme.Scheme)
			Expect(err).To(HaveOccurred())
		})
	})
})
