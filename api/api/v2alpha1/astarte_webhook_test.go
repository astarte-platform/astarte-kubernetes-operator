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
package v2alpha1

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Misc utils testing", Ordered, func() {
	const (
		CustomSecretName       = "custom-secret"
		CustomUsernameKey      = "usr"
		CustomPasswordKey      = "pwd"
		CustomAstarteName      = "my-astarte"
		CustomAstarteNamespace = "default"
		CustomRabbitMQHost     = "custom-rabbitmq-host"
		CustomRabbitMQPort     = 5673
		CustomVerneMQHost      = "vernemq.example.com"
		CustomVerneMQPort      = 8884
		AstarteVersion         = "1.3.0"
	)

	var cr *Astarte
	var log logr.Logger

	BeforeAll(func() {
		log.Info("Starting controllerutils tests")
		if CustomAstarteNamespace != "default" {
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: CustomAstarteNamespace,
				},
			}

			Eventually(func() error {
				return k8sClient.Create(context.Background(), ns)
			}, "10s", "250ms").Should(Succeed())
		}
	})

	AfterAll(func() {
		if CustomAstarteNamespace != "default" {
			astartes := &AstarteList{}
			Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())

			for _, a := range astartes.Items {
				Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())
			}

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteNamespace}, &v1.Namespace{})
			}, "10s", "250ms").ShouldNot(Succeed())
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
				Version: AstarteVersion,
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

	})

	Describe("TestValidateSSLListener", func() {
		It("should return no errors when SSL Listener is disabled", func() {
			cr.Spec.VerneMQ.SSLListener = pointy.Bool(false)
			errs := cr.validateSSLListener()
			Expect(errs).ToNot(BeNil())
			Expect(errs).To(BeEmpty())
		})

		It("should return an error when SSL Listener is enabled and SSLListenerCertSecretName is empty", func() {
			cr.Spec.VerneMQ.SSLListener = pointy.Bool(true)
			cr.Spec.VerneMQ.SSLListenerCertSecretName = ""
			errs := cr.validateSSLListener()
			Expect(errs).ToNot(BeNil())
			Expect(errs).To(HaveLen(1))
		})

		It("should return an error when SSL Listener is valid but there is no a secret", func() {
			cr.Spec.VerneMQ.SSLListener = pointy.Bool(true)
			cr.Spec.VerneMQ.SSLListenerCertSecretName = CustomSecretName
			errs := cr.validateSSLListener()
			Expect(errs).ToNot(BeNil())
			Expect(errs).To(HaveLen(1))
		})

		It("should return no errors when SSL Listener is valid and the secret is deployed", func() {
			cr.Spec.VerneMQ.SSLListener = pointy.Bool(true)
			cr.Spec.VerneMQ.SSLListenerCertSecretName = CustomSecretName

			// Create the secret
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      CustomSecretName,
					Namespace: CustomAstarteNamespace,
				},
				Data: map[string][]byte{
					"cert": []byte("my-cert"),
					"key":  []byte("my-key"),
				},
			}
			Expect(k8sClient.Create(context.Background(), secret)).To(Succeed())

			// Ensure the secret is created
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomSecretName, Namespace: CustomAstarteNamespace}, &v1.Secret{})
			}, "10s", "250ms").Should(Succeed())

			errs := cr.validateSSLListener()
			Expect(errs).ToNot(BeNil())
			Expect(errs).To(BeEmpty())

			// Cleanup the secret
			Expect(k8sClient.Delete(context.Background(), secret)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomSecretName, Namespace: CustomAstarteNamespace}, &v1.Secret{})
			}, "10s", "250ms").ShouldNot(Succeed())
		})
	})

	Describe("TestValidateUpdateAstarteInstanceID", func() {
		It("should return an error when trying to change the instanceID", func() {
			oldAstarte := &Astarte{
				Spec: AstarteSpec{
					AstarteInstanceID: "old-instance-id",
				},
			}
			newAstarte := &Astarte{
				Spec: AstarteSpec{
					AstarteInstanceID: "new-instance-id",
				},
			}

			err := newAstarte.validateUpdateAstarteInstanceID(oldAstarte)
			Expect(err).ToNot(BeNil())
		})

		It("should NOT return an error when the instanceID is unchanged", func() {
			oldAstarte := &Astarte{
				Spec: AstarteSpec{
					AstarteInstanceID: "same-instance-id",
				},
			}
			newAstarte := &Astarte{
				Spec: AstarteSpec{
					AstarteInstanceID: "same-instance-id",
				},
			}

			err := newAstarte.validateUpdateAstarteInstanceID(oldAstarte)
			Expect(err).To(BeNil())
		})

		It("should NOT return an error when the instanceID is empty in both old and new spec", func() {
			oldAstarte := &Astarte{
				Spec: AstarteSpec{
					AstarteInstanceID: "",
				},
			}
			newAstarte := &Astarte{
				Spec: AstarteSpec{
					AstarteInstanceID: "",
				},
			}

			err := newAstarte.validateUpdateAstarteInstanceID(oldAstarte)
			Expect(err).To(BeNil())
		})
	})

	Describe("TestValidatePodLabelsForClusteredResources", func() {
		testComponents := map[string]AstarteSpec{
			"DataUpdaterPlant": {Components: AstarteComponentsSpec{DataUpdaterPlant: AstarteDataUpdaterPlantSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}}},
			"TriggerEngine":    {Components: AstarteComponentsSpec{TriggerEngine: AstarteTriggerEngineSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}}},
			"Flow":             {Components: AstarteComponentsSpec{Flow: AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}}},
			"Housekeeping":     {Components: AstarteComponentsSpec{Housekeeping: AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}}},
			"RealmManagement":  {Components: AstarteComponentsSpec{RealmManagement: AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}}},
			"Pairing":          {Components: AstarteComponentsSpec{Pairing: AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}}},
			"VerneMQ":          {VerneMQ: AstarteVerneMQSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}},
		}

		allowedLabels := map[string]string{
			"custom-label":           "lbl",
			"my.custom.domain/label": "value",
		}

		notAllowedLabels := map[string]string{
			"app":          "my-app",
			"component":    "my-component",
			"astarte-role": "my-role",
			"flow-role":    "my-role",
		}

		It("should not return errors when using allowed custom labels", func() {
			for componentName, baseSpec := range testComponents {
				cr := &Astarte{Spec: baseSpec}

				switch componentName {
				case "DataUpdaterPlant":
					cr.Spec.Components.DataUpdaterPlant.PodLabels = allowedLabels
				case "TriggerEngine":
					cr.Spec.Components.TriggerEngine.PodLabels = allowedLabels
				case "Flow":
					cr.Spec.Components.Flow.PodLabels = allowedLabels
				case "Housekeeping":
					cr.Spec.Components.Housekeeping.PodLabels = allowedLabels
				case "RealmManagement":
					cr.Spec.Components.RealmManagement.PodLabels = allowedLabels
				case "Pairing":
					cr.Spec.Components.Pairing.PodLabels = allowedLabels
				case "VerneMQ":
					cr.Spec.VerneMQ.PodLabels = allowedLabels
				}

				err := cr.validatePodLabelsForClusteredResources()
				Expect(err).ToNot(BeNil())
				Expect(err).To(BeEmpty())
			}
		})

		It("should return errors when using unallowed reserved labels", func() {
			for componentName, baseSpec := range testComponents {
				cr := &Astarte{Spec: baseSpec}

				switch componentName {
				case "DataUpdaterPlant":
					cr.Spec.Components.DataUpdaterPlant.PodLabels = notAllowedLabels
				case "TriggerEngine":
					cr.Spec.Components.TriggerEngine.PodLabels = notAllowedLabels
				case "Flow":
					cr.Spec.Components.Flow.PodLabels = notAllowedLabels
				case "Housekeeping":
					cr.Spec.Components.Housekeeping.PodLabels = notAllowedLabels
				case "RealmManagement":
					cr.Spec.Components.RealmManagement.PodLabels = notAllowedLabels
				case "Pairing":
					cr.Spec.Components.Pairing.PodLabels = notAllowedLabels
				case "VerneMQ":
					cr.Spec.VerneMQ.PodLabels = notAllowedLabels
				}

				err := cr.validatePodLabelsForClusteredResources()
				Expect(err).ToNot(BeNil())
				Expect(err).ToNot(BeEmpty())
			}
		})

		It("should not return errors when no labels are set", func() {
			for componentName, baseSpec := range testComponents {
				cr := &Astarte{Spec: baseSpec}

				switch componentName {
				case "DataUpdaterPlant":
					cr.Spec.Components.DataUpdaterPlant.PodLabels = nil
				case "TriggerEngine":
					cr.Spec.Components.TriggerEngine.PodLabels = nil
				case "Flow":
					cr.Spec.Components.Flow.PodLabels = nil
				case "Housekeeping":
					cr.Spec.Components.Housekeeping.PodLabels = nil
				case "RealmManagement":
					cr.Spec.Components.RealmManagement.PodLabels = nil
				case "Pairing":
					cr.Spec.Components.Pairing.PodLabels = nil
				case "VerneMQ":
					cr.Spec.VerneMQ.PodLabels = nil
				}

				err := cr.validatePodLabelsForClusteredResources()
				Expect(err).ToNot(BeNil())
				Expect(err).To(BeEmpty())
			}
		})
	})

	Describe("TestValidatePodLabelsForClusteredResource", func() {
		It("should return no errors when using allowed custom labels", func() {
			r := &AstarteGenericClusteredResource{
				PodLabels: map[string]string{
					"custom-label":           "lbl",
					"my.custom.domain/label": "value",
				},
			}
			err := validatePodLabelsForClusteredResource(PodLabelsGetter(r))
			Expect(err).ToNot(BeNil())
			Expect(err).To(BeEmpty())
		})

		It("should return errors when using unallowed reserved labels", func() {
			r := &AstarteGenericClusteredResource{
				PodLabels: map[string]string{
					"app":          "my-app",
					"component":    "my-component",
					"astarte-role": "my-role",
					"flow-role":    "my-role",
				},
			}
			err := validatePodLabelsForClusteredResource(PodLabelsGetter(r))
			Expect(err).ToNot(BeNil())
			Expect(err).ToNot(BeEmpty())
		})

		It("should not return errors when no labels are set", func() {
			r := &AstarteGenericClusteredResource{
				PodLabels: nil,
			}
			err := validatePodLabelsForClusteredResource(PodLabelsGetter(r))
			Expect(err).ToNot(BeNil())
			Expect(err).To(BeEmpty())
		})
	})

	Describe("TestValidateAutoscalerForClusteredResources", func() {
	})

	Describe("TestValidateAutoscalerForClusteredResourcesExcluding", func() {
	})

	Describe("TestValidateAstartePriorityClasses", func() {
		It("should not return an error when pod priorities are disabled and values are in correct order", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{
						AstartePodPriorities: &AstartePodPrioritiesSpec{
							Enable:              false,
							AstarteHighPriority: pointy.Int(1000),
							AstarteMidPriority:  pointy.Int(500),
							AstarteLowPriority:  pointy.Int(100),
						},
					},
				},
			}

			err := astarte.validateAstartePriorityClasses()
			Expect(err).To(BeNil())
		})

		It("should not return an error when pod priorities are disabled and values are not in correct order", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{
						AstartePodPriorities: &AstartePodPrioritiesSpec{
							Enable:              false,
							AstarteHighPriority: pointy.Int(0),
							AstarteMidPriority:  pointy.Int(500),
							AstarteLowPriority:  pointy.Int(1000),
						},
					},
				},
			}

			err := astarte.validateAstartePriorityClasses()
			Expect(err).To(BeNil())
		})

		It("should return an error when pod priorities are enabled and values are not in correct order", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{
						AstartePodPriorities: &AstartePodPrioritiesSpec{
							Enable:              true,
							AstarteHighPriority: pointy.Int(500),
							AstarteMidPriority:  pointy.Int(500),
							AstarteLowPriority:  pointy.Int(1000),
						},
					},
				},
			}

			err := astarte.validateAstartePriorityClasses()
			Expect(err).ToNot(BeNil())
			Expect(err.Field).To(Equal("spec.features.astarte{Low|Medium|High}Priority"))
		})

		It("should not return an error when pod priorities are enabled and values are in correct order", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{
						AstartePodPriorities: &AstartePodPrioritiesSpec{
							Enable:              true,
							AstarteHighPriority: pointy.Int(1000),
							AstarteMidPriority:  pointy.Int(500),
							AstarteLowPriority:  pointy.Int(100),
						},
					},
				},
			}

			err := astarte.validateAstartePriorityClasses()
			Expect(err).To(BeNil())
		})
	})

	Describe("TestValidatePriorityClassesValues", func() {
		It("should not return an error when priorities are in correct order", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{
						AstartePodPriorities: &AstartePodPrioritiesSpec{
							Enable:              true,
							AstarteHighPriority: pointy.Int(1000),
							AstarteMidPriority:  pointy.Int(500),
							AstarteLowPriority:  pointy.Int(100),
						},
					},
				},
			}

			err := astarte.validatePriorityClassesValues()
			Expect(err).To(BeNil())
		})

		It("should return an error when high priority is less than mid priority", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{
						AstartePodPriorities: &AstartePodPrioritiesSpec{
							Enable:              true,
							AstarteHighPriority: pointy.Int(400),
							AstarteMidPriority:  pointy.Int(500),
							AstarteLowPriority:  pointy.Int(100),
						},
					},
				},
			}

			err := astarte.validatePriorityClassesValues()
			Expect(err).ToNot(BeNil())
			Expect(err.Field).To(Equal("spec.features.astarte{Low|Medium|High}Priority"))
		})

		It("should return an error when mid priority is less than low priority", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{
						AstartePodPriorities: &AstartePodPrioritiesSpec{
							Enable:              true,
							AstarteHighPriority: pointy.Int(1000),
							AstarteMidPriority:  pointy.Int(50),
							AstarteLowPriority:  pointy.Int(100),
						},
					},
				},
			}

			err := astarte.validatePriorityClassesValues()
			Expect(err).ToNot(BeNil())
			Expect(err.Field).To(Equal("spec.features.astarte{Low|Medium|High}Priority"))
		})

		It("should not return an error when priorities are equal", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{
						AstartePodPriorities: &AstartePodPrioritiesSpec{
							Enable:              true,
							AstarteHighPriority: pointy.Int(500),
							AstarteMidPriority:  pointy.Int(500),
							AstarteLowPriority:  pointy.Int(100),
						},
					},
				},
			}

			err := astarte.validatePriorityClassesValues()
			Expect(err).To(BeNil())
		})
	})

	Describe("TestValidateUpdateAstarteSystemKeyspace", func() {
		oldAstarte := &Astarte{}
		newAstarte := &Astarte{}

		BeforeEach(func() {
			oldAstarte = &Astarte{
				Spec: AstarteSpec{
					Cassandra: AstarteCassandraSpec{
						AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{},
					},
				},
			}

			newAstarte = &Astarte{
				Spec: AstarteSpec{
					Cassandra: AstarteCassandraSpec{
						AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{},
					},
				},
			}
		})

		AfterEach(func() {
			oldAstarte = nil
			newAstarte = nil
		})

		It("should return an error when trying to change the keyspace", func() {
			oldAstarte.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{
				ReplicationStrategy:   "SimpleStrategy",
				ReplicationFactor:     1,
				DataCenterReplication: "dc1:3,dc2:2",
			}
			newAstarte.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{
				ReplicationStrategy:   "NetworkTopologyStrategy",
				ReplicationFactor:     2,
				DataCenterReplication: "dc1:2,dc2:3",
			}

			err := newAstarte.validateUpdateAstarteSystemKeyspace(oldAstarte)
			Expect(err).ToNot(BeNil())
			Expect(err.Field).To(Equal("spec.cassandra.astarteSystemKeyspace"))
		})

		It("should NOT return an error when the keyspace is unchanged", func() {
			oldAstarte.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{
				ReplicationStrategy:   "SimpleStrategy",
				ReplicationFactor:     1,
				DataCenterReplication: "dc1:3,dc2:2",
			}

			newAstarte.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{
				ReplicationStrategy:   "SimpleStrategy",
				ReplicationFactor:     1,
				DataCenterReplication: "dc1:3,dc2:2",
			}

			err := newAstarte.validateUpdateAstarteSystemKeyspace(oldAstarte)
			Expect(err).To(BeNil())
		})

		It("should NOT return an error when the keyspace is empty in both old and new spec", func() {
			oldAstarte.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{}
			newAstarte.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{}

			err := newAstarte.validateUpdateAstarteSystemKeyspace(oldAstarte)
			Expect(err).To(BeNil())
		})
	})

	Describe("TestValidateCFSSLDefinition", func() {
		cr := &Astarte{}
		BeforeEach(func() {
			cr = &Astarte{
				Spec: AstarteSpec{
					CFSSL: AstarteCFSSLSpec{},
				},
			}
		})

		AfterEach(func() {
			cr = nil
		})

		It("should return an error when Deploy is false and URL is empty", func() {
			cr.Spec.CFSSL.Deploy = pointy.Bool(false)
			cr.Spec.CFSSL.URL = ""

			err := cr.validateCFSSLDefinition()
			Expect(err).ToNot(BeNil())
			Expect(err.Field).To(Equal("spec.cfssl.url"))
		})

		It("should NOT return an error when Deploy is false and URL is provided", func() {
			cr.Spec.CFSSL.Deploy = pointy.Bool(false)
			cr.Spec.CFSSL.URL = "http://my-cfssl.com"
			err := cr.validateCFSSLDefinition()
			Expect(err).To(BeNil())
		})

		It("should NOT return an error when Deploy is true and URL is empty", func() {
			cr.Spec.CFSSL.Deploy = pointy.Bool(true)
			cr.Spec.CFSSL.URL = ""

			err := cr.validateCFSSLDefinition()
			Expect(err).To(BeNil())
		})

		It("should NOT return an error when Deploy is true and URL is provided", func() {
			cr.Spec.CFSSL.Deploy = pointy.Bool(true)
			cr.Spec.CFSSL.URL = "http://my-cfssl.com"

			err := cr.validateCFSSLDefinition()
			Expect(err).To(BeNil())
		})

		It("should NOT return an error when Deploy is nil (defaults to true) and URL is empty", func() {
			cr.Spec.CFSSL.Deploy = nil
			cr.Spec.CFSSL.URL = ""

			err := cr.validateCFSSLDefinition()
			Expect(err).To(BeNil())
		})

		It("should NOT return an error when Deploy is nil (defaults to true) and URL is provided", func() {
			cr.Spec.CFSSL.Deploy = nil
			cr.Spec.CFSSL.URL = "http://my-cfssl.com"
			err := cr.validateCFSSLDefinition()
			Expect(err).To(BeNil())
		})
	})

	Describe("TestValidateCreateAstarteSystemKeyspace", func() {
		cr := &Astarte{}

		It("should not return error with SimpleStrategy and valid odd replication factor", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "SimpleStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationFactor = 3

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil())
			Expect(err).To(BeEmpty())
		})

		It("should not return an error with NetworkTopologyStrategy and a single valid DC", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:3"

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil())
			Expect(err).To(BeEmpty())
		})

		It("should not return an error with NetworkTopologyStrategy and multiple valid DCs", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:3,dc2:5,dc3:1"

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil())
			Expect(err).To(BeEmpty())
		})

		It("should return an error with SimpleStrategy and invalid even replication factor", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "SimpleStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationFactor = 2

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil())
			Expect(err).To(HaveLen(1))
		})

		It("should return an error with NetworkTopologyStrategy and invalid format (no colon)", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1"

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).To(HaveLen(1))
		})

		It("should return an error with NetworkTopologyStrategy and invalid format (too many colons)", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:3:bad"

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).To(HaveLen(1))
		})

		It("should return errors with NetworkTopologyStrategy and non-integer replication factor", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:three"
			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil())
			Expect(err).To(HaveLen(2))
		})

		It("should return an error with NetworkTopologyStrategy and even replication factor", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:4"
			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil())
			Expect(err).To(HaveLen(1))

		})

		It("should return an error with NetworkTopologyStrategy and mixed valid and invalid DCs", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:3,dc2:4" // dc2 is invalid
			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil())
			Expect(err).To(HaveLen(1))

		})

		It("should return multiple errors with NetworkTopologyStrategy and multiple invalid entries", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:2,dc2:not-a-number,dc3:5"
			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil())
			Expect(err).To(HaveLen(3))

		})

		It("should return an error with empty replication strategy", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = ""
			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil())
			Expect(err).To(HaveLen(1))

		})
	})

	Describe("TestValidateCreate", func() {

	})

	Describe("TestValidateUpdate", func() {

	})

	Describe("TestValidateAstarte", func() {

	})

	Describe("TestValidateDelete", func() {
		// The function is not implemented, expect nil, nil
		It("should return nil, nil", func() {
			w, err := cr.ValidateDelete()
			Expect(w).To(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TestValidateCreateAstarteInstanceID", func() {
		It("should return error if it cannot list astarte instances", func() {
			// Todo: find a way to simulate this
		})

		It("should return error if other Astarte instances exists with same instanceID", func() {
			// Instance a new CR with a custom instanceID (1)
			cr1 := cr.DeepCopy()
			cr1.ResourceVersion = ""
			cr1.Name = "first-astarte"
			cr1.Spec.AstarteInstanceID = "myuniqueinstanceid"
			Expect(k8sClient.Create(context.Background(), cr1)).To(Succeed())
			// Ensure the instance is created
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: "first-astarte", Namespace: CustomAstarteNamespace}, cr1)
			}, "10s", "250ms").Should(Succeed())

			// Try to validate a new CR with the same instanceID (2)
			cr2 := cr.DeepCopy()
			cr2.ResourceVersion = ""
			cr2.Name = "second-astarte"
			cr2.Spec.AstarteInstanceID = "myuniqueinstanceid"

			err := cr2.validateCreateAstarteInstanceID()
			Expect(err).ToNot(BeNil())
			Expect(err.Field).To(Equal("spec.astarteInstanceID"))

			// Cleanup the first instance
			Expect(k8sClient.Delete(context.Background(), cr1)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: "first-astarte", Namespace: CustomAstarteNamespace}, cr1)
			}, "10s", "250ms").ShouldNot(Succeed())

			// No need to cleanup the second instance, as it was never created
		})

		It("should not return error if no other Astarte instances exists with same instanceID", func() {
			newCr := cr.DeepCopy()
			newCr.Spec.AstarteInstanceID = "myuniqueinstanceid"
			newCr.Name = "another-astarte"

			err := newCr.validateCreateAstarteInstanceID()
			Expect(err).To(BeNil())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			// No need to cleanup the instance, as it was never created
		})
	})
})
