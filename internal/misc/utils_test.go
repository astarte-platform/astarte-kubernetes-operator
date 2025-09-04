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

package misc

import (
	"context"
	"strconv"

	"github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	"github.com/go-logr/logr"
	"go.openly.dev/pointy"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	var cr *v2alpha1.Astarte
	var log logr.Logger
	var reqLogger logr.Logger

	BeforeAll(func() {
		log = reqLogger.WithValues("test", "misc-utils")
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
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: CustomAstarteNamespace,
				},
			}
			Expect(k8sClient.Delete(context.Background(), ns)).To(Succeed())

			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteNamespace}, &v1.Namespace{})
			}, "10s", "250ms").ShouldNot(Succeed())
		}
	})

	BeforeEach(func() {
		// Create and initialize a basic Astarte CR
		cr = &v2alpha1.Astarte{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CustomAstarteName,
				Namespace: CustomAstarteNamespace,
			},
			Spec: v2alpha1.AstarteSpec{
				Version: AstarteVersion,
				RabbitMQ: v2alpha1.AstarteRabbitMQSpec{
					Connection: &v2alpha1.AstarteRabbitMQConnectionSpec{
						HostAndPort: v2alpha1.HostAndPort{
							Host: CustomRabbitMQHost,
							Port: pointy.Int32(CustomRabbitMQPort),
						},
					},
				},
				VerneMQ: v2alpha1.AstarteVerneMQSpec{
					HostAndPort: v2alpha1.HostAndPort{
						Host: CustomVerneMQHost,
						Port: pointy.Int32(CustomVerneMQPort),
					},
				},
			},
		}

		// Create the Astarte CR in the fake client
		Expect(k8sClient.Create(context.Background(), cr)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.Background(), cr)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
		}, "10s", "250ms").ShouldNot(Succeed())
	})

	Describe("ReconcileConfigMap", func() {
		var cmData map[string]string
		var objName string

		BeforeEach(func() {
			cmData = map[string]string{
				"key1": "value1",
				"key2": "value2",
			}

			objName = "example-configmap"
			reqLogger = log.WithValues("test", "ReconcileConfigMap")
		})

		AfterEach(func() {
			cm := &v1.ConfigMap{}
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, cm)
			if err == nil {
				Expect(k8sClient.Delete(context.Background(), cm)).To(Succeed())
			}
		})

		It("should create a ConfigMap", func() {
			_, err := ReconcileConfigMap(objName, cmData, cr, k8sClient, testEnv.Scheme, reqLogger)
			Expect(err).NotTo(HaveOccurred())

			createdCm := &v1.ConfigMap{}
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, createdCm)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdCm.Data).To(Equal(cmData))
		})

	})

	Describe("ReconcileTLSSecret", func() {
		var cert string
		var key string
		var objName string

		BeforeEach(func() {
			cert = "cert-data"
			key = "key-data"
			objName = "example-cert-secret"
			reqLogger = log.WithValues("test", "ReconcileTLSSecret")
		})

		AfterEach(func() {
			createdSecret := &v1.Secret{}
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, createdSecret)
			if err == nil {
				Expect(k8sClient.Delete(context.Background(), createdSecret)).To(Succeed())
			}
		})

		It("should create a TLS Secret", func() {
			_, err := ReconcileTLSSecret(objName, cert, key, cr, k8sClient, testEnv.Scheme, reqLogger)
			Expect(err).NotTo(HaveOccurred())

			createdSecret := &v1.Secret{}

			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, createdSecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdSecret.Type).To(Equal(v1.SecretTypeTLS))
			Expect(string(createdSecret.Data[v1.TLSCertKey])).To(Equal(cert))
			Expect(string(createdSecret.Data[v1.TLSPrivateKeyKey])).To(Equal(key))
		})
	})

	Describe("ReconcileSecret", func() {
		var objName string
		var secretData map[string][]byte

		BeforeEach(func() {
			secretData = map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			}

			objName = "example-secret"
			reqLogger = log.WithValues("test", "ReconcileSecret")
		})

		AfterEach(func() {
			createdSecret := &v1.Secret{}
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, createdSecret)
			if err == nil {
				Expect(k8sClient.Delete(context.Background(), createdSecret)).To(Succeed())
			}
		})

		It("should create a TLS Secret", func() {
			_, err := ReconcileSecret(objName, secretData, cr, k8sClient, testEnv.Scheme, reqLogger)
			Expect(err).NotTo(HaveOccurred())

			createdSecret := &v1.Secret{}

			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, createdSecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdSecret.Data).To(Equal(secretData))
		})
	})

	Describe("ReconcileSecretString", func() {

	})

	Describe("ReconcileSecretStringWithLabels", func() {

	})

	Describe("LogCreateOrUpdateOperationResult", func() {

	})

	Describe("GetVerneMQBrokerURL", func() {
		It("should return the correct VerneMQ broker URL", func() {
			url := GetVerneMQBrokerURL(cr)
			Expect(url).To(Equal("mqtts://" + CustomVerneMQHost + ":" + strconv.Itoa(CustomVerneMQPort)))
		})

	})

	Describe("GetResourcesForAstarteComponent", func() {

	})

	Describe("getAllocationScaledQuantity", func() {

	})

	Describe("getNumberOfDeployedAstarteComponentsAsFloat", func() {
		BeforeEach(func() {
			cr.Spec = v2alpha1.AstarteSpec{
				Components: v2alpha1.AstarteComponentsSpec{
					AppengineAPI: v2alpha1.AstarteAppengineAPISpec{
						AstarteGenericAPIComponentSpec: v2alpha1.AstarteGenericAPIComponentSpec{
							AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
								Deploy: pointy.Bool(false),
							},
						},
					},
					Dashboard: v2alpha1.AstarteDashboardSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
					DataUpdaterPlant: v2alpha1.AstarteDataUpdaterPlantSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
					Flow: v2alpha1.AstarteGenericAPIComponentSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
					Housekeeping: v2alpha1.AstarteGenericAPIComponentSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
					Pairing: v2alpha1.AstarteGenericAPIComponentSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
					RealmManagement: v2alpha1.AstarteGenericAPIComponentSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
					TriggerEngine: v2alpha1.AstarteTriggerEngineSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
				},
			}
		})

		Context("When there are not deployed services", func() {
			It("should return the number of deployed components as float", func() {
				Expect(getNumberOfDeployedAstarteComponentsAsFloat(cr)).To(Equal(0.0))
			})
		})

		Context("When only one component is deployed", func() {
			BeforeEach(func() {
				cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
			})
			It("should return the number of deployed components as float", func() {
				Expect(getNumberOfDeployedAstarteComponentsAsFloat(cr)).To(Equal(1.0))
			})
		})

		Context("When two components are deployed", func() {
			BeforeEach(func() {
				cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
				cr.Spec.Components.Housekeeping.Deploy = pointy.Bool(true)
			})
			It("should return the number of deployed components as float", func() {
				Expect(getNumberOfDeployedAstarteComponentsAsFloat(cr)).To(Equal(2.0))
			})
		})

		Context("When all components are deployed", func() {
			BeforeEach(func() {
				cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
				cr.Spec.Components.Housekeeping.Deploy = pointy.Bool(true)
				cr.Spec.Components.Dashboard.Deploy = pointy.Bool(true)
				cr.Spec.Components.DataUpdaterPlant.Deploy = pointy.Bool(true)
				cr.Spec.Components.Flow.Deploy = pointy.Bool(true)
				cr.Spec.Components.Pairing.Deploy = pointy.Bool(true)
				cr.Spec.Components.RealmManagement.Deploy = pointy.Bool(true)
				cr.Spec.Components.TriggerEngine.Deploy = pointy.Bool(true)
			})
			It("should return the number of deployed components as float", func() {
				Expect(getNumberOfDeployedAstarteComponentsAsFloat(cr)).To(Equal(8.0))
			})
		})
	})

	Describe("getLeftoverCoefficients", func() {

	})

	Describe("checkComponentForLeftoverAllocations", func() {
		Context("When the component is AppengineAPI", func() {
			Context("When the component is deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
				})
				It("should return the allocationCoeffients passed", func() {
					aC := allocationCoefficients{
						CPUCoefficient:    0.5,
						MemoryCoefficient: 0.5,
					}
					Expect(checkComponentForLeftoverAllocations(cr.Spec.Components.AppengineAPI.AstarteGenericClusteredResource, v2alpha1.AppEngineAPI, aC)).To(Equal(aC))
				})
			})
			Context("When the component is not deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(false)
				})

				It("should return default allocation coefficients for that component", func() {
					defAc := allocationCoefficients{
						CPUCoefficient:    defaultComponentAllocations[v2alpha1.AppEngineAPI].CPUCoefficient,
						MemoryCoefficient: defaultComponentAllocations[v2alpha1.AppEngineAPI].MemoryCoefficient,
					}

					Expect(checkComponentForLeftoverAllocations(cr.Spec.Components.AppengineAPI.AstarteGenericClusteredResource, v2alpha1.AppEngineAPI, allocationCoefficients{})).To(Equal(defAc))

				})
			})
		})
		Context("When the component is Housekeeping", func() {
			Context("When the component is deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.Housekeeping.Deploy = pointy.Bool(true)
				})
				It("should return the allocationCoeffients passed", func() {
					aC := allocationCoefficients{
						CPUCoefficient:    0.5,
						MemoryCoefficient: 0.5,
					}
					Expect(checkComponentForLeftoverAllocations(cr.Spec.Components.Housekeeping.AstarteGenericClusteredResource, v2alpha1.Housekeeping, aC)).To(Equal(aC))
				})
			})
			Context("When the component is not deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.Housekeeping.Deploy = pointy.Bool(false)
				})

				It("should return default allocation coefficients for that component", func() {
					defAc := allocationCoefficients{
						CPUCoefficient:    defaultComponentAllocations[v2alpha1.Housekeeping].CPUCoefficient,
						MemoryCoefficient: defaultComponentAllocations[v2alpha1.Housekeeping].MemoryCoefficient,
					}

					Expect(checkComponentForLeftoverAllocations(cr.Spec.Components.Housekeeping.AstarteGenericClusteredResource, v2alpha1.Housekeeping, allocationCoefficients{})).To(Equal(defAc))

				})
			})

			// Omit other components for brevity
		})
		Context("When the component is unknown", func() {
			It("should return the allocationCoeffients passed", func() {
				aC := allocationCoefficients{
					CPUCoefficient:    0.5,
					MemoryCoefficient: 0.5,
				}
				Expect(checkComponentForLeftoverAllocations(v2alpha1.AstarteGenericClusteredResource{}, "UnknownComponent", aC)).To(Equal(aC))
			})
		})

	})

	Describe("getWeightedDefaultAllocationFor", func() {

	})

	// Test IsAstarteComponentDeployed
	Describe("IsAstarteComponentDeployed", func() {
		BeforeEach(func() {
			cr.Spec = v2alpha1.AstarteSpec{
				Components: v2alpha1.AstarteComponentsSpec{
					AppengineAPI: v2alpha1.AstarteAppengineAPISpec{
						AstarteGenericAPIComponentSpec: v2alpha1.AstarteGenericAPIComponentSpec{
							AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
								Deploy: pointy.Bool(false),
							},
						},
					},
					Dashboard: v2alpha1.AstarteDashboardSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
					DataUpdaterPlant: v2alpha1.AstarteDataUpdaterPlantSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
					Flow: v2alpha1.AstarteGenericAPIComponentSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
					Housekeeping: v2alpha1.AstarteGenericAPIComponentSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
					Pairing: v2alpha1.AstarteGenericAPIComponentSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
					RealmManagement: v2alpha1.AstarteGenericAPIComponentSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
					TriggerEngine: v2alpha1.AstarteTriggerEngineSpec{
						AstarteGenericClusteredResource: v2alpha1.AstarteGenericClusteredResource{
							Deploy: pointy.Bool(false),
						},
					},
				},
			}
		})

		Context("When the component is AppengineAPI", func() {
			Context("When the component is deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
				})
				It("should return true", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.AppEngineAPI)).To(BeTrue())
				})
			})
			Context("When the component is not deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(false)
				})
				It("should return false", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.AppEngineAPI)).To(BeFalse())
				})
			})
		})
		Context("When the component is Housekeeping", func() {
			Context("When the component is deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.Housekeeping.Deploy = pointy.Bool(true)
				})
				It("should return true", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.Housekeeping)).To(BeTrue())
				})
			})
			Context("When the component is not deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.Housekeeping.Deploy = pointy.Bool(false)
				})
				It("should return false", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.Housekeeping)).To(BeFalse())
				})
			})
		})
		Context("When the component is Dashboard", func() {
			Context("When the component is deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.Dashboard.Deploy = pointy.Bool(true)
				})
				It("should return true", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.Dashboard)).To(BeTrue())
				})
			})
			Context("When the component is not deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.Dashboard.Deploy = pointy.Bool(false)
				})
				It("should return false", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.Dashboard)).To(BeFalse())
				})
			})
		})
		Context("When the component is DataUpdaterPlant", func() {
			Context("When the component is deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.DataUpdaterPlant.Deploy = pointy.Bool(true)
				})
				It("should return true", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.DataUpdaterPlant)).To(BeTrue())
				})
			})
			Context("When the component is not deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.DataUpdaterPlant.Deploy = pointy.Bool(false)
				})
				It("should return false", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.DataUpdaterPlant)).To(BeFalse())
				})
			})
		})
		Context("When the component is Flow", func() {
			Context("When the component is deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.Flow.Deploy = pointy.Bool(true)
				})
				It("should return true", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.FlowComponent)).To(BeTrue())
				})
			})
			Context("When the component is not deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.Flow.Deploy = pointy.Bool(false)
				})
				It("should return false", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.FlowComponent)).To(BeFalse())
				})
			})
		})
		Context("When the component is Pairing", func() {
			Context("When the component is deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.Pairing.Deploy = pointy.Bool(true)
				})
				It("should return true", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.Pairing)).To(BeTrue())
				})
			})
			Context("When the component is not deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.Pairing.Deploy = pointy.Bool(false)
				})
				It("should return false", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.Pairing)).To(BeFalse())
				})
			})
		})
		Context("When the component is RealmManagement", func() {
			Context("When the component is deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.RealmManagement.Deploy = pointy.Bool(true)
				})
				It("should return true", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.RealmManagement)).To(BeTrue())
				})
			})
			Context("When the component is not deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.RealmManagement.Deploy = pointy.Bool(false)
				})
				It("should return false", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.RealmManagement)).To(BeFalse())
				})
			})
		})
		Context("When the component is TriggerEngine", func() {
			Context("When the component is deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.TriggerEngine.Deploy = pointy.Bool(true)
				})
				It("should return true", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.TriggerEngine)).To(BeTrue())
				})
			})
			Context("When the component is not deployed", func() {
				BeforeEach(func() {
					cr.Spec.Components.TriggerEngine.Deploy = pointy.Bool(false)
				})
				It("should return false", func() {
					Expect(IsAstarteComponentDeployed(cr, v2alpha1.TriggerEngine)).To(BeFalse())
				})
			})
		})
		Context("When the component is unknown", func() {
			It("should return false", func() {
				Expect(IsAstarteComponentDeployed(cr, "UnknownComponent")).To(BeFalse())
			})
		})
	})

	// Test GetRabbitMQHostnameAndPort
	Describe("GetRabbitMQHostnameAndPort", func() {
		Context("When retrieving RabbitMQ host and port", func() {
			It("should return the custom host and port", func() {
				host, port := GetRabbitMQHostnameAndPort(cr)
				Expect(host).To(Equal(CustomRabbitMQHost))
				Expect(port).To(Equal(int32(CustomRabbitMQPort)))
			})
		})
	})

	// Test GetRabbitMQUserCredentialsSecret
	Describe("GetRabbitMQUserCredentialsSecret", func() {
		Context("When retrieving RabbitMQ credentials from secret", func() {
			BeforeEach(func() {
				cr.Spec = v2alpha1.AstarteSpec{
					RabbitMQ: v2alpha1.AstarteRabbitMQSpec{
						Connection: &v2alpha1.AstarteRabbitMQConnectionSpec{},
					},
				}
			})
			Context("CredentialsSecret is nil", func() {
				It("should return the default secret info (SecretName, UsernameKey, PasswordKey)", func() {
					secretName, usernameKey, passwordKey := GetRabbitMQUserCredentialsSecret(cr)
					Expect(secretName).To(Equal(CustomAstarteName + "-rabbitmq-user-credentials"))
					Expect(usernameKey).To(Equal(RabbitMQDefaultUserCredentialsUsernameKey))
					Expect(passwordKey).To(Equal(RabbitMQDefaultUserCredentialsPasswordKey))
				})
			})
			Context("CredentialsSecret is set", func() {
				BeforeEach(func() {
					cr.Spec.RabbitMQ.Connection.CredentialsSecret = &v2alpha1.LoginCredentialsSecret{
						Name:        CustomSecretName,
						UsernameKey: CustomUsernameKey,
						PasswordKey: CustomPasswordKey,
					}
				})

				It("should return the custom secret info (SecretName, UsernameKey, PasswordKey)", func() {
					secretName, usernameKey, passwordKey := GetRabbitMQUserCredentialsSecret(cr)
					Expect(secretName).To(Equal(CustomSecretName))
					Expect(usernameKey).To(Equal(CustomUsernameKey))
					Expect(passwordKey).To(Equal(CustomPasswordKey))
				})
			})
		})
	})

	// Test GetCassandraUserCredentialsSecret
	Describe("GetCassandraUserCredentialsSecret", func() {
		Context("When retrieving Cassandra credentials from secret", func() {
			BeforeEach(func() {
				cr.Spec = v2alpha1.AstarteSpec{
					Cassandra: v2alpha1.AstarteCassandraSpec{
						Connection: &v2alpha1.AstarteCassandraConnectionSpec{},
					},
				}
			})
			Context("CredentialsSecret is nil", func() {
				It("should return the default secret info (SecretName, UsernameKey, PasswordKey)", func() {
					secretName, usernameKey, passwordKey := GetCassandraUserCredentialsSecret(cr)
					Expect(secretName).To(Equal(CustomAstarteName + "-cassandra-user-credentials"))
					Expect(usernameKey).To(Equal(CassandraDefaultUserCredentialsUsernameKey))
					Expect(passwordKey).To(Equal(CassandraDefaultUserCredentialsPasswordKey))
				})
			})
			Context("CredentialsSecret is set", func() {
				BeforeEach(func() {
					cr.Spec.Cassandra.Connection.CredentialsSecret = &v2alpha1.LoginCredentialsSecret{
						Name:        CustomSecretName,
						UsernameKey: CustomUsernameKey,
						PasswordKey: CustomPasswordKey,
					}
				})

				It("should return the custom secret info (SecretName, UsernameKey, PasswordKey)", func() {
					secretName, usernameKey, passwordKey := GetCassandraUserCredentialsSecret(cr)
					Expect(secretName).To(Equal(CustomSecretName))
					Expect(usernameKey).To(Equal(CustomUsernameKey))
					Expect(passwordKey).To(Equal(CustomPasswordKey))
				})
			})
		})
	})
})
