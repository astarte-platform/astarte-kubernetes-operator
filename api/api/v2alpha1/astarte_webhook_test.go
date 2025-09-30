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

	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("Astarte Webhook testing", Ordered, Serial, func() {
	const (
		CustomSecretName       = "custom-secret"
		CustomAstarteName      = "example-astarte"
		CustomAstarteNamespace = "astarte-webhook-tests"
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

	Describe("TestValidateSSLListener", func() {
		BeforeEach(func() {
			// Customize cr for SSL listener testing
			cr.Spec.VerneMQ = AstarteVerneMQSpec{}
		})

		It("should return no errors when SSL Listener is disabled", func() {
			cr.Spec.VerneMQ.SSLListener = pointy.Bool(false)
			errs := cr.validateSSLListener()
			Expect(errs).ToNot(BeNil())
			Expect(errs).To(BeEmpty())
		})

		It("should return no errors when SSL Listener is nil (default false)", func() {
			cr.Spec.VerneMQ.SSLListener = nil
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
			Expect(errs[0].Type).To(Equal(field.ErrorTypeInvalid))
			Expect(errs[0].Field).To(Equal("spec.vernemq.sslListenerCertSecretName"))
		})

		It("should return an error when SSL Listener is valid but there is no a secret", func() {
			cr.Spec.VerneMQ.SSLListener = pointy.Bool(true)
			cr.Spec.VerneMQ.SSLListenerCertSecretName = CustomSecretName
			errs := cr.validateSSLListener()
			Expect(errs).ToNot(BeNil())
			Expect(errs).To(HaveLen(1))
			Expect(errs[0].Type).To(Equal(field.ErrorTypeNotFound))
			Expect(errs[0].Field).To(Equal("spec.vernemq.sslListenerCertSecretName"))
		})

		It("should return no errors when SSL Listener is valid and the secret is deployed", func() {
			cr.Spec.VerneMQ.SSLListener = pointy.Bool(true)
			secretName := "ssl-secret-" + cr.Name // Use unique name to avoid conflicts
			cr.Spec.VerneMQ.SSLListenerCertSecretName = secretName

			// Create the secret
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: CustomAstarteNamespace,
				},
				Data: map[string][]byte{
					"cert": []byte("my-cert"),
					"key":  []byte("my-key"),
				},
			}
			Expect(k8sClient.Create(context.Background(), secret)).To(Succeed())

			// Ensure the secret is created and available in the client cache
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: CustomAstarteNamespace}, &v1.Secret{})
			}, Timeout, Interval).Should(Succeed())

			// Wait for the validation to pass - give more time for webhook client cache sync
			Eventually(func() bool {
				errs := cr.validateSSLListener()
				return errs != nil && len(errs) == 0
			}, Timeout, Interval).Should(BeTrue())

			// Cleanup the secret
			Expect(k8sClient.Delete(context.Background(), secret)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: CustomAstarteNamespace}, &v1.Secret{})
			}, Timeout, Interval).ShouldNot(Succeed())
		})
	})

	Describe("TestValidateUpdateAstarteInstanceID", func() {
		var oldAstarte *Astarte

		BeforeEach(func() {
			// Set up old Astarte instance for update validation
			oldAstarte = cr.DeepCopy()
		})

		It("should return an error when trying to change the instanceID", func() {
			oldAstarte.Spec.AstarteInstanceID = "old-instance-id"
			cr.Spec.AstarteInstanceID = "new-instance-id"

			err := cr.validateUpdateAstarteInstanceID(oldAstarte)
			Expect(err).ToNot(BeNil())
			Expect(err.Type).To(Equal(field.ErrorTypeInvalid))
			Expect(err.Field).To(Equal("spec.astarteInstanceID"))
		})

		It("should NOT return an error when the instanceID is unchanged", func() {
			oldAstarte.Spec.AstarteInstanceID = "same-instance-id"
			cr.Spec.AstarteInstanceID = "same-instance-id"

			err := cr.validateUpdateAstarteInstanceID(oldAstarte)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should NOT return an error when the instanceID is empty in both old and new spec", func() {
			oldAstarte.Spec.AstarteInstanceID = ""
			cr.Spec.AstarteInstanceID = ""

			err := cr.validateUpdateAstarteInstanceID(oldAstarte)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TestValidatePodLabelsForClusteredResources", func() {
		var allowedLabels, notAllowedLabels map[string]string

		BeforeEach(func() {
			// Set up test labels
			allowedLabels = map[string]string{
				"custom-label":           "lbl",
				"my.custom.domain/label": "value",
			}

			notAllowedLabels = map[string]string{
				"app":          "my-app",
				"component":    "my-component",
				"astarte-role": "my-role",
				"flow-role":    "my-role",
			}

			// Initialize components for testing
			cr.Spec.Components = AstarteComponentsSpec{
				DataUpdaterPlant: AstarteDataUpdaterPlantSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}},
				TriggerEngine:    AstarteTriggerEngineSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}},
				Flow:             AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}},
				Housekeeping:     AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}},
				RealmManagement:  AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}},
				Pairing:          AstarteGenericAPIComponentSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}},
			}
			cr.Spec.VerneMQ = AstarteVerneMQSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{}}
		})

		It("should not return errors when using allowed custom labels", func() {
			cr.Spec.Components.DataUpdaterPlant.PodLabels = allowedLabels
			cr.Spec.Components.TriggerEngine.PodLabels = allowedLabels
			cr.Spec.Components.Flow.PodLabels = allowedLabels
			cr.Spec.Components.Housekeeping.PodLabels = allowedLabels
			cr.Spec.Components.RealmManagement.PodLabels = allowedLabels
			cr.Spec.Components.Pairing.PodLabels = allowedLabels
			cr.Spec.VerneMQ.PodLabels = allowedLabels

			err := cr.validatePodLabelsForClusteredResources()
			Expect(err).ToNot(BeNil())
			Expect(err).To(BeEmpty())
		})

		It("should return errors when using unallowed reserved labels", func() {
			cr.Spec.Components.DataUpdaterPlant.PodLabels = notAllowedLabels

			err := cr.validatePodLabelsForClusteredResources()
			Expect(err).ToNot(BeNil())
			Expect(err).ToNot(BeEmpty())
		})

		It("should not return errors when no labels are set", func() {
			cr.Spec.Components.DataUpdaterPlant.PodLabels = nil
			cr.Spec.Components.TriggerEngine.PodLabels = nil
			cr.Spec.Components.Flow.PodLabels = nil
			cr.Spec.Components.Housekeeping.PodLabels = nil
			cr.Spec.Components.RealmManagement.PodLabels = nil
			cr.Spec.Components.Pairing.PodLabels = nil
			cr.Spec.VerneMQ.PodLabels = nil

			err := cr.validatePodLabelsForClusteredResources()
			Expect(err).ToNot(BeNil())
			Expect(err).To(BeEmpty())
		})

		It("should handle CFSSL component pod labels validation", func() {
			cr.Spec.CFSSL = AstarteCFSSLSpec{
				PodLabels: map[string]string{
					"app": "invalid", // Should trigger error
				},
			}
			err := cr.validatePodLabelsForClusteredResources()
			Expect(err).ToNot(BeNil())
			Expect(err).ToNot(BeEmpty())
			Expect(err[0].Field).To(Equal("podLabels"))
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

		It("should handle multiple invalid labels", func() {
			r := &AstarteGenericClusteredResource{
				PodLabels: map[string]string{
					"app":          "my-app",
					"component":    "my-component",
					"astarte-role": "my-role",
					"flow-role":    "my-role",
					"astarte-test": "test",
					"flow-test":    "test",
				},
			}
			err := validatePodLabelsForClusteredResource(PodLabelsGetter(r))
			Expect(err).ToNot(BeNil())
			Expect(err).To(HaveLen(6)) // All 6 labels should be invalid
		})

		It("should allow labels with astarte- and flow- in the middle or end", func() {
			r := &AstarteGenericClusteredResource{
				PodLabels: map[string]string{
					"example-astarte-label": "valid", // astarte- in middle is OK
					"label-flow-end":        "valid", // flow- in middle is OK
					"myflow":                "valid", // flow without dash is OK
					"myastarte":             "valid", // astarte without dash is OK
				},
			}
			err := validatePodLabelsForClusteredResource(PodLabelsGetter(r))
			Expect(err).ToNot(BeNil())
			Expect(err).To(BeEmpty())
		})
	})

	Describe("TestValidateAutoscalerForClusteredResources", func() {
		BeforeEach(func() {
			// Initialize components for autoscaler testing
			cr.Spec.Components = AstarteComponentsSpec{
				DataUpdaterPlant: AstarteDataUpdaterPlantSpec{
					AstarteGenericClusteredResource: AstarteGenericClusteredResource{},
				},
			}
		})

		It("should return error when autoscaling horizontally on excluded components with autoscaling enabled", func() {
			cr.Spec.Features = AstarteFeatures{Autoscaling: true}
			cr.Spec.Components.DataUpdaterPlant.Autoscale = &AstarteGenericClusteredResourceAutoscalerSpec{Horizontal: "hpa"}

			err := validateAutoscalerForClusteredResources(cr)
			Expect(err).ToNot(BeNil())
		})

		It("should not return error when autoscaling disabled", func() {
			cr.Spec.Features = AstarteFeatures{Autoscaling: false}
			cr.Spec.Components.DataUpdaterPlant.Autoscale = &AstarteGenericClusteredResourceAutoscalerSpec{Horizontal: "hpa"}

			err := validateAutoscalerForClusteredResources(cr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not return error when autoscaling enabled but Autoscale is nil", func() {
			cr.Spec.Features = AstarteFeatures{Autoscaling: true}
			cr.Spec.Components.DataUpdaterPlant.Autoscale = nil

			err := validateAutoscalerForClusteredResources(cr)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TestValidateAutoscalerForClusteredResourcesExcluding", func() {
		BeforeEach(func() {
			// Initialize components for autoscaler excluding testing
			cr.Spec.Components = AstarteComponentsSpec{
				DataUpdaterPlant: AstarteDataUpdaterPlantSpec{
					AstarteGenericClusteredResource: AstarteGenericClusteredResource{},
				},
			}
		})

		It("should return error when excluded resources include a horizontally autoscaled component", func() {
			cr.Spec.Features = AstarteFeatures{Autoscaling: true}
			cr.Spec.Components.DataUpdaterPlant.Autoscale = &AstarteGenericClusteredResourceAutoscalerSpec{Horizontal: "hpa"}

			excluded := []AstarteGenericClusteredResource{cr.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource}
			err := validateAutoscalerForClusteredResourcesExcluding(cr, excluded)
			Expect(err).ToNot(BeNil())
		})

		It("should not return error when excluded resources include a horizontally autoscaled component but Autoscale is nil", func() {
			cr.Spec.Features = AstarteFeatures{Autoscaling: true}
			cr.Spec.Components.DataUpdaterPlant.Autoscale = nil

			excluded := []AstarteGenericClusteredResource{cr.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource}
			err := validateAutoscalerForClusteredResourcesExcluding(cr, excluded)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TestValidateAstartePriorityClasses", func() {
		BeforeEach(func() {
			// Initialize features for priority class testing
			cr.Spec.Features = AstarteFeatures{}
		})

		It("should not return an error when pod priorities are disabled and values are in correct order", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{
				Enable:              false,
				AstarteHighPriority: pointy.Int(1000),
				AstarteMidPriority:  pointy.Int(500),
				AstarteLowPriority:  pointy.Int(100),
			}

			err := cr.validateAstartePriorityClasses()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not return an error when pod priorities are nil", func() {
			cr.Spec.Features.AstartePodPriorities = nil

			err := cr.validateAstartePriorityClasses()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not return an error when pod priorities are disabled and values are not in correct order", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{
				Enable:              false,
				AstarteHighPriority: pointy.Int(0),
				AstarteMidPriority:  pointy.Int(500),
				AstarteLowPriority:  pointy.Int(1000),
			}

			err := cr.validateAstartePriorityClasses()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return an error when pod priorities are enabled and values are not in correct order", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{
				Enable:              true,
				AstarteHighPriority: pointy.Int(500),
				AstarteMidPriority:  pointy.Int(500),
				AstarteLowPriority:  pointy.Int(1000),
			}

			err := cr.validateAstartePriorityClasses()
			Expect(err).ToNot(BeNil())
			Expect(err.Field).To(Equal("spec.features.astarte{Low|Medium|High}Priority"))
		})

		It("should not return an error when pod priorities are enabled and values are in correct order", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{
				Enable:              true,
				AstarteHighPriority: pointy.Int(1000),
				AstarteMidPriority:  pointy.Int(500),
				AstarteLowPriority:  pointy.Int(100),
			}

			err := cr.validateAstartePriorityClasses()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TestValidatePriorityClassesValues", func() {
		BeforeEach(func() {
			// Initialize features for priority class values testing
			cr.Spec.Features = AstarteFeatures{}
		})

		It("should not return an error when priorities are in correct order", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{
				Enable:              true,
				AstarteHighPriority: pointy.Int(1000),
				AstarteMidPriority:  pointy.Int(500),
				AstarteLowPriority:  pointy.Int(100),
			}

			err := cr.validatePriorityClassesValues()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return an error when high priority is less than mid priority", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{
				Enable:              true,
				AstarteHighPriority: pointy.Int(400),
				AstarteMidPriority:  pointy.Int(500),
				AstarteLowPriority:  pointy.Int(100),
			}

			err := cr.validatePriorityClassesValues()
			Expect(err).ToNot(BeNil())
			Expect(err.Field).To(Equal("spec.features.astarte{Low|Medium|High}Priority"))
		})

		It("should return an error when mid priority is less than low priority", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{
				Enable:              true,
				AstarteHighPriority: pointy.Int(1000),
				AstarteMidPriority:  pointy.Int(50),
				AstarteLowPriority:  pointy.Int(100),
			}

			err := cr.validatePriorityClassesValues()
			Expect(err).ToNot(BeNil())
			Expect(err.Field).To(Equal("spec.features.astarte{Low|Medium|High}Priority"))
		})

		It("should not return an error when priorities are equal", func() {
			cr.Spec.Features.AstartePodPriorities = &AstartePodPrioritiesSpec{
				Enable:              true,
				AstarteHighPriority: pointy.Int(500),
				AstarteMidPriority:  pointy.Int(500),
				AstarteLowPriority:  pointy.Int(100),
			}

			err := cr.validatePriorityClassesValues()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TestValidateUpdateAstarteSystemKeyspace", func() {
		var oldAstarte *Astarte

		BeforeEach(func() {
			// Initialize Cassandra configuration for keyspace update testing
			cr.Spec.Cassandra = AstarteCassandraSpec{
				AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{},
			}

			oldAstarte = cr.DeepCopy()
			oldAstarte.Spec.Cassandra = AstarteCassandraSpec{
				AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{},
			}
		})

		It("should return an error when trying to change the keyspace", func() {
			oldAstarte.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{
				ReplicationStrategy:   "SimpleStrategy",
				ReplicationFactor:     1,
				DataCenterReplication: "dc1:3,dc2:2",
			}
			cr.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{
				ReplicationStrategy:   "NetworkTopologyStrategy",
				DataCenterReplication: "dc1:2,dc2:3",
			}

			err := cr.validateUpdateAstarteSystemKeyspace(oldAstarte)
			Expect(err).ToNot(BeNil())
			Expect(err.Field).To(Equal("spec.cassandra.astarteSystemKeyspace"))
		})

		It("should NOT return an error when the keyspace is unchanged", func() {
			oldAstarte.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{
				ReplicationStrategy:   "SimpleStrategy",
				ReplicationFactor:     1,
				DataCenterReplication: "dc1:3,dc2:2",
			}

			cr.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{
				ReplicationStrategy:   "SimpleStrategy",
				ReplicationFactor:     1,
				DataCenterReplication: "dc1:3,dc2:2",
			}

			err := cr.validateUpdateAstarteSystemKeyspace(oldAstarte)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should NOT return an error when the keyspace is empty in both old and new spec", func() {
			oldAstarte.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{}
			cr.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{}

			err := cr.validateUpdateAstarteSystemKeyspace(oldAstarte)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TestValidateCFSSLDefinition", func() {
		BeforeEach(func() {
			// Initialize CFSSL configuration for testing
			cr.Spec.CFSSL = AstarteCFSSLSpec{}
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
			Expect(err).ToNot(HaveOccurred())
		})

		It("should NOT return an error when Deploy is true and URL is empty", func() {
			cr.Spec.CFSSL.Deploy = pointy.Bool(true)
			cr.Spec.CFSSL.URL = ""

			err := cr.validateCFSSLDefinition()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should NOT return an error when Deploy is true and URL is provided", func() {
			cr.Spec.CFSSL.Deploy = pointy.Bool(true)
			cr.Spec.CFSSL.URL = "http://my-cfssl.com"

			err := cr.validateCFSSLDefinition()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should NOT return an error when Deploy is nil (defaults to true) and URL is empty", func() {
			cr.Spec.CFSSL.Deploy = nil
			cr.Spec.CFSSL.URL = ""

			err := cr.validateCFSSLDefinition()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should NOT return an error when Deploy is nil (defaults to true) and URL is provided", func() {
			cr.Spec.CFSSL.Deploy = nil
			cr.Spec.CFSSL.URL = "http://my-cfssl.com"
			err := cr.validateCFSSLDefinition()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle URL with whitespace", func() {
			cr.Spec.CFSSL.Deploy = pointy.Bool(false)
			cr.Spec.CFSSL.URL = "   " // Whitespace only
			err := cr.validateCFSSLDefinition()
			// Current implementation only checks for empty string, not whitespace
			// This is a potential improvement area - whitespace-only URLs should probably be invalid
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TestValidateCreateAstarteSystemKeyspace", func() {
		BeforeEach(func() {
			// Initialize Cassandra keyspace configuration for create testing
			cr.Spec.Cassandra = AstarteCassandraSpec{
				AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{},
			}
		})

		It("should not return error with SimpleStrategy and valid odd replication factor", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "SimpleStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationFactor = 3

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil())
			Expect(err).To(BeEmpty())
		})

		It("should return an error with SimpleStrategy and zero replication factor", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "SimpleStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationFactor = 0

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil())
			Expect(err).To(HaveLen(1))
			Expect(err[0].Field).To(Equal("spec.cassandra.astarteSystemKeyspace.replicationFactor"))
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
			Expect(err[0].Field).To(Equal("spec.cassandra.astarteSystemKeyspace.dataCenterReplication"))
		})

		It("should handle empty DataCenterReplication with NetworkTopologyStrategy", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = ""
			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil())
			Expect(err).To(HaveLen(1))
		})
	})

	Describe("TestValidateCreate", func() {
		BeforeEach(func() {
			// Set up basic configuration for create validation testing
			cr.Spec.API = AstarteAPISpec{Host: "test.example.com"}
			cr.Spec.VerneMQ = AstarteVerneMQSpec{SSLListener: pointy.Bool(false)}
			cr.Spec.Cassandra = AstarteCassandraSpec{
				AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
					ReplicationStrategy: "SimpleStrategy",
					ReplicationFactor:   3,
				},
			}
		})

		It("should succeed when spec passes all validations", func() {
			cr.Spec.AstarteInstanceID = "coverageid1"
			w, err := cr.ValidateCreate()
			Expect(w).To(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return invalid when SSL listener enabled without secret", func() {
			cr.Spec.AstarteInstanceID = "coverageid2"
			cr.Spec.VerneMQ = AstarteVerneMQSpec{SSLListener: pointy.Bool(true), SSLListenerCertSecretName: ""}
			w, err := cr.ValidateCreate()
			Expect(w).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsInvalid(err)).To(BeTrue())
		})

		It("should return invalid when keyspace has invalid replication factor", func() {
			cr.Spec.AstarteInstanceID = "coverageid3"
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationFactor = 2 // Even number - invalid
			w, err := cr.ValidateCreate()
			Expect(w).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsInvalid(err)).To(BeTrue())
		})
	})

	Describe("TestValidateUpdate", func() {
		var oldObj *Astarte

		BeforeEach(func() {
			// Set up old object for update validation testing
			oldObj = cr.DeepCopy()
		})

		It("should return invalid when astarteInstanceID changes", func() {
			oldObj.Spec.AstarteInstanceID = "old"
			cr.Spec.AstarteInstanceID = "new"
			w, err := cr.ValidateUpdate(oldObj)
			Expect(w).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsInvalid(err)).To(BeTrue())
		})

		It("should return invalid when keyspace changes", func() {
			oldObj.Spec.Cassandra = AstarteCassandraSpec{
				AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
					ReplicationStrategy: "SimpleStrategy",
					ReplicationFactor:   3,
				},
			}
			cr.Spec.Cassandra = AstarteCassandraSpec{
				AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
					ReplicationStrategy: "NetworkTopologyStrategy",
				},
			}
			w, err := cr.ValidateUpdate(oldObj)
			Expect(w).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsInvalid(err)).To(BeTrue())
		})

		It("should succeed when no changes violate validations", func() {
			oldObj.Spec.AstarteInstanceID = "same"
			oldObj.Spec.Cassandra = AstarteCassandraSpec{AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{}}
			cr.Spec.AstarteInstanceID = "same"
			cr.Spec.Cassandra = AstarteCassandraSpec{AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{}}
			w, err := cr.ValidateUpdate(oldObj)
			Expect(w).To(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TestValidateAstarte", func() {
		BeforeEach(func() {
			// Set up configuration that will trigger multiple validation errors
			cr.Spec.VerneMQ = AstarteVerneMQSpec{
				SSLListener:               pointy.Bool(true),
				SSLListenerCertSecretName: "missing-secret",
				AstarteGenericClusteredResource: AstarteGenericClusteredResource{
					PodLabels: map[string]string{"app": "bad"},
				},
			}
			cr.Spec.Features = AstarteFeatures{
				Autoscaling: true,
				AstartePodPriorities: &AstartePodPrioritiesSpec{
					Enable:              true,
					AstarteHighPriority: pointy.Int(500),
					AstarteMidPriority:  pointy.Int(600),
					AstarteLowPriority:  pointy.Int(700),
				},
			}
			cr.Spec.Components = AstarteComponentsSpec{
				DataUpdaterPlant: AstarteDataUpdaterPlantSpec{
					AstarteGenericClusteredResource: AstarteGenericClusteredResource{
						Autoscale: &AstarteGenericClusteredResourceAutoscalerSpec{Horizontal: "hpa"},
					},
				},
			}
			cr.Spec.CFSSL = AstarteCFSSLSpec{Deploy: pointy.Bool(false), URL: ""}
		})

		It("should aggregate multiple errors across validators", func() {
			errs := cr.validateAstarte()
			Expect(errs).ToNot(BeNil())
			Expect(len(errs)).To(BeNumerically(">=", 4))
		})
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
		It("should not return error for instance ID when no other same IDs exist", func() {
			// In the beforeEach of the parent Describe, a CR with empty ID is created
			newCr := cr.DeepCopy()
			newCr.ObjectMeta.Name = "test-empty-id-no-conflict"
			newCr.Spec.AstarteInstanceID = "a1"
			newCr.ResourceVersion = ""

			// Create should succeed
			Eventually(func() error {
				return k8sClient.Create(context.Background(), newCr)
			}, Timeout, Interval).Should(Succeed())

			// Fetch to ensure it's created
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: newCr.Name, Namespace: CustomAstarteNamespace}, &Astarte{})
			}, Timeout, Interval).Should(Succeed())

			// Cleanup
			Eventually(func() error {
				return k8sClient.Delete(context.Background(), newCr)
			}, Timeout, Interval).Should(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: newCr.Name, Namespace: CustomAstarteNamespace}, &Astarte{})
			}, Timeout, Interval).ShouldNot(Succeed())
		})

		It("should return error for empty instance ID when another empty ID exists", func() {
			// A CR with empty ID is already created in the beforeEach of the parent Describe
			// Now try to create another with empty ID - should fail
			newCr := cr.DeepCopy()
			newCr.ObjectMeta.Name = "test-empty-id-conflict"
			newCr.Spec.AstarteInstanceID = ""
			newCr.ResourceVersion = ""

			// Create should fail due to webhook rejection
			Eventually(func() error {
				return k8sClient.Create(context.Background(), newCr)
			}, Timeout, Interval).ShouldNot(Succeed())

			// Test the validator directly
			Eventually(func() bool {
				err := newCr.validateCreateAstarteInstanceID()
				return err != nil && err.Field == "spec.astarteInstanceID" && err.Type == field.ErrorTypeInvalid
			}, Timeout, Interval).Should(BeTrue())
		})

		It("should return error if other Astarte instances exists with same instanceID", func() {
			// We create a CR with a specific instanceID, then try to create another with the same ID
			// Create a CR with a specific instanceID
			cr1 := cr.DeepCopy()
			cr1.ObjectMeta.Name = "first-astarte-unique"
			cr1.Spec.AstarteInstanceID = "myuniqueinstanceid001"
			cr1.ResourceVersion = ""

			// Create should succeed
			Eventually(func() error {
				return k8sClient.Create(context.Background(), cr1)
			}, Timeout, Interval).Should(Succeed())

			// Fetch to ensure it's created
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr1.Name, Namespace: CustomAstarteNamespace}, &Astarte{})
			}, Timeout, Interval).Should(Succeed())

			// Try to validate a new CR with the same instanceID
			cr2 := cr1.DeepCopy()
			cr2.ObjectMeta.Name = "second-astarte-unique"
			cr2.ResourceVersion = ""
			// Keep same instanceID
			// Create should fail due to webhook rejection

			// Create should not succeed
			Eventually(func() error {
				return k8sClient.Create(context.Background(), cr2)
			}, Timeout, Interval).Should(Not(Succeed()))

			// Test the validator directly
			Eventually(func() bool {
				err := cr2.validateCreateAstarteInstanceID()
				return err != nil && err.Field == "spec.astarteInstanceID" && err.Type == field.ErrorTypeInvalid
			}, Timeout, Interval).Should(BeTrue())

			// Cleanup
			Expect(k8sClient.Delete(context.Background(), cr1)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr1.Name, Namespace: CustomAstarteNamespace}, &Astarte{})
			}, Timeout, Interval).ShouldNot(Succeed())
		})

		It("should not return error if no other Astarte instances exists with same instanceID", func() {
			// We create a CR with a specific instanceID, then try to create another with a different ID
			cr1 := cr.DeepCopy()
			cr1.ObjectMeta.Name = "first-astarte-unique"
			cr1.Spec.AstarteInstanceID = "myuniqueinstanceid002a"
			cr1.ResourceVersion = ""

			// Create should succeed
			Eventually(func() error {
				return k8sClient.Create(context.Background(), cr1)
			}, Timeout, Interval).Should(Succeed())

			// Fetch to ensure it's created
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr1.Name, Namespace: CustomAstarteNamespace}, &Astarte{})
			}, Timeout, Interval).Should(Succeed())

			// Create another with a different instanceID
			cr2 := cr1.DeepCopy()
			cr2.ObjectMeta.Name = "second-astarte-unique"
			cr2.Spec.AstarteInstanceID = "myuniqueinstanceid002b"
			cr2.ResourceVersion = ""

			// Create should succeed
			Eventually(func() error {
				return k8sClient.Create(context.Background(), cr2)
			}, Timeout, Interval).Should(Succeed())

			// Fetch to ensure it's created
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr2.Name, Namespace: CustomAstarteNamespace}, &Astarte{})
			}, Timeout, Interval).Should(Succeed())

			// Cleanup
			Expect(k8sClient.Delete(context.Background(), cr1)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr1.Name, Namespace: CustomAstarteNamespace}, &Astarte{})
			}, Timeout, Interval).ShouldNot(Succeed())

			Expect(k8sClient.Delete(context.Background(), cr2)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: cr2.Name, Namespace: CustomAstarteNamespace}, &Astarte{})
			}, Timeout, Interval).ShouldNot(Succeed())
		})
	})
})
