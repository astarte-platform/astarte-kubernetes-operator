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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base32"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strconv"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"go.openly.dev/pointy"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/misc"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/version"
)

const (
	astarteServicesPort int32  = 4000
	noneClusterIP       string = "None"
	highPriority        string = "high"
	midPriority         string = "mid"
	lowPriority         string = "low"
)

func encodePEMBlockToEncodedBytes(block *pem.Block) string {
	return string(pem.EncodeToMemory(block))
}

func storePublicKeyInSecret(name string, publicKey *rsa.PublicKey, cr *apiv2alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	pkixBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	var publicKeyPEM = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pkixBytes,
	}

	publicKeySecretData := encodePEMBlockToEncodedBytes(publicKeyPEM)

	secretData := map[string]string{
		"public-key": publicKeySecretData,
	}

	// Set Astarte instance as the owner and controller
	_, err = misc.ReconcileSecretString(name, secretData, cr, c, scheme, log)
	return err
}

func storePrivateKeyInSecret(name string, privateKey *rsa.PrivateKey, cr *apiv2alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	var privateKeyPEM = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeySecretData := encodePEMBlockToEncodedBytes(privateKeyPEM)

	secretData := map[string]string{
		"private-key": privateKeySecretData,
	}

	// Set Astarte instance as the owner and controller
	_, err := misc.ReconcileSecretString(name, secretData, cr, c, scheme, log)
	return err
}

func generateKeyPair() (*rsa.PrivateKey, error) {
	reader := rand.Reader
	bitSize := 4096

	return rsa.GenerateKey(reader, bitSize)
}

func getStandardAntiAffinityForAppLabel(app string) *v1.Affinity {
	return &v1.Affinity{
		PodAntiAffinity: &v1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "app",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{app},
							},
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				},
			},
		},
	}
}

func reconcileStandardRBACForClusteringForApp(name string, policyRules []rbacv1.PolicyRule, cr *apiv2alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	// Service Account
	serviceAccount := &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cr.Namespace}}
	if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, serviceAccount, func() error {
		if err := controllerutil.SetControllerReference(cr, serviceAccount, scheme); err != nil {
			return err
		}
		// Actually nothing to do here.
		return nil
	}); err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, serviceAccount)
	} else {
		return err
	}

	// Role
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cr.Namespace}}
	if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, role, func() error {
		if err := controllerutil.SetControllerReference(cr, role, scheme); err != nil {
			return err
		}
		// Always impose what we want in terms of policy roles without caring.
		role.Rules = policyRules
		return nil
	}); err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, serviceAccount)
	} else {
		return err
	}

	// Role Binding
	roleBinding := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cr.Namespace}}
	if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, roleBinding, func() error {
		if err := controllerutil.SetControllerReference(cr, roleBinding, scheme); err != nil {
			return err
		}
		// Always impose what we want in terms of policy roles without caring.
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: name,
			},
		}
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		}
		return nil
	}); err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, serviceAccount)
	} else {
		return err
	}

	return nil
}

func reconcileRBACForFlow(name string, cr *apiv2alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	// Service Account
	serviceAccount := &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cr.Namespace}}
	if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, serviceAccount, func() error {
		if err := controllerutil.SetControllerReference(cr, serviceAccount, scheme); err != nil {
			return err
		}
		// Actually nothing to do here.
		return nil
	}); err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, serviceAccount)
	} else {
		return err
	}

	// Role
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cr.Namespace}}
	if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, role, func() error {
		if err := controllerutil.SetControllerReference(cr, role, scheme); err != nil {
			return err
		}
		// Always impose what we want in terms of policy roles without caring.
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"api.astarte-platform.org"},
				Resources: []string{"flows"},
				Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
		}
		return nil
	}); err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, serviceAccount)
	} else {
		return err
	}

	// Role Binding
	roleBinding := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cr.Namespace}}
	if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, roleBinding, func() error {
		if err := controllerutil.SetControllerReference(cr, roleBinding, scheme); err != nil {
			return err
		}
		// Always impose what we want in terms of policy roles without caring.
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: name,
			},
		}
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		}
		return nil
	}); err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, serviceAccount)
	} else {
		return err
	}

	return nil
}

