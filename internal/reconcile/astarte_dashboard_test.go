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

//nolint:goconst,dupl
package reconcile

import (
	"context"
	"encoding/json"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Astarte Dashboard reconcile tests", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "my-astarte-dashboard"
		CustomAstarteNamespace = "astarte-dashboard-test"
		AstarteVersion         = "1.3.0"
		CustomRabbitMQHost     = "rabbitmq.example.com"
		CustomRabbitMQPort     = 5672
		CustomVerneMQHost      = "vernemq.example.com"
		CustomVerneMQPort      = 8883
	)

	var cr *apiv2alpha1.Astarte

	BeforeAll(func() {
		if CustomAstarteNamespace != "default" {
			ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}}
			Eventually(func() error {
				err := k8sClient.Create(context.Background(), ns)
				if apierrors.IsAlreadyExists(err) {
					return nil
				}
				return err
			}, "10s", "250ms").Should(Succeed())
		}
	})

	AfterAll(func() {
		if CustomAstarteNamespace != "default" {
			astartes := &apiv2alpha1.AstarteList{}
			Expect(k8sClient.List(context.Background(), astartes, client.InNamespace(CustomAstarteNamespace))).To(Succeed())
			for _, a := range astartes.Items {
				_ = k8sClient.Delete(context.Background(), &a)
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &apiv2alpha1.Astarte{})
				}, "10s", "250ms").ShouldNot(Succeed())
			}
			_ = k8sClient.Delete(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}})
		}
	})

	BeforeEach(func() {
		cr = &apiv2alpha1.Astarte{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CustomAstarteName,
				Namespace: CustomAstarteNamespace,
			},
			Spec: apiv2alpha1.AstarteSpec{
				Version: AstarteVersion,
				RabbitMQ: apiv2alpha1.AstarteRabbitMQSpec{
					Connection: &apiv2alpha1.AstarteRabbitMQConnectionSpec{
						HostAndPort: apiv2alpha1.HostAndPort{Host: CustomRabbitMQHost, Port: pointy.Int32(CustomRabbitMQPort)},
					},
				},
				VerneMQ: apiv2alpha1.AstarteVerneMQSpec{
					HostAndPort: apiv2alpha1.HostAndPort{Host: CustomVerneMQHost, Port: pointy.Int32(CustomVerneMQPort)},
				},
				Cassandra: apiv2alpha1.AstarteCassandraSpec{
					Connection: &apiv2alpha1.AstarteCassandraConnectionSpec{Nodes: []apiv2alpha1.HostAndPort{{Host: "cassandra.example.com", Port: pointy.Int32(9042)}}},
				},
				Components: apiv2alpha1.AstarteComponentsSpec{
					// Dashboard spec gets filled in each test as needed.
					Dashboard: apiv2alpha1.AstarteDashboardSpec{},
				},
			},
		}

		Expect(k8sClient.Create(context.Background(), cr)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
		}, "10s", "250ms").Should(Succeed())
	})

	AfterEach(func() {
		astartes := &apiv2alpha1.AstarteList{}
		Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, a := range astartes.Items {
			Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &apiv2alpha1.Astarte{})
			}, "10s", "250ms").ShouldNot(Succeed())
		}

		// Clean up any deployments left behind (defensive)
		deployments := &appsv1.DeploymentList{}
		Expect(k8sClient.List(context.Background(), deployments, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, d := range deployments.Items {
			_ = k8sClient.Delete(context.Background(), &d)
		}

		Eventually(func() int {
			list := &apiv2alpha1.AstarteList{}
			if err := k8sClient.List(context.Background(), list, &client.ListOptions{Namespace: CustomAstarteNamespace}); err != nil {
				return -1
			}
			return len(list.Items)
		}, "10s", "250ms").Should(Equal(0))
	})

	Describe("Test EnsureAstarteDashboard", func() {
		It("should not create a deployment when disabled", func() {
			cr.Spec.Components.Dashboard.Deploy = pointy.Bool(false)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteDashboard(cr, cr.Spec.Components.Dashboard, k8sClient, scheme.Scheme)).To(Succeed())

			// Deployment should not exist
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name + "-dashboard", Namespace: cr.Namespace}, dep)).ToNot(Succeed())
		})

		It("should create deployment, service and configmap with defaults", func() {
			cr.Spec.Components.Dashboard.Deploy = pointy.Bool(true)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteDashboard(cr, cr.Spec.Components.Dashboard, k8sClient, scheme.Scheme)).To(Succeed())

			// Deployment
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name + "-dashboard", Namespace: cr.Namespace}, dep)).To(Succeed())
			Expect(dep.Labels).To(Not(BeNil()))
			Expect(dep.Labels["app"]).To(Equal(cr.Name + "-dashboard"))
			Expect(dep.Labels["component"]).To(Equal("astarte"))
			Expect(dep.Labels["astarte-component"]).To(Equal("dashboard"))

			// Service
			svc := &v1.Service{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name + "-dashboard", Namespace: cr.Namespace}, svc)).To(Succeed())
			Expect(svc.Spec.Ports).To(HaveLen(1))
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(80)))

			// ConfigMap
			cm := &v1.ConfigMap{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name + "-dashboard-config", Namespace: cr.Namespace}, cm)).To(Succeed())
			cfgStr, ok := cm.Data["config.json"]
			Expect(ok).To(BeTrue())

			var cfg map[string]interface{}
			Expect(json.Unmarshal([]byte(cfgStr), &cfg)).To(Succeed())
			Expect(cfg["astarte_api_url"]).ToNot(BeNil())
			Expect(cfg["default_auth"]).To(Equal("token"))
			Expect(cfg["enable_flow_preview"]).To(BeFalse())
			// Optional fields should be absent
			Expect(cfg).ToNot(HaveKey("realm_management_api_url"))
			Expect(cfg).ToNot(HaveKey("appengine_api_url"))
			Expect(cfg).ToNot(HaveKey("flow_api_url"))
			Expect(cfg).ToNot(HaveKey("pairing_api_url"))
			Expect(cfg).ToNot(HaveKey("default_realm"))
			// Auth array default
			authIface, exists := cfg["auth"]
			Expect(exists).To(BeTrue())
			authArr, ok := authIface.([]interface{})
			Expect(ok).To(BeTrue())
			Expect(authArr).To(HaveLen(1))
			firstAuth, ok := authArr[0].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(firstAuth["type"]).To(Equal("token"))
		})

		It("should delete existing deployment when disabling", func() {
			// Enable first
			cr.Spec.Components.Dashboard.Deploy = pointy.Bool(true)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, "10s", "250ms").Should(Succeed())
			Expect(EnsureAstarteDashboard(cr, cr.Spec.Components.Dashboard, k8sClient, scheme.Scheme)).To(Succeed())
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name + "-dashboard", Namespace: cr.Namespace}, dep)).To(Succeed())

			// Now disable
			cr.Spec.Components.Dashboard.Deploy = pointy.Bool(false)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, "10s", "250ms").Should(Succeed())
			Expect(EnsureAstarteDashboard(cr, cr.Spec.Components.Dashboard, k8sClient, scheme.Scheme)).To(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name + "-dashboard", Namespace: cr.Namespace}, &appsv1.Deployment{})
				return err != nil
			}, "10s", "250ms").Should(BeTrue())
		})

		It("should enable flow preview when Flow component is deployed", func() {
			// Mark Flow component as deployed in CR
			cr.Spec.Components.Flow = apiv2alpha1.AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: apiv2alpha1.AstarteGenericClusteredResource{Deploy: pointy.Bool(true)}}
			cr.Spec.Components.Dashboard.Deploy = pointy.Bool(true)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteDashboard(cr, cr.Spec.Components.Dashboard, k8sClient, scheme.Scheme)).To(Succeed())

			cm := &v1.ConfigMap{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name + "-dashboard-config", Namespace: cr.Namespace}, cm)).To(Succeed())
			cfgStr := cm.Data["config.json"]
			var cfg map[string]interface{}
			Expect(json.Unmarshal([]byte(cfgStr), &cfg)).To(Succeed())
			Expect(cfg["enable_flow_preview"]).To(BeTrue())
		})

		It("should apply optional config fields and replica count", func() {
			dashboardSpec := apiv2alpha1.AstarteDashboardSpec{
				AstarteGenericClusteredResource: apiv2alpha1.AstarteGenericClusteredResource{
					Deploy:   pointy.Bool(true),
					Replicas: pointy.Int32(2),
				},
				AstarteDashboardConfigSpec: apiv2alpha1.AstarteDashboardConfigSpec{
					RealmManagementAPIURL: "https://realm-mng.example.com",
					AppEngineAPIURL:       "https://appengine.example.com",
					FlowAPIURL:            "https://flow.example.com",
					PairingAPIURL:         "https://pairing.example.com",
					DefaultRealm:          "myrealm",
					DefaultAuth:           "token",
					Auth:                  []apiv2alpha1.AstarteDashboardConfigAuthSpec{{Type: "token"}},
				},
			}

			// Update CR's spec for consistency with helper functions
			cr.Spec.Components.Dashboard = dashboardSpec
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteDashboard(cr, dashboardSpec, k8sClient, scheme.Scheme)).To(Succeed())

			// Deployment should reflect replica count
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name + "-dashboard", Namespace: cr.Namespace}, dep)).To(Succeed())
			Expect(dep.Spec.Replicas).ToNot(BeNil())
			Expect(*dep.Spec.Replicas).To(Equal(int32(2)))

			// ConfigMap should contain optional fields
			cm := &v1.ConfigMap{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: cr.Name + "-dashboard-config", Namespace: cr.Namespace}, cm)).To(Succeed())
			cfgStr := cm.Data["config.json"]
			var cfg map[string]interface{}
			Expect(json.Unmarshal([]byte(cfgStr), &cfg)).To(Succeed())
			Expect(cfg["realm_management_api_url"]).To(Equal("https://realm-mng.example.com"))
			Expect(cfg["appengine_api_url"]).To(Equal("https://appengine.example.com"))
			Expect(cfg["flow_api_url"]).To(Equal("https://flow.example.com"))
			Expect(cfg["pairing_api_url"]).To(Equal("https://pairing.example.com"))
			Expect(cfg["default_realm"]).To(Equal("myrealm"))
			Expect(cfg["default_auth"]).To(Equal("token"))
		})
	})
})
