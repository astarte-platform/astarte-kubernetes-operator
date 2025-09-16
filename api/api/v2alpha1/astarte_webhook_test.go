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
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Astarte Webhook testing", Ordered, Serial, func() {
	const (
		CustomSecretName       = "custom-secret"
		CustomUsernameKey      = "usr"
		CustomPasswordKey      = "pwd"
		CustomAstarteName      = "my-astarte"
		CustomAstarteNamespace = "astarte-webhook-tests"
		CustomRabbitMQHost     = "custom-rabbitmq-host"
		CustomRabbitMQPort     = 5673
		CustomVerneMQHost      = "vernemq.example.com"
		CustomVerneMQPort      = 8884
		AstarteVersion         = "1.3.0"
		Timeout                = "30s"
		Interval               = "1s"
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
				err := k8sClient.Create(context.Background(), ns)
				if apierrors.IsAlreadyExists(err) {
					return nil
				}
				return err
			}, Timeout, Interval).Should(Succeed())
		}
	})

	AfterAll(func() {
		if CustomAstarteNamespace != "default" {
			astartes := &AstarteList{}
			Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())

			for _, a := range astartes.Items {
				Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &Astarte{})
				}, Timeout, Interval).ShouldNot(Succeed())
			}

			// Attempt namespace deletion but don't block on it in envtest
			_ = k8sClient.Delete(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}})
		}
	})

	BeforeEach(func() {
		// Ensure we start with a clean namespace
		astartes := &AstarteList{}
		Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())

		// If there are any leftover resources, wait for them to be cleaned up
		if len(astartes.Items) > 0 {
			Eventually(func() bool {
				list := &AstarteList{}
				if err := k8sClient.List(context.Background(), list, &client.ListOptions{Namespace: CustomAstarteNamespace}); err != nil {
					return false
				}
				return len(list.Items) == 0
			}, Timeout, Interval).Should(BeTrue())
		}

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
		}, Timeout, Interval).Should(Succeed())
	})

	AfterEach(func() {
		// Get a fresh list to avoid stale references
		astartes := &AstarteList{}
		Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())

		for i := range astartes.Items {
			a := &astartes.Items[i]

			// Delete each Astarte resource
			Eventually(func() error {
				return k8sClient.Delete(context.Background(), &astartes.Items[i])
			}, Timeout, Interval).Should(Succeed())

			// Wait for deletion to complete
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &Astarte{})
			}, Timeout, Interval).ShouldNot(Succeed())
		}

		// Also clean up any leftover secrets
		secrets := &v1.SecretList{}
		if err := k8sClient.List(context.Background(), secrets, &client.ListOptions{Namespace: CustomAstarteNamespace}); err == nil {
			for i := range secrets.Items {
				secret := &secrets.Items[i]

				// Delete the secret
				Eventually(func() error {
					return k8sClient.Delete(context.Background(), secret)
				}, Timeout, Interval).Should(Succeed())

				// Wait for deletion to complete
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: secret.Name, Namespace: CustomAstarteNamespace}, &v1.Secret{})
				}, Timeout, Interval).ShouldNot(Succeed())
			}
		}
	})

	Describe("TestValidateSSLListener", func() {
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
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err.Type).To(Equal(field.ErrorTypeInvalid))
			Expect(err.Field).To(Equal("spec.astarteInstanceID"))
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
			Expect(err).ToNot(HaveOccurred())
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
			Expect(err).ToNot(HaveOccurred())
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
				Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
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
				Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
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
				Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
				Expect(err).To(BeEmpty())
			}
		})

		It("should handle CFSSL component pod labels validation", func() {
			cr := &Astarte{
				Spec: AstarteSpec{
					CFSSL: AstarteCFSSLSpec{
						PodLabels: map[string]string{
							"app": "invalid", // Should trigger error
						},
					},
				},
			}
			err := cr.validatePodLabelsForClusteredResources()
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
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
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
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
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).ToNot(BeEmpty())
		})

		It("should not return errors when no labels are set", func() {
			r := &AstarteGenericClusteredResource{
				PodLabels: nil,
			}
			err := validatePodLabelsForClusteredResource(PodLabelsGetter(r))
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
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
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(HaveLen(6)) // All 6 labels should be invalid
		})

		It("should allow labels with astarte- and flow- in the middle or end", func() {
			r := &AstarteGenericClusteredResource{
				PodLabels: map[string]string{
					"my-astarte-label": "valid", // astarte- in middle is OK
					"label-flow-end":   "valid", // flow- in middle is OK
					"myflow":           "valid", // flow without dash is OK
					"myastarte":        "valid", // astarte without dash is OK
				},
			}
			err := validatePodLabelsForClusteredResource(PodLabelsGetter(r))
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(BeEmpty())
		})
	})

	Describe("TestValidateAutoscalerForClusteredResources", func() {
		It("should return error when autoscaling horizontally on excluded components with autoscaling enabled", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{Autoscaling: true},
					Components: AstarteComponentsSpec{
						DataUpdaterPlant: AstarteDataUpdaterPlantSpec{
							AstarteGenericClusteredResource: AstarteGenericClusteredResource{
								Autoscale: &AstarteGenericClusteredResourceAutoscalerSpec{Horizontal: "hpa"},
							},
						},
					},
				},
			}

			err := validateAutoscalerForClusteredResources(astarte)
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
		})

		It("should not return error when autoscaling disabled", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{Autoscaling: false},
					Components: AstarteComponentsSpec{
						DataUpdaterPlant: AstarteDataUpdaterPlantSpec{
							AstarteGenericClusteredResource: AstarteGenericClusteredResource{
								Autoscale: &AstarteGenericClusteredResourceAutoscalerSpec{Horizontal: "hpa"},
							},
						},
					},
				},
			}

			err := validateAutoscalerForClusteredResources(astarte)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not return error when autoscaling enabled but Autoscale is nil", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{Autoscaling: true},
					Components: AstarteComponentsSpec{
						DataUpdaterPlant: AstarteDataUpdaterPlantSpec{
							AstarteGenericClusteredResource: AstarteGenericClusteredResource{
								Autoscale: nil,
							},
						},
					},
				},
			}

			err := validateAutoscalerForClusteredResources(astarte)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TestValidateAutoscalerForClusteredResourcesExcluding", func() {
		It("should return error when excluded resources include a horizontally autoscaled component", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{Autoscaling: true},
					Components: AstarteComponentsSpec{
						DataUpdaterPlant: AstarteDataUpdaterPlantSpec{
							AstarteGenericClusteredResource: AstarteGenericClusteredResource{
								Autoscale: &AstarteGenericClusteredResourceAutoscalerSpec{Horizontal: "hpa"},
							},
						},
					},
				},
			}
			excluded := []AstarteGenericClusteredResource{astarte.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource}
			err := validateAutoscalerForClusteredResourcesExcluding(astarte, excluded)
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
		})

		It("should not return error when excluded resources include a horizontally autoscaled component but Autoscale is nil", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{Autoscaling: true},
					Components: AstarteComponentsSpec{
						DataUpdaterPlant: AstarteDataUpdaterPlantSpec{
							AstarteGenericClusteredResource: AstarteGenericClusteredResource{
								Autoscale: nil,
							},
						},
					},
				},
			}
			excluded := []AstarteGenericClusteredResource{astarte.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource}
			err := validateAutoscalerForClusteredResourcesExcluding(astarte, excluded)
			Expect(err).ToNot(HaveOccurred())
		})
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
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not return an error when pod priorities are nil", func() {
			astarte := &Astarte{
				Spec: AstarteSpec{
					Features: AstarteFeatures{
						AstartePodPriorities: nil,
					},
				},
			}

			err := astarte.validateAstartePriorityClasses()
			Expect(err).ToNot(HaveOccurred())
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
			Expect(err).ToNot(HaveOccurred())
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
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
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
			Expect(err).ToNot(HaveOccurred())
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
			Expect(err).ToNot(HaveOccurred())
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
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
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
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
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
			Expect(err).ToNot(HaveOccurred())
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
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
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
			Expect(err).ToNot(HaveOccurred())
		})

		It("should NOT return an error when the keyspace is empty in both old and new spec", func() {
			oldAstarte.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{}
			newAstarte.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{}

			err := newAstarte.validateUpdateAstarteSystemKeyspace(oldAstarte)
			Expect(err).ToNot(HaveOccurred())
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
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
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
			// Reset keyspace to empty state
			cr.Spec.Cassandra.AstarteSystemKeyspace = AstarteSystemKeyspaceSpec{}
		})

		It("should not return error with SimpleStrategy and valid odd replication factor", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "SimpleStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationFactor = 3

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(BeEmpty())
		})

		It("should return an error with SimpleStrategy and zero replication factor", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "SimpleStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationFactor = 0

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(HaveLen(1))
			Expect(err[0].Field).To(Equal("spec.cassandra.astarteSystemKeyspace.replicationFactor"))
		})

		It("should not return an error with NetworkTopologyStrategy and a single valid DC", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:3"

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(BeEmpty())
		})

		It("should not return an error with NetworkTopologyStrategy and multiple valid DCs", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:3,dc2:5,dc3:1"

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(BeEmpty())
		})

		It("should return an error with SimpleStrategy and invalid even replication factor", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "SimpleStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationFactor = 2

			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
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
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(HaveLen(2))
		})

		It("should return an error with NetworkTopologyStrategy and even replication factor", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:4"
			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(HaveLen(1))

		})

		It("should return an error with NetworkTopologyStrategy and mixed valid and invalid DCs", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:3,dc2:4" // dc2 is invalid
			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(HaveLen(1))

		})

		It("should return multiple errors with NetworkTopologyStrategy and multiple invalid entries", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:2,dc2:not-a-number,dc3:5"
			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(HaveLen(3))

		})

		It("should return an error with empty replication strategy", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = ""
			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(HaveLen(1))
			Expect(err[0].Field).To(Equal("spec.cassandra.astarteSystemKeyspace.dataCenterReplication"))
		})

		It("should handle empty DataCenterReplication with NetworkTopologyStrategy", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "NetworkTopologyStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = ""
			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(HaveLen(1))
		})

		It("should handle unknown replication strategy (defaults to NetworkTopologyStrategy behavior)", func() {
			cr.Spec.Cassandra.AstarteSystemKeyspace.ReplicationStrategy = "UnknownStrategy"
			cr.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication = "dc1:3"
			err := cr.validateCreateAstarteSystemKeyspace()
			Expect(err).ToNot(BeNil()) //nolint:ginkgolinter
			Expect(err).To(BeEmpty())
		})
	})

	Describe("TestValidateCreate", func() {
		It("should succeed when spec passes all validations", func() {
			obj := &Astarte{
				ObjectMeta: metav1.ObjectMeta{Name: "vc-astarte", Namespace: CustomAstarteNamespace},
				Spec: AstarteSpec{
					Version:           AstarteVersion,
					AstarteInstanceID: "coverageid1",
					VerneMQ:           AstarteVerneMQSpec{SSLListener: pointy.Bool(false)},
					Cassandra: AstarteCassandraSpec{AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy: "SimpleStrategy",
						ReplicationFactor:   3,
					}},
					API: AstarteAPISpec{Host: "test.example.com"},
				},
			}
			w, err := obj.ValidateCreate()
			Expect(w).To(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return invalid when SSL listener enabled without secret", func() {
			obj := &Astarte{
				ObjectMeta: metav1.ObjectMeta{Name: "vc-astarte-invalid", Namespace: CustomAstarteNamespace},
				Spec: AstarteSpec{
					Version:           AstarteVersion,
					AstarteInstanceID: "coverageid2",
					API:               AstarteAPISpec{Host: "test.example.com"},
					VerneMQ:           AstarteVerneMQSpec{SSLListener: pointy.Bool(true), SSLListenerCertSecretName: ""},
					Cassandra: AstarteCassandraSpec{AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy: "SimpleStrategy",
						ReplicationFactor:   3,
					}},
				},
			}
			w, err := obj.ValidateCreate()
			Expect(w).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsInvalid(err)).To(BeTrue())
		})

		It("should return invalid when keyspace has invalid replication factor", func() {
			obj := &Astarte{
				ObjectMeta: metav1.ObjectMeta{Name: "vc-astarte-keyspace", Namespace: CustomAstarteNamespace},
				Spec: AstarteSpec{
					Version:           AstarteVersion,
					AstarteInstanceID: "coverageid3",
					API:               AstarteAPISpec{Host: "test.example.com"},
					VerneMQ:           AstarteVerneMQSpec{SSLListener: pointy.Bool(false)},
					Cassandra: AstarteCassandraSpec{AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
						ReplicationStrategy: "SimpleStrategy",
						ReplicationFactor:   2, // Even number - invalid
					}},
				},
			}
			w, err := obj.ValidateCreate()
			Expect(w).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsInvalid(err)).To(BeTrue())
		})
	})

	Describe("TestValidateUpdate", func() {
		It("should return invalid when astarteInstanceID changes", func() {
			oldObj := &Astarte{Spec: AstarteSpec{AstarteInstanceID: "old"}}
			newObj := &Astarte{Spec: AstarteSpec{AstarteInstanceID: "new"}}
			w, err := newObj.ValidateUpdate(oldObj)
			Expect(w).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsInvalid(err)).To(BeTrue())
		})

		It("should return invalid when keyspace changes", func() {
			oldObj := &Astarte{
				Spec: AstarteSpec{
					Cassandra: AstarteCassandraSpec{
						AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
							ReplicationStrategy: "SimpleStrategy",
							ReplicationFactor:   3,
						},
					},
				},
			}
			newObj := &Astarte{
				Spec: AstarteSpec{
					Cassandra: AstarteCassandraSpec{
						AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{
							ReplicationStrategy: "NetworkTopologyStrategy",
							ReplicationFactor:   1,
						},
					},
				},
			}
			w, err := newObj.ValidateUpdate(oldObj)
			Expect(w).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsInvalid(err)).To(BeTrue())
		})

		It("should succeed when no changes violate validations", func() {
			oldObj := &Astarte{Spec: AstarteSpec{AstarteInstanceID: "same", Cassandra: AstarteCassandraSpec{AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{}}}}
			newObj := &Astarte{Spec: AstarteSpec{AstarteInstanceID: "same", Cassandra: AstarteCassandraSpec{AstarteSystemKeyspace: AstarteSystemKeyspaceSpec{}}}}
			w, err := newObj.ValidateUpdate(oldObj)
			Expect(w).To(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TestValidateAstarte", func() {
		It("should aggregate multiple errors across validators", func() {
			obj := &Astarte{
				ObjectMeta: metav1.ObjectMeta{Name: "va-astarte", Namespace: CustomAstarteNamespace},
				Spec: AstarteSpec{
					VerneMQ: AstarteVerneMQSpec{
						SSLListener:               pointy.Bool(true),
						SSLListenerCertSecretName: "missing-secret",
						AstarteGenericClusteredResource: AstarteGenericClusteredResource{
							PodLabels: map[string]string{"app": "bad"},
						},
					},
					Features: AstarteFeatures{
						Autoscaling: true,
						AstartePodPriorities: &AstartePodPrioritiesSpec{
							Enable:              true,
							AstarteHighPriority: pointy.Int(500),
							AstarteMidPriority:  pointy.Int(600),
							AstarteLowPriority:  pointy.Int(700),
						},
					},
					Components: AstarteComponentsSpec{
						DataUpdaterPlant: AstarteDataUpdaterPlantSpec{AstarteGenericClusteredResource: AstarteGenericClusteredResource{Autoscale: &AstarteGenericClusteredResourceAutoscalerSpec{Horizontal: "hpa"}}},
					},
					CFSSL: AstarteCFSSLSpec{Deploy: pointy.Bool(false), URL: ""},
				},
			}
			errs := obj.validateAstarte()
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