func getAstarteImageFromChannel(name, tag string, cr *apiv2alpha1.Astarte) string {
	var distributionChannel string
	if cr.Spec.DistributionChannel != "" {
		distributionChannel = cr.Spec.DistributionChannel
	}

	return fmt.Sprintf("%s/%s:%s", distributionChannel, name, tag)
}

func getImagePullPolicy(cr *apiv2alpha1.Astarte, astarteComponent apiv2alpha1.AstarteGenericClusteredResource) v1.PullPolicy {
	if astarteComponent.ImagePullPolicy != nil {
		return *astarteComponent.ImagePullPolicy
	}

	return *cr.Spec.ImagePullPolicy
}

func ensureErlangCookieSecret(secretName string, cr *apiv2alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	theCookie := &v1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: cr.Namespace}, theCookie); err != nil {
		if kerrors.IsNotFound(err) {
			// Create it.
			// TODO: Throw a reconcile error and/or delete the persistent volume if we are in that situation.
			reqLogger.Info("Creating new Cookie", "cookie-name", secretName)
			cookie := make([]byte, 32)
			if _, e := rand.Read(cookie); e != nil {
				return e
			}

			cookieSecret := v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: cr.Namespace,
				},
				StringData: map[string]string{"erlang-cookie": base32.StdEncoding.EncodeToString(cookie)},
			}
			if e := controllerutil.SetControllerReference(cr, &cookieSecret, scheme); e != nil {
				return e
			}
			// We force creation as for no reason in the world we want to even think about updating this.
			if e := c.Create(context.TODO(), &cookieSecret); e != nil {
				return e
			}
		} else {
			// Return here
			return err
		}
	}

	// All went well
	return nil
}

func computePersistentVolumeClaim(defaultName string, defaultSize *resource.Quantity, storageSpec *apiv2alpha1.AstartePersistentStorageSpec,
	cr *apiv2alpha1.Astarte) (string, *v1.PersistentVolumeClaim) {
	var storageClassName string
	dataVolumeSize := defaultSize
	dataVolumeName := defaultName
	if storageSpec != nil {
		if storageSpec.VolumeDefinition != nil {
			return storageSpec.VolumeDefinition.Name, nil
		}
		if storageSpec.Size != nil {
			dataVolumeSize = storageSpec.Size
		}
		if storageSpec.ClassName != "" {
			storageClassName = storageSpec.ClassName
		} else if cr.Spec.StorageClassName != "" {
			storageClassName = cr.Spec.StorageClassName
		}
	}

	persistentVolumeClaimSpec := v1.PersistentVolumeClaimSpec{
		AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
		Resources:   v1.VolumeResourceRequirements{Requests: v1.ResourceList{v1.ResourceStorage: *dataVolumeSize}},
	}
	if storageClassName != "" {
		persistentVolumeClaimSpec.StorageClassName = &storageClassName
	}

	return dataVolumeName, &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataVolumeName,
		},
		Spec: persistentVolumeClaimSpec,
	}
}

