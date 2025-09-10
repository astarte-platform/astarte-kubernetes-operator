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
package v2alpha1

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Astarte types testing", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "my-astarte"
		CustomAstarteNamespace = "astarte-types-test"
		CustomRabbitMQHost     = "custom-rabbitmq-host"
		CustomRabbitMQPort     = 5673
		CustomVerneMQHost      = "vernemq.example.com"
		CustomVerneMQPort      = 8884
		AstarteVersion         = "1.3.0"
	)

	var cr *Astarte
	var log logr.Logger

	BeforeAll(func() {
		log = logr.Discard()
		log.Info("Starting astarte_types tests")
		if CustomAstarteNamespace != "default" {
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: CustomAstarteNamespace,
				},
			}

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
			// Delete any leftover Astarte CRs in the namespace
			astartes := &AstarteList{}
			Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
			for _, a := range astartes.Items {
				Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &Astarte{})
				}, "10s", "250ms").ShouldNot(Succeed())
			}

			// Attempt namespace deletion but don't block on it in envtest
			_ = k8sClient.Delete(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}})
		}
	})

	BeforeEach(func() {
		// Create and initialize a basic Astarte CR
		cr = &Astarte{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CustomAstarteName,
				Namespace: CustomAstarteNamespace,
			},
			Spec: AstarteSpec{
				// Use a unique AstarteInstanceID to satisfy webhook uniqueness checks across the cluster
				AstarteInstanceID: "astarteinstancetypes",
				Version:           AstarteVersion,
				API: AstarteAPISpec{
					Host: "api.example.com",
				},
				RabbitMQ: AstarteRabbitMQSpec{
					Connection: &AstarteRabbitMQConnectionSpec{
						HostAndPort: HostAndPort{
							Host: CustomRabbitMQHost,
							Port: pointy.Int32(CustomRabbitMQPort),
						},
					},
				},
				VerneMQ: AstarteVerneMQSpec{
					HostAndPort: HostAndPort{
						Host: CustomVerneMQHost,
						Port: pointy.Int32(CustomVerneMQPort),
					},
				},
				Cassandra: AstarteCassandraSpec{
					Connection: &AstarteCassandraConnectionSpec{
						Nodes: []HostAndPort{
							{
								Host: "cassandra.example.com",
								Port: pointy.Int32(9042),
							},
						},
					},
				},
			},
		}

		Expect(k8sClient.Create(context.Background(), cr)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
		}, "10s", "250ms").Should(Succeed())
	})

	AfterEach(func() {
		astartes := &AstarteList{}
		Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, a := range astartes.Items {
			Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &Astarte{})
			}, "10s", "250ms").ShouldNot(Succeed())
		}

		// Ensure all Astarte CRs are gone in the test namespace
		Eventually(func() int {
			list := &AstarteList{}
			if err := k8sClient.List(context.Background(), list, &client.ListOptions{Namespace: CustomAstarteNamespace}); err != nil {
				return -1
			}
			return len(list.Items)
		}, "10s", "250ms").Should(Equal(0))

	})

	Describe("Test AstartePodPriorities.IsEnabled()", func() {
		It("should return true if AstartePodPrioritiesSpec is enabled", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{
				Enable: true,
			}

			enabled := cr.Spec.Features.AstartePodPriorities.IsEnabled()
			Expect(enabled).To(BeTrue())
		})
		It("should return false if AstartePodPrioritiesSpec is disabled", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{
				Enable: false,
			}
			disabled := cr.Spec.Features.AstartePodPriorities.IsEnabled()
			Expect(disabled).To(BeFalse())
		})
		It("should return false if AstartePodPrioritiesSpec is nil", func() {
			cr.Spec.Features.AstartePodPriorities = nil
			disabled := cr.Spec.Features.AstartePodPriorities.IsEnabled()
			Expect(disabled).To(BeFalse())
		})
	})

	Describe("Test CFSSL.GetPodLabels()", func() {
		It("should return a map with the correct pod labels", func() {
			// Set some labels to AstarteCFSSLSpec
			cr.Spec.CFSSL = AstarteCFSSLSpec{
				PodLabels: map[string]string{
					"cfssl-label": "cfssl",
				},
			}
			// Check the returned map
			labels := cr.Spec.CFSSL.GetPodLabels()
			Expect(labels).ToNot(BeNil())
			Expect(labels).To(HaveKeyWithValue("cfssl-label", "cfssl"))
		})

		It("should return nil when PodLabels is not set", func() {
			cr.Spec.CFSSL = AstarteCFSSLSpec{}
			labels := cr.Spec.CFSSL.GetPodLabels()
			Expect(labels).To(BeNil())
		})
	})

	Describe("Test AstarteGenericClusteredResource.GetPodLabels()", func() {
		It("should return a map when PodLabels are set", func() {
			gr := AstarteGenericClusteredResource{
				PodLabels: map[string]string{"k": "v"},
			}
			labels := gr.GetPodLabels()
			Expect(labels).ToNot(BeNil())
			Expect(labels).To(HaveKeyWithValue("k", "v"))
		})

		It("should return nil when PodLabels is not set", func() {
			gr := AstarteGenericClusteredResource{}
			Expect(gr.GetPodLabels()).To(BeNil())
		})
	})

	Describe("Test ReconciliationPhase.String()", func() {
		It("should return the string value of the phase", func() {
			p := ReconciliationPhaseUpgrading
			Expect((&p).String()).To(Equal("Upgrading"))
		})

		It("should return empty string for unknown phase", func() {
			p := ReconciliationPhaseUnknown
			Expect((&p).String()).To(Equal(""))
		})
	})

	Describe("Test AstarteComponent helpers", func() {
		It("String() should return the underlying value", func() {
			c := AppEngineAPI
			Expect((&c).String()).To(Equal("appengine_api"))
		})

		It("DashedString() should replace underscores with hyphens", func() {
			c := DataUpdaterPlant
			Expect((&c).DashedString()).To(Equal("data-updater-plant"))
		})

		It("DockerImageName() should special-case dashboard and prefix others", func() {
			cDash := Dashboard
			cDup := DataUpdaterPlant
			Expect((&cDash).DockerImageName()).To(Equal("astarte-dashboard"))
			Expect((&cDup).DockerImageName()).To(Equal("astarte_data_updater_plant"))
		})

		It("ServiceName() should equal DashedString()", func() {
			c := TriggerEngine
			Expect((&c).ServiceName()).To(Equal("trigger-engine"))
		})

		It("ServiceRelativePath() should return expected values per component", func() {
			// API-like components or explicitly allowed ones
			cApp := AppEngineAPI
			cFlow := FlowComponent
			cDash := Dashboard
			Expect((&cApp).ServiceRelativePath()).To(Equal("appengine"))
			Expect((&cFlow).ServiceRelativePath()).To(Equal("flow"))
			Expect((&cDash).ServiceRelativePath()).To(Equal("dashboard"))

			// Non-API components
			cDup := DataUpdaterPlant
			cHk := Housekeeping
			cRm := RealmManagement
			cPair := Pairing
			cTrig := TriggerEngine
			Expect((&cDup).ServiceRelativePath()).To(BeEmpty())
			Expect((&cHk).ServiceRelativePath()).To(BeEmpty())
			Expect((&cRm).ServiceRelativePath()).To(BeEmpty())
			Expect((&cPair).ServiceRelativePath()).To(BeEmpty())
			Expect((&cTrig).ServiceRelativePath()).To(BeEmpty())
		})
	})

	Describe("Test AstarteResourceEvent.String()", func() {
		It("should return the string representation of the event", func() {
			e := AstarteResourceEventMigration
			Expect(e.String()).To(Equal("Migration"))
		})

		It("should return other event names correctly", func() {
			e := AstarteResourceEventUpgradeError
			Expect(e.String()).To(Equal("ErrUpgrade"))
		})
	})
})
