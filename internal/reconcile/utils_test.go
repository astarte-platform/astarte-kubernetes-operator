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
	"crypto/rsa"
	"crypto/x509"
	"encoding/base32"
	"encoding/pem"

	"go.openly.dev/pointy"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	builder "github.com/astarte-platform/astarte-kubernetes-operator/test/builder"
	"github.com/astarte-platform/astarte-kubernetes-operator/test/integrationutils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
)

var _ = Describe("Utils functions testing", Ordered, Serial, func() {
	const (
		CustomAstarteName      = "test-astarte-utils"
		CustomAstarteNamespace = "astarte-utils-test"
	)

	var cr *apiv2alpha1.Astarte
	var b *builder.TestAstarteBuilder

	BeforeAll(func() {
		integrationutils.CreateNamespace(k8sClient, CustomAstarteNamespace)
	})

	AfterAll(func() {
		integrationutils.TeardownNamespace(k8sClient, CustomAstarteNamespace)
	})

	BeforeEach(func() {
		b = builder.NewTestAstarteBuilder(CustomAstarteName, CustomAstarteNamespace)
		cr = b.Build()
		integrationutils.DeployAstarte(k8sClient, CustomAstarteName, CustomAstarteNamespace, cr)
	})

	AfterEach(func() {
		integrationutils.TeardownResources(context.Background(), k8sClient, CustomAstarteNamespace)
	})

	Describe("Test getAstarteCommonEnvVars", func() {
		// The function just returns a predefined env var slice; no complex logic to test.
	})

	Describe("Test appendCassandraConnectionEnvVars", func() {
		// The function just appends predefined env vars based on input; no complex logic to test.
	})

	Describe("Test appendAstarteKeyspaceEnvVars", func() {
		// The function just appends predefined env vars based on input; no complex logic to test.
	})

	Describe("Test appendRabbitMQConnectionEnvVars", func() {
		// The function just appends predefined env vars based on input; no complex logic to test.
	})

	Describe("Test getStandardAntiAffinityForAppLabel", func() {
		// The function just returns a predefined affinity object; no complex logic to test.
	})

	Describe("Test encodePEMBlockToEncodedBytes", func() {
		It("should correctly encode PEM block to string", func() {
			testData := []byte("test data")
			block := &pem.Block{
				Type:  "TEST BLOCK",
				Bytes: testData,
			}

			result := encodePEMBlockToEncodedBytes(block)

			Expect(result).To(ContainSubstring("BEGIN TEST BLOCK"))
			Expect(result).To(ContainSubstring("END TEST BLOCK"))

			// Verify it can be decoded back
			decodedBlock, _ := pem.Decode([]byte(result))
			Expect(decodedBlock).ToNot(BeNil())
			Expect(decodedBlock.Type).To(Equal("TEST BLOCK"))
			Expect(decodedBlock.Bytes).To(Equal(testData))
		})
	})

	Describe("Test generateKeyPair", func() {
		It("should generate a valid RSA key pair", func() {
			privateKey, err := generateKeyPair()
			Expect(err).ToNot(HaveOccurred())
			Expect(privateKey).ToNot(BeNil())

			// Check key size is reasonable (should be 4096 bits = 512 bytes, but be flexible)
			keySize := privateKey.Size()
			Expect(keySize).To(BeNumerically(">=", 256))  // At least 2048 bits
			Expect(keySize).To(BeNumerically("<=", 1024)) // At most 8192 bits

			// Verify the public key can be extracted
			publicKey := &privateKey.PublicKey
			Expect(publicKey).ToNot(BeNil())
			Expect(publicKey.N).ToNot(BeNil())

			// Verify key strength
			Expect(publicKey.E).To(Equal(65537)) // Common RSA exponent
		})
	})

	Describe("Test storePublicKeyInSecret", func() {
		It("should store public key in secret correctly", func() {
			privateKey, err := generateKeyPair()
			Expect(err).ToNot(HaveOccurred())

			secretName := "test-public-key"
			err = storePublicKeyInSecret(secretName, &privateKey.PublicKey, cr, k8sClient, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			// Verify secret was created
			secret := &v1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      secretName,
					Namespace: CustomAstarteNamespace,
				}, secret)
			}, Timeout, Interval).Should(Succeed())

			// Verify secret content
			Expect(secret.Data).To(HaveKey("public-key"))
			publicKeyPEM := string(secret.Data["public-key"])
			Expect(publicKeyPEM).To(ContainSubstring("BEGIN PUBLIC KEY"))
			Expect(publicKeyPEM).To(ContainSubstring("END PUBLIC KEY"))

			// Verify the stored key can be parsed
			block, _ := pem.Decode([]byte(publicKeyPEM))
			Expect(block).ToNot(BeNil())
			Expect(block.Type).To(Equal("PUBLIC KEY"))

			parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedKey).To(BeAssignableToTypeOf(&rsa.PublicKey{}))
		})
	})

	Describe("Test storePrivateKeyInSecret", func() {
		It("should store private key in secret correctly", func() {
			privateKey, err := generateKeyPair()
			Expect(err).ToNot(HaveOccurred())

			secretName := "test-private-key"
			err = storePrivateKeyInSecret(secretName, privateKey, cr, k8sClient, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			// Verify secret was created
			secret := &v1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      secretName,
					Namespace: CustomAstarteNamespace,
				}, secret)
			}, Timeout, Interval).Should(Succeed())

			// Verify secret content
			Expect(secret.Data).To(HaveKey("private-key"))
			privateKeyPEM := string(secret.Data["private-key"])
			Expect(privateKeyPEM).To(ContainSubstring("BEGIN RSA PRIVATE KEY"))
			Expect(privateKeyPEM).To(ContainSubstring("END RSA PRIVATE KEY"))

			// Verify the stored key can be parsed
			block, _ := pem.Decode([]byte(privateKeyPEM))
			Expect(block).ToNot(BeNil())
			Expect(block.Type).To(Equal("RSA PRIVATE KEY"))

			parsedKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedKey).ToNot(BeNil())
		})
	})

	Describe("Test reconcileStandardRBACForClusteringForApp", func() {
		It("should create ServiceAccount, Role, and RoleBinding", func() {
			name := "test-rbac"
			policyRules := []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get", "list", "watch"},
				},
			}

			err := reconcileStandardRBACForClusteringForApp(name, policyRules, cr, k8sClient, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			// Verify ServiceAccount
			sa := &v1.ServiceAccount{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      name,
					Namespace: CustomAstarteNamespace,
				}, sa)
			}, Timeout, Interval).Should(Succeed())

			// Verify Role
			role := &rbacv1.Role{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      name,
					Namespace: CustomAstarteNamespace,
				}, role)
			}, Timeout, Interval).Should(Succeed())

			Expect(role.Rules).To(Equal(policyRules))

			// Verify RoleBinding
			rb := &rbacv1.RoleBinding{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      name,
					Namespace: CustomAstarteNamespace,
				}, rb)
			}, Timeout, Interval).Should(Succeed())

			Expect(rb.Subjects).To(HaveLen(1))
			Expect(rb.Subjects[0].Kind).To(Equal("ServiceAccount"))
			Expect(rb.Subjects[0].Name).To(Equal(name))
			Expect(rb.RoleRef.Kind).To(Equal("Role"))
			Expect(rb.RoleRef.Name).To(Equal(name))
			Expect(rb.RoleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))
		})
	})

	Describe("Test reconcileRBACForFlow", func() {
		// Tests regarding Flow are not implemented at the moment.
	})

	Describe("Test ensureErlangCookieSecret", func() {
		It("should create new cookie secret if it doesn't exist", func() {
			secretName := "test-erlang-cookie"

			err := ensureErlangCookieSecret(secretName, cr, k8sClient, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			// Verify secret was created
			secret := &v1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      secretName,
					Namespace: CustomAstarteNamespace,
				}, secret)
			}, Timeout, Interval).Should(Succeed())

			// Verify cookie content
			Eventually(func() bool {
				if err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      secretName,
					Namespace: CustomAstarteNamespace,
				}, secret); err != nil {
					return false
				}
				return len(secret.Data) > 0 && len(secret.Data["erlang-cookie"]) > 0
			}, Timeout, Interval).Should(BeTrue())

			Expect(secret.Data).To(HaveKey("erlang-cookie"))
			cookieData := string(secret.Data["erlang-cookie"])
			Expect(cookieData).ToNot(BeEmpty())

			// Verify it's valid base32
			_, err = base32.StdEncoding.DecodeString(cookieData)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not recreate cookie secret if it already exists", func() {
			secretName := "existing-erlang-cookie"

			// Create secret first
			err := ensureErlangCookieSecret(secretName, cr, k8sClient, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			// Get the original cookie
			secret := &v1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      secretName,
					Namespace: CustomAstarteNamespace,
				}, secret)
			}, Timeout, Interval).Should(Succeed())
			originalCookie := string(secret.Data["erlang-cookie"])

			// Call again
			err = ensureErlangCookieSecret(secretName, cr, k8sClient, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			// Verify cookie wasn't changed
			Eventually(func() string {
				if err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      secretName,
					Namespace: CustomAstarteNamespace,
				}, secret); err != nil {
					return ""
				}
				return string(secret.Data["erlang-cookie"])
			}, Timeout, Interval).Should(Equal(originalCookie))
		})
	})

	Describe("Test computePersistentVolumeClaim", func() {
		It("should return correct PVC with default settings", func() {
			defaultName := "test-pvc"
			defaultSize := resource.MustParse("10Gi")

			name, pvc := computePersistentVolumeClaim(defaultName, &defaultSize, nil, cr)

			Expect(name).To(Equal(defaultName))
			Expect(pvc).ToNot(BeNil())
			Expect(pvc.Name).To(Equal(defaultName))
			Expect(pvc.Spec.AccessModes).To(Equal([]v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}))
			Expect(pvc.Spec.Resources.Requests[v1.ResourceStorage]).To(Equal(defaultSize))
			Expect(pvc.Spec.StorageClassName).To(BeNil())
		})

		It("should use custom storage spec when provided", func() {
			defaultName := "test-pvc"
			defaultSize := resource.MustParse("10Gi")
			customSize := resource.MustParse("20Gi")
			customClassName := "fast-ssd"

			storageSpec := &apiv2alpha1.AstartePersistentStorageSpec{
				Size:      &customSize,
				ClassName: customClassName,
			}

			name, pvc := computePersistentVolumeClaim(defaultName, &defaultSize, storageSpec, cr)

			Expect(name).To(Equal(defaultName))
			Expect(pvc).ToNot(BeNil())
			Expect(pvc.Spec.Resources.Requests[v1.ResourceStorage]).To(Equal(customSize))
			Expect(*pvc.Spec.StorageClassName).To(Equal(customClassName))
		})

		It("should return volume name and nil PVC when VolumeDefinition is provided", func() {
			defaultName := "test-pvc"
			defaultSize := resource.MustParse("10Gi")
			volumeName := "existing-volume"

			storageSpec := &apiv2alpha1.AstartePersistentStorageSpec{
				VolumeDefinition: &v1.Volume{
					Name: volumeName,
				},
			}

			name, pvc := computePersistentVolumeClaim(defaultName, &defaultSize, storageSpec, cr)

			Expect(name).To(Equal(volumeName))
			Expect(pvc).To(BeNil())
		})

		It("should use global storage class when not specified in storage spec", func() {
			defaultName := "test-pvc"
			defaultSize := resource.MustParse("10Gi")
			globalStorageClass := "global-storage"

			cr.Spec.StorageClassName = globalStorageClass
			storageSpec := &apiv2alpha1.AstartePersistentStorageSpec{
				Size: &defaultSize,
			}

			name, pvc := computePersistentVolumeClaim(defaultName, &defaultSize, storageSpec, cr)

			Expect(name).To(Equal(defaultName))
			Expect(pvc).ToNot(BeNil())
			Expect(*pvc.Spec.StorageClassName).To(Equal(globalStorageClass))
		})
	})

	Describe("Test getAffinityForClusteredResource", func() {
		It("should return custom affinity when provided", func() {
			customAffinity := &v1.Affinity{
				NodeAffinity: &v1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
						NodeSelectorTerms: []v1.NodeSelectorTerm{
							{
								MatchExpressions: []v1.NodeSelectorRequirement{
									{
										Key:      "kubernetes.io/arch",
										Operator: v1.NodeSelectorOpIn,
										Values:   []string{"amd64"},
									},
								},
							},
						},
					},
				},
			}

			resource := apiv2alpha1.AstarteGenericClusteredResource{
				CustomAffinity: customAffinity,
			}

			result := getAffinityForClusteredResource("test-app", resource)
			Expect(result).To(Equal(customAffinity))
		})

		It("should return anti-affinity when enabled and no custom affinity", func() {
			resource := apiv2alpha1.AstarteGenericClusteredResource{
				AntiAffinity: pointy.Bool(true),
			}

			result := getAffinityForClusteredResource("test-app", resource)
			Expect(result).ToNot(BeNil())
			Expect(result.PodAntiAffinity).ToNot(BeNil())
		})

		It("should return nil when anti-affinity is disabled and no custom affinity", func() {
			resource := apiv2alpha1.AstarteGenericClusteredResource{
				AntiAffinity: pointy.Bool(false),
			}

			result := getAffinityForClusteredResource("test-app", resource)
			Expect(result).To(BeNil())
		})
	})

	Describe("Test getAstarteImageForClusteredResource", func() {
		It("should return custom image when provided", func() {
			customImage := "custom/image:latest"
			resource := apiv2alpha1.AstarteGenericClusteredResource{
				Image: customImage,
			}

			result := getAstarteImageForClusteredResource("default-image", resource, cr)
			Expect(result).To(Equal(customImage))
		})

		It("should return image from channel when no custom image", func() {
			cr.Spec.DistributionChannel = "test-channel"
			resource := apiv2alpha1.AstarteGenericClusteredResource{}

			result := getAstarteImageForClusteredResource("test-image", resource, cr)
			Expect(result).To(ContainSubstring("test-channel/test-image"))
		})
	})

	Describe("Test getAstarteImageFromChannel", func() {
		It("should construct image URL with distribution channel", func() {
			cr.Spec.DistributionChannel = "astarte.example.com/registry"
			imageName := "astarte-housekeeping"
			tag := "1.3.0"

			result := getAstarteImageFromChannel(imageName, tag, cr)
			expected := "astarte.example.com/registry/astarte-housekeeping:1.3.0"
			Expect(result).To(Equal(expected))
		})

		It("should construct image URL with empty distribution channel", func() {
			cr.Spec.DistributionChannel = ""
			imageName := "astarte-realm-management"
			tag := "1.2.0"

			result := getAstarteImageFromChannel(imageName, tag, cr)
			expected := "/astarte-realm-management:1.2.0"
			Expect(result).To(Equal(expected))
		})
	})

	Describe("Test getDeploymentStrategyForClusteredResource", func() {
		It("should return Recreate strategy for DataUpdaterPlant", func() {
			resource := apiv2alpha1.AstarteGenericClusteredResource{}

			result := getDeploymentStrategyForClusteredResource(cr, resource, apiv2alpha1.DataUpdaterPlant)
			Expect(result.Type).To(Equal(appsv1.RecreateDeploymentStrategyType))
		})

		It("should return Recreate strategy for TriggerEngine", func() {
			resource := apiv2alpha1.AstarteGenericClusteredResource{}

			result := getDeploymentStrategyForClusteredResource(cr, resource, apiv2alpha1.TriggerEngine)
			Expect(result.Type).To(Equal(appsv1.RecreateDeploymentStrategyType))
		})

		It("should return Recreate strategy for FlowComponent", func() {
			resource := apiv2alpha1.AstarteGenericClusteredResource{}

			result := getDeploymentStrategyForClusteredResource(cr, resource, apiv2alpha1.FlowComponent)
			Expect(result.Type).To(Equal(appsv1.RecreateDeploymentStrategyType))
		})

		It("should return custom strategy when provided", func() {
			customStrategy := &appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
				},
			}

			resource := apiv2alpha1.AstarteGenericClusteredResource{
				DeploymentStrategy: customStrategy,
			}

			result := getDeploymentStrategyForClusteredResource(cr, resource, apiv2alpha1.AppEngineAPI)
			Expect(result).To(Equal(*customStrategy))
		})

		It("should return RollingUpdate as default for API components", func() {
			resource := apiv2alpha1.AstarteGenericClusteredResource{}

			result := getDeploymentStrategyForClusteredResource(cr, resource, apiv2alpha1.AppEngineAPI)
			Expect(result.Type).To(Equal(appsv1.RollingUpdateDeploymentStrategyType))
		})
	})

	Describe("Test getDataQueueCount", func() {
		It("should return custom data queue count when specified", func() {
			cr.Spec.Components.DataUpdaterPlant.DataQueueCount = pointy.Int(256)

			result := getDataQueueCount(cr)
			Expect(result).To(Equal(256))
		})

		It("should return default data queue count when not specified", func() {
			result := getDataQueueCount(cr)
			Expect(result).To(Equal(128)) // Default value
		})
	})

	Describe("Test getAppEngineAPIMaxResultslimit", func() {
		It("should return custom max results limit when specified", func() {
			cr.Spec.Components.AppengineAPI.MaxResultsLimit = pointy.Int(5000)

			result := getAppEngineAPIMaxResultslimit(cr)
			Expect(result).To(Equal(5000))
		})

		It("should return default max results limit when not specified", func() {
			result := getAppEngineAPIMaxResultslimit(cr)
			Expect(result).To(Equal(10000)) // Default value
		})
	})

	Describe("Test getBaseAstarteAPIURL", func() {
		It("should return HTTPS URL when SSL is enabled", func() {
			cr.Spec.API.SSL = pointy.Bool(true)
			cr.Spec.API.Host = "api.astarte.example.com"

			result := getBaseAstarteAPIURL(cr)
			Expect(result).To(Equal("https://api.astarte.example.com"))
		})

		It("should return HTTP URL when SSL is disabled", func() {
			cr.Spec.API.SSL = pointy.Bool(false)
			cr.Spec.API.Host = "api.astarte.example.com"

			result := getBaseAstarteAPIURL(cr)
			Expect(result).To(Equal("http://api.astarte.example.com"))
		})

		It("should return HTTPS URL by default when SSL is not specified", func() {
			cr.Spec.API.Host = "api.astarte.example.com"

			result := getBaseAstarteAPIURL(cr)
			Expect(result).To(Equal("https://api.astarte.example.com"))
		})
	})

	Describe("Test createOrUpdateService", func() {
		It("should create service with correct specifications", func() {
			serviceName := "test-service"
			matchLabels := map[string]string{"app": "test-app"}
			labels := map[string]string{"component": "astarte", "app": "test-app"}

			err := createOrUpdateService(cr, k8sClient, serviceName, scheme.Scheme, matchLabels, labels)
			Expect(err).ToNot(HaveOccurred())

			// Verify service was created
			service := &v1.Service{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      serviceName,
					Namespace: CustomAstarteNamespace,
				}, service)
			}, Timeout, Interval).Should(Succeed())

			// Verify service specifications
			Expect(service.Labels).To(Equal(labels))
			Expect(service.Spec.Type).To(Equal(v1.ServiceTypeClusterIP))
			Expect(service.Spec.ClusterIP).To(Equal(noneClusterIP))
			Expect(service.Spec.Selector).To(Equal(matchLabels))
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Name).To(Equal("http"))
			Expect(service.Spec.Ports[0].Port).To(Equal(astarteServicesPort))
			Expect(service.Spec.Ports[0].TargetPort).To(Equal(intstr.FromString("http")))
			Expect(service.Spec.Ports[0].Protocol).To(Equal(v1.ProtocolTCP))
		})

		It("should update existing service", func() {
			serviceName := "update-test-service"
			matchLabels := map[string]string{"app": "test-app"}
			labels := map[string]string{"component": "astarte"}

			// Create service first
			err := createOrUpdateService(cr, k8sClient, serviceName, scheme.Scheme, matchLabels, labels)
			Expect(err).ToNot(HaveOccurred())

			// Wait for service creation to complete
			service := &v1.Service{}
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      serviceName,
					Namespace: CustomAstarteNamespace,
				}, service)
			}, Timeout, Interval).Should(Succeed())

			// Update with new labels
			newLabels := map[string]string{"component": "astarte", "version": "v2"}
			err = createOrUpdateService(cr, k8sClient, serviceName, scheme.Scheme, matchLabels, newLabels)
			Expect(err).ToNot(HaveOccurred())

			// Verify service was updated
			Eventually(func() map[string]string {
				if err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      serviceName,
					Namespace: CustomAstarteNamespace,
				}, service); err != nil {
					return nil
				}
				return service.Labels
			}, Timeout, Interval).Should(Equal(newLabels))
		})
	})

	Describe("Test getReplicaCountForResource", func() {
		// This function uses HPA which is hard to test with envtest. Leaving it untested for now.
	})

	Describe("Test getCassandraNodes", func() {
		It("should return formatted cassandra nodes string", func() {
			result := getCassandraNodes(cr)
			expected := "cassandra.example.com:9042"
			Expect(result).To(Equal(expected))
		})

		It("should return empty string when no cassandra connection", func() {
			crWithoutCassandra := cr.DeepCopy()
			crWithoutCassandra.Spec.Cassandra.Connection = nil

			result := getCassandraNodes(crWithoutCassandra)
			Expect(result).To(Equal(""))
		})

		It("should handle multiple cassandra nodes", func() {
			cr.Spec.Cassandra.Connection.Nodes = append(cr.Spec.Cassandra.Connection.Nodes,
				apiv2alpha1.HostAndPort{
					Host: "cassandra2.example.com",
					Port: pointy.Int32(9042),
				})

			result := getCassandraNodes(cr)
			Expect(result).To(Equal("cassandra.example.com:9042,cassandra2.example.com:9042"))
		})
	})

	Describe("Test getErlangClusteringCookieSecretName", func() {
		It("should generate correct cookie secret name", func() {
			result := getErlangClusteringCookieSecretName(cr)
			expected := CustomAstarteName + "-erlang-clustering-cookie"
			Expect(result).To(Equal(expected))
		})
	})

	Describe("Test getErlangClusteringCookieSecretReference", func() {
		It("should create correct secret reference", func() {
			result := getErlangClusteringCookieSecretReference(cr)

			Expect(result).ToNot(BeNil())
			Expect(result.SecretKeyRef).ToNot(BeNil())
			Expect(result.SecretKeyRef.Name).To(Equal(CustomAstarteName + "-erlang-clustering-cookie"))
			Expect(result.SecretKeyRef.Key).To(Equal("erlang-cookie"))
		})
	})

	Describe("Test getImagePullPolicy", func() {
		It("should return component-specific image pull policy when set", func() {
			component := apiv2alpha1.AstarteGenericClusteredResource{
				ImagePullPolicy: &[]v1.PullPolicy{v1.PullAlways}[0],
			}

			result := getImagePullPolicy(cr, component)
			Expect(result).To(Equal(v1.PullAlways))
		})

		It("should return global image pull policy when component policy not set", func() {
			cr.Spec.ImagePullPolicy = &[]v1.PullPolicy{v1.PullIfNotPresent}[0]
			component := apiv2alpha1.AstarteGenericClusteredResource{}

			result := getImagePullPolicy(cr, component)
			Expect(result).To(Equal(v1.PullIfNotPresent))
		})
	})

	Describe("Test getAstarteCommonVolumes", func() {
		It("should return basic volumes without SSL configuration", func() {
			crWithoutSSL := cr.DeepCopy()
			crWithoutSSL.Spec.RabbitMQ.Connection.SSLConfiguration.CustomCASecret.Name = ""
			crWithoutSSL.Spec.Cassandra.Connection.SSLConfiguration.CustomCASecret.Name = ""

			volumes := getAstarteCommonVolumes(crWithoutSSL)

			Expect(volumes).To(HaveLen(1))
			Expect(volumes[0].Name).To(Equal("beam-config"))
			Expect(volumes[0].VolumeSource.ConfigMap).ToNot(BeNil())
			Expect(volumes[0].VolumeSource.ConfigMap.Name).To(Equal(CustomAstarteName + "-generic-erlang-configuration"))
		})

		It("should include RabbitMQ SSL volume when configured", func() {
			cr.Spec.RabbitMQ.Connection.SSLConfiguration.CustomCASecret.Name = "rabbitmq-ca-secret"

			volumes := getAstarteCommonVolumes(cr)

			Expect(volumes).To(HaveLen(2))

			var foundRabbitMQVolume bool
			for _, vol := range volumes {
				if vol.Name == "rabbitmq-ssl-ca" {
					foundRabbitMQVolume = true
					Expect(vol.VolumeSource.Secret).ToNot(BeNil())
					Expect(vol.VolumeSource.Secret.SecretName).To(Equal("rabbitmq-ca-secret"))
					Expect(vol.VolumeSource.Secret.Items).To(HaveLen(1))
					Expect(vol.VolumeSource.Secret.Items[0].Key).To(Equal("ca.crt"))
					Expect(vol.VolumeSource.Secret.Items[0].Path).To(Equal("ca.crt"))
				}
			}
			Expect(foundRabbitMQVolume).To(BeTrue())
		})

		It("should include Cassandra SSL volume when configured", func() {
			cr.Spec.Cassandra.Connection.SSLConfiguration.CustomCASecret.Name = "cassandra-ca-secret"

			volumes := getAstarteCommonVolumes(cr)

			Expect(volumes).To(HaveLen(2))

			var foundCassandraVolume bool
			for _, vol := range volumes {
				if vol.Name == "cassandra-ssl-ca" {
					foundCassandraVolume = true
					Expect(vol.VolumeSource.Secret).ToNot(BeNil())
					Expect(vol.VolumeSource.Secret.SecretName).To(Equal("cassandra-ca-secret"))
					Expect(vol.VolumeSource.Secret.Items).To(HaveLen(1))
					Expect(vol.VolumeSource.Secret.Items[0].Key).To(Equal("ca.crt"))
					Expect(vol.VolumeSource.Secret.Items[0].Path).To(Equal("ca.crt"))
				}
			}
			Expect(foundCassandraVolume).To(BeTrue())
		})

		It("should include both SSL volumes when both are configured", func() {
			cr.Spec.RabbitMQ.Connection.SSLConfiguration.CustomCASecret.Name = "rabbitmq-ca-secret"
			cr.Spec.Cassandra.Connection.SSLConfiguration.CustomCASecret.Name = "cassandra-ca-secret"

			volumes := getAstarteCommonVolumes(cr)

			Expect(volumes).To(HaveLen(3))

			volumeNames := make([]string, len(volumes))
			for i, vol := range volumes {
				volumeNames[i] = vol.Name
			}
			Expect(volumeNames).To(ContainElement("beam-config"))
			Expect(volumeNames).To(ContainElement("rabbitmq-ssl-ca"))
			Expect(volumeNames).To(ContainElement("cassandra-ssl-ca"))
		})
	})

	Describe("Test getAstarteCommonVolumeMounts", func() {
		It("should return basic volume mounts without SSL configuration", func() {
			crWithoutSSL := cr.DeepCopy()
			crWithoutSSL.Spec.RabbitMQ.Connection.SSLConfiguration.CustomCASecret.Name = ""
			crWithoutSSL.Spec.Cassandra.Connection.SSLConfiguration.CustomCASecret.Name = ""

			mounts := getAstarteCommonVolumeMounts(crWithoutSSL)

			Expect(mounts).To(HaveLen(1))
			Expect(mounts[0].Name).To(Equal("beam-config"))
			Expect(mounts[0].MountPath).To(Equal("/beamconfig"))
			Expect(mounts[0].ReadOnly).To(BeTrue())
		})

		It("should include RabbitMQ SSL mount when configured", func() {
			cr.Spec.RabbitMQ.Connection.SSLConfiguration.CustomCASecret.Name = "rabbitmq-ca-secret"

			mounts := getAstarteCommonVolumeMounts(cr)

			Expect(mounts).To(HaveLen(2))

			var foundRabbitMQMount bool
			for _, mount := range mounts {
				if mount.Name == "rabbitmq-ssl-ca" {
					foundRabbitMQMount = true
					Expect(mount.MountPath).To(Equal("/rabbitmq-ssl"))
					Expect(mount.ReadOnly).To(BeTrue())
				}
			}
			Expect(foundRabbitMQMount).To(BeTrue())
		})

		It("should include Cassandra SSL mount when configured", func() {
			cr.Spec.Cassandra.Connection.SSLConfiguration.CustomCASecret.Name = "cassandra-ca-secret"

			mounts := getAstarteCommonVolumeMounts(cr)

			Expect(mounts).To(HaveLen(2))

			var foundCassandraMount bool
			for _, mount := range mounts {
				if mount.Name == "cassandra-ssl-ca" {
					foundCassandraMount = true
					Expect(mount.MountPath).To(Equal("/cassandra-ssl"))
					Expect(mount.ReadOnly).To(BeTrue())
				}
			}
			Expect(foundCassandraMount).To(BeTrue())
		})

		It("should include both SSL mounts when both are configured", func() {
			cr.Spec.RabbitMQ.Connection.SSLConfiguration.CustomCASecret.Name = "rabbitmq-ca-secret"
			cr.Spec.Cassandra.Connection.SSLConfiguration.CustomCASecret.Name = "cassandra-ca-secret"

			mounts := getAstarteCommonVolumeMounts(cr)

			Expect(mounts).To(HaveLen(3))

			mountNames := make([]string, len(mounts))
			for i, mount := range mounts {
				mountNames[i] = mount.Name
			}
			Expect(mountNames).To(ContainElement("beam-config"))
			Expect(mountNames).To(ContainElement("rabbitmq-ssl-ca"))
			Expect(mountNames).To(ContainElement("cassandra-ssl-ca"))
		})
	})

	Describe("Test getHPAStatusForResource", func() {
		// This function uses HPA which is hard to test with envtest. Leaving it untested for now.
	})
})