func getAstarteCommonEnvVars(deploymentName string, cr *apiv2alpha1.Astarte, backend apiv2alpha1.AstarteGenericClusteredResource, component apiv2alpha1.AstarteComponent) []v1.EnvVar {
	ret := []v1.EnvVar{
		{
			Name:  "RELEASE_CONFIG_DIR",
			Value: "/beamconfig",
		},
		{
			Name:  "REPLACE_OS_VARS",
			Value: strconv.FormatBool(true),
		},
		{
			Name:      "MY_POD_IP",
			ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "status.podIP"}},
		},
		{
			Name:  "RELEASE_NAME",
			Value: component.DockerImageName(),
		},
	}

	// We need extra care for Erlang cookie, as some services share the same one
	if component == apiv2alpha1.AppEngineAPI || component == apiv2alpha1.DataUpdaterPlant {
		ret = append(ret, v1.EnvVar{
			Name:      "RELEASE_COOKIE",
			ValueFrom: getErlangClusteringCookieSecretReference(cr),
		})

		// If Astarte version is bigger than 1.2.0, we need to set clustering environment variables.
		if cr.Spec.Version != "" && semver.MustParse(cr.Spec.Version).GreaterThan(semver.MustParse("1.2.0")) {
			ret = append(ret, v1.EnvVar{
				Name:  "CLUSTERING_STRATEGY",
				Value: "kubernetes",
			})

			ret = append(ret,
				v1.EnvVar{
					Name:  "DATA_UPDATER_PLANT_CLUSTERING_KUBERNETES_SELECTOR",
					Value: fmt.Sprint("app=", cr.Name, "-data-updater-plant"),
				})

			ret = append(ret,
				v1.EnvVar{
					Name:  "VERNEMQ_CLUSTERING_KUBERNETES_SELECTOR",
					Value: fmt.Sprint("app=", cr.Name, "-vernemq"),
				})

			ret = append(ret,
				v1.EnvVar{
					Name:  "CLUSTERING_KUBERNETES_NAMESPACE",
					Value: cr.Namespace,
				})
			ret = append(ret,
				v1.EnvVar{
					Name:  "VERNEMQ_CLUSTERING_KUBERNETES_SERVICE_NAME",
					Value: fmt.Sprint(cr.Name, "-vernemq"),
				})
		}

	} else {
		ret = append(ret, v1.EnvVar{
			Name: "RELEASE_COOKIE",
			ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: deploymentName + "-cookie"},
				Key:                  "erlang-cookie",
			}},
		})
	}

	// Add Port (needed for all components, since we also have metrics)
	suffix := "_PORT"
	if component == apiv2alpha1.Housekeeping || component == apiv2alpha1.Pairing ||
		component == apiv2alpha1.RealmManagement {
		suffix = "_API_PORT"
	}
	ret = append(ret, v1.EnvVar{
		Name:  strings.ToUpper(component.String()) + suffix,
		Value: strconv.Itoa(int(astarteServicesPort)),
	})

	// Add any explicit additional env
	if len(backend.AdditionalEnv) > 0 {
		ret = append(ret, backend.AdditionalEnv...)
	}

	// Return with the RabbitMQ variables appended
	return appendRabbitMQConnectionEnvVars(ret, "RPC_AMQP_CONNECTION", cr)
}

func appendCassandraConnectionEnvVars(ret []v1.EnvVar, cr *apiv2alpha1.Astarte) []v1.EnvVar {
	spec := cr.Spec.Cassandra.Connection
	if spec == nil {
		return ret
	}

	// pool size
	if spec.PoolSize != nil {
		ret = append(ret,
			v1.EnvVar{
				Name:  "CASSANDRA_POOL_SIZE",
				Value: strconv.Itoa(*spec.PoolSize),
			},
		)
	}

	// SSL
	if spec.SSLConfiguration.Enable {
		ret = append(ret, v1.EnvVar{
			Name:  "CASSANDRA_SSL_ENABLED",
			Value: "true",
		})

		// CA configuration
		if spec.SSLConfiguration.CustomCASecret.Name != "" {
			// getAstarteCommonVolumes will mount the volume for us, if we're here. So trust the rest of our code.
			ret = append(ret, v1.EnvVar{
				Name:  "CASSANDRA_SSL_CA_FILE",
				Value: "/cassandra-ssl/ca.crt",
			})
		}

		// SNI configuration
		switch {
		case spec.SSLConfiguration.CustomSNI != "":
			ret = append(ret, v1.EnvVar{
				Name:  "CASSANDRA_SSL_CUSTOM_SNI",
				Value: spec.SSLConfiguration.CustomSNI,
			})
		case !pointy.BoolValue(spec.SSLConfiguration.SNI, true):
			ret = append(ret, v1.EnvVar{
				Name:  "CASSANDRA_SSL_DISABLE_SNI",
				Value: "true",
			})
		}
	}

	if spec.CredentialsSecret != nil {
		// Fetch our Credentials for Cassandra
		userCredentialsSecretName, userCredentialsSecretUsernameKey, userCredentialsSecretPasswordKey := misc.GetCassandraUserCredentialsSecret(cr)

		// Standard Cassandra env vars that we need to plug in
		ret = append(ret,
			v1.EnvVar{
				Name: "CASSANDRA_USERNAME",
				ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
					Key:                  userCredentialsSecretUsernameKey,
				}},
			},
			v1.EnvVar{
				Name: "CASSANDRA_PASSWORD",
				ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
					Key:                  userCredentialsSecretPasswordKey,
				}},
			},
		)
	}

	// Enable or disable the keepalive option for the xandra connection. Default to true.
	ret = append(ret,
		v1.EnvVar{
			Name:  "CASSANDRA_ENABLE_KEEPALIVE",
			Value: strconv.FormatBool(pointy.BoolValue(spec.EnableKeepalive, true)),
		},
	)

	return ret
}

