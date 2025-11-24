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

package v2alpha1

import (
	"context"

	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
)

var _ = Describe("Astarte types testing", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "example-astarte"
		CustomAstarteNamespace = "astarte-types-test"
		CustomRabbitMQHost     = "custom-rabbitmq-host"
		CustomRabbitMQPort     = 5673
		CustomVerneMQHost      = "broker.astarte-example.com"
		CustomVerneMQPort      = 8884
	)

	var cr *Astarte

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
		cr.Spec.RabbitMQ.Connection.Host = CustomRabbitMQHost
		cr.Spec.RabbitMQ.Connection.Port = pointy.Int32(CustomRabbitMQPort)
		cr.Spec.VerneMQ.Host = CustomVerneMQHost
		cr.Spec.VerneMQ.Port = pointy.Int32(CustomVerneMQPort)
		integrationutils.DeployAstarte(k8sClient, cr)
	})

	AfterEach(func() {
		integrationutils.TeardownResourcesInNamespace(context.Background(), k8sClient, CustomAstarteNamespace)
	})

	Describe("Test AstartePodPriorities.IsEnabled()", func() {
		It("should return true if AstartePodPrioritiesSpec is enabled", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{}
			cr.Spec.Features.AstartePodPriorities.Enable = true
			enabled := cr.Spec.Features.AstartePodPriorities.IsEnabled()
			Expect(enabled).To(BeTrue())
		})
		It("should return false if AstartePodPrioritiesSpec is disabled", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{}
			cr.Spec.Features.AstartePodPriorities.Enable = false
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
			cr.Spec.CFSSL.PodLabels = map[string]string{
				"cfssl-label": "cfssl",
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
			cApp := AppEngineAPI
			cFlow := FlowComponent
			cDash := Dashboard
			cDup := DataUpdaterPlant
			cHk := Housekeeping
			cRm := RealmManagement
			cPair := Pairing
			cTrig := TriggerEngine
			Expect((&cApp).ServiceRelativePath()).To(Equal("appengine"))
			Expect((&cFlow).ServiceRelativePath()).To(Equal("flow"))
			Expect((&cDash).ServiceRelativePath()).To(Equal("dashboard"))
			Expect((&cDup).ServiceRelativePath()).To(Equal("dataupdaterplant"))
			Expect((&cHk).ServiceRelativePath()).To((Equal("housekeeping")))
			Expect((&cRm).ServiceRelativePath()).To((Equal("realmmanagement")))
			Expect((&cPair).ServiceRelativePath()).To((Equal("pairing")))
			Expect((&cTrig).ServiceRelativePath()).To((Equal("triggerengine")))
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
})
