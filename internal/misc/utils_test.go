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
	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"
	"github.com/go-logr/logr"
	"go.openly.dev/pointy"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Misc utils testing", Ordered, Serial, func() {
	const (
		CustomSecretName       = "custom-secret"
		CustomUsernameKey      = "usr"
		CustomPasswordKey      = "pwd"
		CustomAstarteName      = "example-astarte"
		CustomAstarteNamespace = "utils-test"
		CustomRabbitMQHost     = "rabbitmq.example.com"
		CustomRabbitMQPort     = 5672
		CustomVerneMQHost      = "broker.astarte-example.com"
		CustomVerneMQPort      = 8883
	)

	var log logr.Logger
	var cr *v2alpha1.Astarte

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

	Describe("ReconcileConfigMap", func() {
		var cmData map[string]string
		var objName string

		BeforeEach(func() {
			cmData = map[string]string{
				"key1": "value1",
				"key2": "value2",
			}

			objName = "example-configmap"
		})

		AfterEach(func() {
			cm := &v1.ConfigMap{}
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, cm)
			if err == nil {
				Expect(k8sClient.Delete(context.Background(), cm)).To(Succeed())
			}
		})

		It("should create a ConfigMap", func() {
			_, err := ReconcileConfigMap(objName, cmData, cr, k8sClient, testEnv.Scheme, log)
			Expect(err).ToNot(HaveOccurred())

			createdCm := &v1.ConfigMap{}
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, createdCm)
			Expect(err).ToNot(HaveOccurred())
			Expect(createdCm.Data).To(Equal(cmData))
		})

		It("should update an existing ConfigMap", func() {
			// Create first
			_, err := ReconcileConfigMap(objName, cmData, cr, k8sClient, testEnv.Scheme, log)
			Expect(err).ToNot(HaveOccurred())

			// Update
			updated := map[string]string{"key1": "new", "key3": "added"}
			_, err = ReconcileConfigMap(objName, updated, cr, k8sClient, testEnv.Scheme, log)
			Expect(err).ToNot(HaveOccurred())

			cm := &v1.ConfigMap{}
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, cm)
			Expect(err).ToNot(HaveOccurred())
			Expect(cm.Data).To(Equal(updated))
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
		})

		AfterEach(func() {
			createdSecret := &v1.Secret{}
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, createdSecret)
			if err == nil {
				Expect(k8sClient.Delete(context.Background(), createdSecret)).To(Succeed())
			}
		})

		It("should create a TLS Secret", func() {
			_, err := ReconcileTLSSecret(objName, cert, key, cr, k8sClient, testEnv.Scheme, log)
			Expect(err).ToNot(HaveOccurred())

			createdSecret := &v1.Secret{}

			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, createdSecret)
			Expect(err).ToNot(HaveOccurred())
			Expect(createdSecret.Type).To(Equal(v1.SecretTypeTLS))
			Expect(string(createdSecret.Data[v1.TLSCertKey])).To(Equal(cert))
			Expect(string(createdSecret.Data[v1.TLSPrivateKeyKey])).To(Equal(key))
		})

		It("should update an existing TLS Secret", func() {
			_, err := ReconcileTLSSecret(objName, cert, key, cr, k8sClient, testEnv.Scheme, log)
			Expect(err).ToNot(HaveOccurred())

			newCert := "new-cert"
			newKey := "new-key"
			_, err = ReconcileTLSSecret(objName, newCert, newKey, cr, k8sClient, testEnv.Scheme, log)
			Expect(err).ToNot(HaveOccurred())

			s := &v1.Secret{}
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, s)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(s.Data[v1.TLSCertKey])).To(Equal(newCert))
			Expect(string(s.Data[v1.TLSPrivateKeyKey])).To(Equal(newKey))
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
		})

		AfterEach(func() {
			createdSecret := &v1.Secret{}
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, createdSecret)
			if err == nil {
				Expect(k8sClient.Delete(context.Background(), createdSecret)).To(Succeed())
			}
		})

		It("should create an Opaque Secret", func() {
			_, err := ReconcileSecret(objName, secretData, cr, k8sClient, testEnv.Scheme, log)
			Expect(err).ToNot(HaveOccurred())

			createdSecret := &v1.Secret{}

			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, createdSecret)
			Expect(err).ToNot(HaveOccurred())
			Expect(createdSecret.Data).To(Equal(secretData))
		})

		It("should update an existing Secret", func() {
			_, err := ReconcileSecret(objName, secretData, cr, k8sClient, testEnv.Scheme, log)
			Expect(err).ToNot(HaveOccurred())

			updated := map[string][]byte{"key1": []byte("new"), "k": []byte("v")}
			_, err = ReconcileSecret(objName, updated, cr, k8sClient, testEnv.Scheme, log)
			Expect(err).ToNot(HaveOccurred())

			s := &v1.Secret{}
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: objName, Namespace: CustomAstarteNamespace}, s)
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Data).To(Equal(updated))
		})
	})

	Describe("ReconcileSecretString", func() {
		It("should create a Secret using StringData", func() {
			name := "example-secret-string"
			data := map[string]string{"a": "1", "b": "2"}

			_, err := ReconcileSecretString(name, data, cr, k8sClient, testEnv.Scheme, log)
			Expect(err).ToNot(HaveOccurred())

			s := &v1.Secret{}
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: CustomAstarteNamespace}, s)
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Type).To(Equal(v1.SecretTypeOpaque))
			Expect(string(s.Data["a"])).To(Equal("1"))
			Expect(string(s.Data["b"])).To(Equal("2"))

			// cleanup
			Expect(k8sClient.Delete(context.Background(), s)).To(Succeed())
		})
	})

	Describe("ReconcileSecretStringWithLabels", func() {
		It("should create a Secret with labels and StringData", func() {
			name := "example-secret-string-labels"
			labels := map[string]string{"foo": "bar"}
			data := map[string]string{"x": "y"}

			_, err := ReconcileSecretStringWithLabels(name, data, labels, cr, k8sClient, testEnv.Scheme, log)
			Expect(err).ToNot(HaveOccurred())

			s := &v1.Secret{}
			err = k8sClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: CustomAstarteNamespace}, s)
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Labels).To(HaveKeyWithValue("foo", "bar"))
			Expect(string(s.Data["x"])).To(Equal("y"))

			// cleanup
			Expect(k8sClient.Delete(context.Background(), s)).To(Succeed())
		})
	})

	Describe("LogCreateOrUpdateOperationResult", func() {
		It("should not panic when logging results", func() {
			dummy := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "dummy", Namespace: CustomAstarteNamespace}}
			Expect(func() { LogCreateOrUpdateOperationResult(log, controllerutil.OperationResultNone, cr, dummy) }).ToNot(Panic())
			Expect(func() { LogCreateOrUpdateOperationResult(log, controllerutil.OperationResultCreated, cr, dummy) }).ToNot(Panic())
			Expect(func() { LogCreateOrUpdateOperationResult(log, controllerutil.OperationResultUpdated, cr, dummy) }).ToNot(Panic())
		})
	})

	Describe("GetVerneMQBrokerURL", func() {
		BeforeEach(func() {
			// Ensure VerneMQ configuration is set
			cr.Spec.VerneMQ.Host = CustomVerneMQHost
			cr.Spec.VerneMQ.Port = pointy.Int32(CustomVerneMQPort)
		})

		It("should return the correct VerneMQ broker URL", func() {
			url := GetVerneMQBrokerURL(cr)
			Expect(url).To(Equal("mqtts://" + CustomVerneMQHost + ":" + strconv.Itoa(CustomVerneMQPort)))
		})

	})

	Describe("GetResourcesForAstarteComponent", func() {
		It("should return requested resources when explicitly provided", func() {
			req := v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("500m"),
					v1.ResourceMemory: resource.MustParse("256Mi"),
				},
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("200m"),
					v1.ResourceMemory: resource.MustParse("128Mi"),
				},
			}
			got := GetResourcesForAstarteComponent(cr, &req, v2alpha1.AppEngineAPI)
			Expect(got).To(Equal(req))
		})

		It("should return empty resources when global resources are not set", func() {
			cr.Spec.Components.Resources = nil
			got := GetResourcesForAstarteComponent(cr, nil, v2alpha1.AppEngineAPI)
			Expect(got.Limits).To(BeNil())
			Expect(got.Requests).To(BeNil())
		})

		Context("with global resources set", func() {
			BeforeEach(func() {
				cr.Spec.Components.Resources = &v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("800m"),
						v1.ResourceMemory: resource.MustParse("800Mi"),
					},
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("400m"),
						v1.ResourceMemory: resource.MustParse("400Mi"),
					},
				}
			})

			It("should scale and adjust small requests (thresholds)", func() {
				got := GetResourcesForAstarteComponent(cr, nil, v2alpha1.AppEngineAPI)

				// Requests: CPU <150m -> 0, Memory <128Mi -> raised to 128M (decimal)
				Expect(got.Requests.Cpu().MilliValue()).To(Equal(int64(0)))
				Expect(got.Requests.Memory().Cmp(resource.MustParse("128M"))).To(Equal(0))

				// Limits: CPU bumped to >=300m, Memory scaled by coeff and must be >= requests
				Expect(got.Limits.Cpu().MilliValue()).To(Equal(int64(300)))
				Expect(got.Limits.Memory().Cmp(*got.Requests.Memory())).To(BeNumerically(">=", 0))
			})

			It("should scale normally when above thresholds", func() {
				// Use larger totals
				cr.Spec.Components.Resources = &v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("4000m"),
						v1.ResourceMemory: resource.MustParse("4000Mi"),
					},
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("2000m"),
						v1.ResourceMemory: resource.MustParse("2000Mi"),
					},
				}
				got := GetResourcesForAstarteComponent(cr, nil, v2alpha1.AppEngineAPI)

				// Requests scaled: 0.18 * 2000m = 360m; Memory around 360-380Mi depending on decimal vs binary
				Expect(got.Requests.Cpu().MilliValue()).To(Equal(int64(360)))
				Expect(got.Requests.Memory().Cmp(resource.MustParse("300Mi"))).To(BeNumerically(">", 0))
				Expect(got.Requests.Memory().Cmp(resource.MustParse("400Mi"))).To(BeNumerically("<", 0))

				// Limits scaled: 0.18 * 4000m = 720m; Memory > 600Mi
				Expect(got.Limits.Cpu().MilliValue()).To(Equal(int64(720)))
				Expect(got.Limits.Memory().Cmp(resource.MustParse("600Mi"))).To(BeNumerically(">", 0))
			})
		})
	})

	Describe("getAllocationScaledQuantity", func() {
		It("should scale a quantity by coefficient and scale", func() {
			base := resource.NewScaledQuantity(1000, resource.Milli)
			got := getAllocationScaledQuantity(base, resource.Milli, 0.5)
			Expect(got.MilliValue()).To(Equal(int64(500)))
		})
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
			It("should return the number of deployed components as float", func() {
				// Disable all but one
				cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
				cr.Spec.Components.Housekeeping.Deploy = pointy.Bool(false)
				cr.Spec.Components.Dashboard.Deploy = pointy.Bool(false)
				cr.Spec.Components.DataUpdaterPlant.Deploy = pointy.Bool(false)
				cr.Spec.Components.Flow.Deploy = pointy.Bool(false)
				cr.Spec.Components.Pairing.Deploy = pointy.Bool(false)
				cr.Spec.Components.RealmManagement.Deploy = pointy.Bool(false)
				cr.Spec.Components.TriggerEngine.Deploy = pointy.Bool(false)
				Expect(getNumberOfDeployedAstarteComponentsAsFloat(cr)).To(Equal(1.0))
			})
		})

		Context("When two components are deployed", func() {
			It("should return the number of deployed components as float", func() {
				// Disable all but two
				cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
				cr.Spec.Components.Housekeeping.Deploy = pointy.Bool(true)
				cr.Spec.Components.Dashboard.Deploy = pointy.Bool(false)
				cr.Spec.Components.DataUpdaterPlant.Deploy = pointy.Bool(false)
				cr.Spec.Components.Flow.Deploy = pointy.Bool(false)
				cr.Spec.Components.Pairing.Deploy = pointy.Bool(false)
				cr.Spec.Components.RealmManagement.Deploy = pointy.Bool(false)
				cr.Spec.Components.TriggerEngine.Deploy = pointy.Bool(false)
				Expect(getNumberOfDeployedAstarteComponentsAsFloat(cr)).To(Equal(2.0))
			})
		})

		Context("When all components are deployed", func() {
			It("should return the number of deployed components as float", func() {
				cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
				cr.Spec.Components.Housekeeping.Deploy = pointy.Bool(true)
				cr.Spec.Components.Dashboard.Deploy = pointy.Bool(true)
				cr.Spec.Components.DataUpdaterPlant.Deploy = pointy.Bool(true)
				cr.Spec.Components.Flow.Deploy = pointy.Bool(true)
				cr.Spec.Components.Pairing.Deploy = pointy.Bool(true)
				cr.Spec.Components.RealmManagement.Deploy = pointy.Bool(true)
				cr.Spec.Components.TriggerEngine.Deploy = pointy.Bool(true)
				Expect(getNumberOfDeployedAstarteComponentsAsFloat(cr)).To(Equal(8.0))
			})
		})
	})

	Describe("getLeftoverCoefficients", func() {
		BeforeEach(func() {
			// Ensure defaults (all deployed) then disable a subset
			cr.Spec.Components = v2alpha1.AstarteComponentsSpec{}
			cr.Spec.Components.Dashboard.Deploy = pointy.Bool(false)
			cr.Spec.Components.Flow.Deploy = pointy.Bool(false)
		})

		It("should sum coefficients for not deployed components", func() {
			leftovers := getLeftoverCoefficients(cr)
			expectedCPU := defaultComponentAllocations[v2alpha1.Dashboard].CPUCoefficient +
				defaultComponentAllocations[v2alpha1.FlowComponent].CPUCoefficient
			expectedMem := defaultComponentAllocations[v2alpha1.Dashboard].MemoryCoefficient +
				defaultComponentAllocations[v2alpha1.FlowComponent].MemoryCoefficient
			Expect(leftovers.CPUCoefficient).To(BeNumerically("~", expectedCPU, 1e-9))
			Expect(leftovers.MemoryCoefficient).To(BeNumerically("~", expectedMem, 1e-9))
		})
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
				Expect(checkComponentForLeftoverAllocations(v2alpha1.AstarteGenericClusteredResource{}, v2alpha1.AstarteComponent("UnknownComponent"), aC)).To(Equal(aC))
			})
		})

	})

	Describe("getWeightedDefaultAllocationFor", func() {
		It("should return default allocation when no leftovers", func() {
			// Force all components to be deployed
			cr.Spec.Components.Flow.Deploy = pointy.Bool(true)
			cr.Spec.Components.Dashboard.Deploy = pointy.Bool(true)
			cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
			cr.Spec.Components.DataUpdaterPlant.Deploy = pointy.Bool(true)
			cr.Spec.Components.Housekeeping.Deploy = pointy.Bool(true)
			cr.Spec.Components.Pairing.Deploy = pointy.Bool(true)
			cr.Spec.Components.RealmManagement.Deploy = pointy.Bool(true)
			cr.Spec.Components.TriggerEngine.Deploy = pointy.Bool(true)

			ac := getWeightedDefaultAllocationFor(cr, v2alpha1.AppEngineAPI)
			def := defaultComponentAllocations[v2alpha1.AppEngineAPI]
			Expect(ac.CPUCoefficient).To(BeNumerically("~", def.CPUCoefficient, 1e-9))
			Expect(ac.MemoryCoefficient).To(BeNumerically("~", def.MemoryCoefficient, 1e-9))
		})

		It("should distribute leftovers among deployed components", func() {
			// Disable two components -> leftovers redistributed among 6 deployed
			cr.Spec.Components = v2alpha1.AstarteComponentsSpec{}
			cr.Spec.Components.Dashboard.Deploy = pointy.Bool(false)
			cr.Spec.Components.Flow.Deploy = pointy.Bool(false)

			ac := getWeightedDefaultAllocationFor(cr, v2alpha1.AppEngineAPI)
			leftoversCPU := defaultComponentAllocations[v2alpha1.Dashboard].CPUCoefficient + defaultComponentAllocations[v2alpha1.FlowComponent].CPUCoefficient
			leftoversMem := defaultComponentAllocations[v2alpha1.Dashboard].MemoryCoefficient + defaultComponentAllocations[v2alpha1.FlowComponent].MemoryCoefficient
			deployed := 6.0
			Expect(ac.CPUCoefficient).To(BeNumerically("~", defaultComponentAllocations[v2alpha1.AppEngineAPI].CPUCoefficient+(leftoversCPU/deployed), 1e-9))
			Expect(ac.MemoryCoefficient).To(BeNumerically("~", defaultComponentAllocations[v2alpha1.AppEngineAPI].MemoryCoefficient+(leftoversMem/deployed), 1e-9))
		})
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
				Expect(IsAstarteComponentDeployed(cr, v2alpha1.AstarteComponent("UnknownComponent"))).To(BeFalse())
			})
		})
	})

	// Test GetRabbitMQHostnameAndPort
	Describe("GetRabbitMQHostnameAndPort", func() {
		BeforeEach(func() {
			// Initialize RabbitMQ.Connection if it's nil
			if cr.Spec.RabbitMQ.Connection == nil {
				cr.Spec.RabbitMQ.Connection = &v2alpha1.AstarteRabbitMQConnectionSpec{}
			}
			cr.Spec.RabbitMQ.Connection.Host = CustomRabbitMQHost
			cr.Spec.RabbitMQ.Connection.Port = pointy.Int32(CustomRabbitMQPort)
		})

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

			// Test GetRabbitMQCredentialsFor
			Describe("GetRabbitMQCredentialsFor", func() {
				BeforeEach(func() {
					// Ensure RabbitMQ connection is configured to avoid nil dereference
					cr.Spec.RabbitMQ = v2alpha1.AstarteRabbitMQSpec{
						Connection: &v2alpha1.AstarteRabbitMQConnectionSpec{
							HostAndPort: v2alpha1.HostAndPort{
								Host: CustomRabbitMQHost,
								Port: pointy.Int32(CustomRabbitMQPort),
							},
						},
					}
				})
				It("should retrieve host, port and credentials from secret", func() {
					// Prepare secret with default naming and keys
					sec := &v1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      CustomAstarteName + "-rabbitmq-user-credentials",
							Namespace: CustomAstarteNamespace,
						},
						Data: map[string][]byte{
							RabbitMQDefaultUserCredentialsUsernameKey: []byte("user"),
							RabbitMQDefaultUserCredentialsPasswordKey: []byte("pass"),
						},
					}
					Expect(k8sClient.Create(context.Background(), sec)).To(Succeed())
					defer func() { _ = k8sClient.Delete(context.Background(), sec) }()

					host, port, user, pass, err := GetRabbitMQCredentialsFor(cr, k8sClient)
					Expect(err).ToNot(HaveOccurred())
					Expect(host).To(Equal(CustomRabbitMQHost))
					Expect(port).To(Equal(int32(CustomRabbitMQPort)))
					Expect(user).To(Equal("user"))
					Expect(pass).To(Equal("pass"))
				})
			})
		})
	})
})