func appendRabbitMQConnectionEnvVars(ret []v1.EnvVar, prefix string, cr *apiv2alpha1.Astarte) []v1.EnvVar {
	spec := cr.Spec.RabbitMQ.Connection

	if spec == nil {
		return ret
	}

	// Let's verify Virtualhost and default to "/" where needed. Al
	virtualHost := "/"
	if spec.VirtualHost != "" {
		virtualHost = spec.VirtualHost
		ret = append(ret, v1.EnvVar{
			Name:  prefix + "_VIRTUAL_HOST",
			Value: spec.VirtualHost,
		})
	}

	// SSL
	if spec.SSLConfiguration.Enable {
		ret = append(ret, v1.EnvVar{
			Name:  prefix + "_SSL_ENABLED",
			Value: "true",
		})

		// CA configuration
		if spec.SSLConfiguration.CustomCASecret.Name != "" {
			// getAstarteCommonVolumes will mount the volume for us, if we're here. So trust the rest of our code.
			ret = append(ret, v1.EnvVar{
				Name:  prefix + "_SSL_CA_FILE",
				Value: "/rabbitmq-ssl/ca.crt",
			})
		}

		// SNI configuration
		switch {
		case spec.SSLConfiguration.CustomSNI != "":
			ret = append(ret, v1.EnvVar{
				Name:  prefix + "_SSL_CUSTOM_SNI",
				Value: spec.SSLConfiguration.CustomSNI,
			})
		case !pointy.BoolValue(spec.SSLConfiguration.SNI, true):
			ret = append(ret, v1.EnvVar{
				Name:  prefix + "_SSL_DISABLE_SNI",
				Value: "true",
			})
		}
	}

	// Fetch our Credentials for RabbitMQ
	rabbitMQHost, rabbitMQPort := misc.GetRabbitMQHostnameAndPort(cr)
	userCredentialsSecretName, userCredentialsSecretUsernameKey, userCredentialsSecretPasswordKey := misc.GetRabbitMQUserCredentialsSecret(cr)

	// Standard RMQ env vars that, like it or not, we need to plug in everywhere.
	ret = append(ret,
		v1.EnvVar{
			Name:  prefix + "_HOST",
			Value: rabbitMQHost,
		},
		v1.EnvVar{
			Name:  prefix + "_PORT",
			Value: strconv.Itoa(int(rabbitMQPort)),
		},
		v1.EnvVar{
			Name:  prefix + "_VIRTUAL_HOST",
			Value: virtualHost,
		},
		v1.EnvVar{
			Name: prefix + "_USERNAME",
			ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
				Key:                  userCredentialsSecretUsernameKey,
			}},
		},
		v1.EnvVar{
			Name: prefix + "_PASSWORD",
			ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
				Key:                  userCredentialsSecretPasswordKey,
			}},
		},
	)

	// Here we go
	return ret
}

func getErlangClusteringCookieSecretReference(cr *apiv2alpha1.Astarte) *v1.EnvVarSource {
	return &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
		LocalObjectReference: v1.LocalObjectReference{Name: getErlangClusteringCookieSecretName(cr)},
		Key:                  "erlang-cookie",
	}}
}

func getErlangClusteringCookieSecretName(cr *apiv2alpha1.Astarte) string {
	return cr.Name + "-erlang-clustering-cookie"
}

func getAstarteCommonVolumes(cr *apiv2alpha1.Astarte) []v1.Volume {
	ret := []v1.Volume{
		{
			Name: "beam-config",
			VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{Name: fmt.Sprintf("%s-generic-erlang-configuration", cr.Name)},
				Items:                []v1.KeyToPath{{Key: "vm.args", Path: "vm.args"}},
			}},
		},
	}

	if cr.Spec.RabbitMQ.Connection != nil {
		if cr.Spec.RabbitMQ.Connection.SSLConfiguration.CustomCASecret.Name != "" {
			// Mount the secret!
			ret = append(ret, v1.Volume{
				Name: "rabbitmq-ssl-ca",
				VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{
					SecretName: cr.Spec.RabbitMQ.Connection.SSLConfiguration.CustomCASecret.Name,
					Items:      []v1.KeyToPath{{Key: "ca.crt", Path: "ca.crt"}},
				}},
			})
		}
	}

	if cr.Spec.Cassandra.Connection != nil {
		if cr.Spec.Cassandra.Connection.SSLConfiguration.CustomCASecret.Name != "" {
			// Mount the secret!
			ret = append(ret, v1.Volume{
				Name: "cassandra-ssl-ca",
				VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{
					SecretName: cr.Spec.Cassandra.Connection.SSLConfiguration.CustomCASecret.Name,
					Items:      []v1.KeyToPath{{Key: "ca.crt", Path: "ca.crt"}},
				}},
			})
		}
	}

	return ret
}

func getAstarteCommonVolumeMounts(cr *apiv2alpha1.Astarte) []v1.VolumeMount {
	ret := []v1.VolumeMount{
		{
			Name:      "beam-config",
			MountPath: "/beamconfig",
			ReadOnly:  true,
		},
	}

	if cr.Spec.RabbitMQ.Connection != nil {
		if cr.Spec.RabbitMQ.Connection.SSLConfiguration.CustomCASecret.Name != "" {
			// Mount the secret!
			ret = append(ret, v1.VolumeMount{
				Name:      "rabbitmq-ssl-ca",
				MountPath: "/rabbitmq-ssl",
				ReadOnly:  true,
			})
		}
	}

	if cr.Spec.Cassandra.Connection != nil {
		if cr.Spec.Cassandra.Connection.SSLConfiguration.CustomCASecret.Name != "" {
			// Mount the secret!
			ret = append(ret, v1.VolumeMount{
				Name:      "cassandra-ssl-ca",
				MountPath: "/cassandra-ssl",
				ReadOnly:  true,
			})
		}
	}

	return ret
}

func getAffinityForClusteredResource(appLabel string, resource apiv2alpha1.AstarteGenericClusteredResource) *v1.Affinity {
	affinity := resource.CustomAffinity
	if affinity == nil && pointy.BoolValue(resource.AntiAffinity, true) {
		affinity = getStandardAntiAffinityForAppLabel(appLabel)
	}
	return affinity
}

func getAstarteImageForClusteredResource(defaultImageName string, resource apiv2alpha1.AstarteGenericClusteredResource, cr *apiv2alpha1.Astarte) string {
	if resource.Image != "" {
		return resource.Image
	}

	return getAstarteImageFromChannel(defaultImageName, version.GetVersionForAstarteComponent(cr.Spec.Version, resource.Version), cr)
}

func getDeploymentStrategyForClusteredResource(cr *apiv2alpha1.Astarte, resource apiv2alpha1.AstarteGenericClusteredResource, component apiv2alpha1.AstarteComponent) appsv1.DeploymentStrategy {
	switch {
	case component == apiv2alpha1.DataUpdaterPlant, component == apiv2alpha1.TriggerEngine,
		component == apiv2alpha1.FlowComponent:
		return appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	case resource.DeploymentStrategy != nil:
		return *resource.DeploymentStrategy
	case cr.Spec.DeploymentStrategy != nil:
		return *cr.Spec.DeploymentStrategy
	default:
		return appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
		}
	}
}

func getDataQueueCount(cr *apiv2alpha1.Astarte) int {
	return pointy.IntValue(cr.Spec.Components.DataUpdaterPlant.DataQueueCount, 128)
}

func getAppEngineAPIMaxResultslimit(cr *apiv2alpha1.Astarte) int {
	return pointy.IntValue(cr.Spec.Components.AppengineAPI.MaxResultsLimit, 10000)
}

func getBaseAstarteAPIURL(cr *apiv2alpha1.Astarte) string {
	scheme := "https"
	if !pointy.BoolValue(cr.Spec.API.SSL, true) {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s", scheme, cr.Spec.API.Host)
}

func createOrUpdateService(cr *apiv2alpha1.Astarte, c client.Client, serviceName string, scheme *runtime.Scheme,
	matchLabels, labels map[string]string) error {
	service := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: serviceName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, service, func() error {
		if e := controllerutil.SetControllerReference(cr, service, scheme); e != nil {
			return e
		}
		// Always set everything to what we require.
		service.ObjectMeta.Labels = labels
		service.Spec.Type = v1.ServiceTypeClusterIP
		service.Spec.ClusterIP = noneClusterIP
		service.Spec.Ports = []v1.ServicePort{
			{
				Name:       "http",
				Port:       astarteServicesPort,
				TargetPort: intstr.FromString("http"),
				Protocol:   v1.ProtocolTCP,
			},
		}
		service.Spec.Selector = matchLabels
		return nil
	})

	misc.LogCreateOrUpdateOperationResult(log, result, cr, service)
	return err
}

func computePodLabels(r apiv2alpha1.PodLabelsGetter, labels map[string]string) map[string]string {
	// Validating webhook guarantees that custom user labels won't interfere with operator's.
	podLabels := map[string]string{}
	for k, v := range labels {
		podLabels[k] = v
	}
	for k, v := range r.GetPodLabels() {
		podLabels[k] = v
	}
	return podLabels
}

func getReplicaCountForResource(resource *apiv2alpha1.AstarteGenericClusteredResource, cr *apiv2alpha1.Astarte, c client.Client, log logr.Logger) *int32 {
	if cr.Spec.Features.Autoscaling && resource.Autoscale != nil {
		if hpaStatus, err := getHPAStatusForResource(resource.Autoscale.Horizontal, cr, c, log); err == nil {
			// This is a special case to avoid a race condition with HPA, which can lead to the Operator
			// and HPA fighting over replica count, causing service disruption. This can happen when
			// the HPA isn't able to fetch metrics for the pods, and decides to scale down to 0.
			// This is a known issue in HPA, and this is a workaround to avoid it.
			if hpaStatus.DesiredReplicas == 0 {
				log.Info("HPA is reporting 0 desired replicas. This is likely a transient state. Ignoring HPA and using the spec's replica count", "HPA.Name", resource.Autoscale.Horizontal)
				return resource.Replicas
			}
			log.Info("Getting replica count from HPA", "value", hpaStatus.DesiredReplicas)
			return &hpaStatus.DesiredReplicas
		}
	}
	return resource.Replicas
}

func getHPAStatusForResource(autoscalerName string, cr *apiv2alpha1.Astarte, c client.Client, log logr.Logger) (autoscalingv2.HorizontalPodAutoscalerStatus, error) {
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: autoscalerName, Namespace: cr.Namespace}, hpa); err != nil {
		log.Info("Could not get HPA", "name", autoscalerName, "namespace", cr.Namespace)
		return autoscalingv2.HorizontalPodAutoscalerStatus{}, fmt.Errorf("not found")
	}
	return hpa.Status, nil
}

// This stuff is useful for other components which need to interact with Cassandra
func getCassandraNodes(cr *apiv2alpha1.Astarte) string {
	if cr.Spec.Cassandra.Connection == nil {
		return ""
	}

	nodes := []string{}
	for _, node := range cr.Spec.Cassandra.Connection.Nodes {
		nodes = append(nodes, fmt.Sprintf("%s:%d", node.Host, *node.Port))
	}

	return strings.Join(nodes, ",")
}

func appendAstarteKeyspaceEnvVars(cr *apiv2alpha1.Astarte) []v1.EnvVar {
	// Return empty slice if Cassandra is not configured
	if cr.Spec.Cassandra.Connection == nil {
		return []v1.EnvVar{}
	}

	ask := cr.Spec.Cassandra.AstarteSystemKeyspace

	ret := []v1.EnvVar{
		{
			Name:  "HOUSEKEEPING_ASTARTE_KEYSPACE_REPLICATION_STRATEGY",
			Value: ask.ReplicationStrategy,
		},
	}

	if ask.ReplicationStrategy == "SimpleStrategy" {
		ret = append(ret, v1.EnvVar{
			Name:  "HOUSEKEEPING_ASTARTE_KEYSPACE_REPLICATION_FACTOR",
			Value: strconv.Itoa(ask.ReplicationFactor),
		})
		return ret
	}

	// if we're here, we must handle NetworkTopologyStrategy
	theMap := make(map[string]int)

	pairs := strings.Split(ask.DataCenterReplication, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, ":")
		key := kv[0]
		// no need to check the error, this is covered in validation webhooks
		val, _ := strconv.Atoi(kv[1])
		theMap[key] = val
	}

	b, _ := json.Marshal(theMap)

	ret = append(ret, v1.EnvVar{
		Name:  "HOUSEKEEPING_ASTARTE_KEYSPACE_NETWORK_REPLICATION_MAP",
		Value: string(b),
	})
	return ret
}
