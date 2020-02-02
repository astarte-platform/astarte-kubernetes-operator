/*
  This file is part of Astarte.

  Copyright 2020 Ispirata Srl

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
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strconv"

	semver "github.com/Masterminds/semver/v3"
	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/misc"
	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func encodePEMBlockToEncodedBytes(block *pem.Block) []byte {
	return []byte(base64.StdEncoding.EncodeToString(pem.EncodeToMemory(block)))
}

func storePublicKeyInSecret(name string, publicKey *rsa.PublicKey, cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	pkixBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	var publicKeyPEM = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pkixBytes,
	}

	publicKeySecretData := encodePEMBlockToEncodedBytes(publicKeyPEM)

	secretData := map[string][]byte{
		"public-key": publicKeySecretData,
	}

	// Set Astarte instance as the owner and controller
	_, err = reconcileSecret(name, secretData, cr, c, scheme)
	return err
}

func storePrivateKeyInSecret(name string, privateKey *rsa.PrivateKey, cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	var privateKeyPEM = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeySecretData := encodePEMBlockToEncodedBytes(privateKeyPEM)
	fmt.Printf("aaaaaaa %v\n", privateKeySecretData)

	secretData := map[string][]byte{
		"private-key": privateKeySecretData,
	}

	// Set Astarte instance as the owner and controller
	_, err := reconcileSecret(name, secretData, cr, c, scheme)
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
				v1.PodAffinityTerm{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							metav1.LabelSelectorRequirement{
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

func reconcileConfigMap(objName string, data map[string]string, cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) (controllerutil.OperationResult, error) {
	return misc.ReconcileConfigMap(objName, data, cr, c, scheme, log)
}

func reconcileSecret(objName string, data map[string][]byte, cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) (controllerutil.OperationResult, error) {
	return misc.ReconcileSecret(objName, data, cr, c, scheme, log)
}

func reconcileStandardRBACForClusteringForApp(name string, policyRules []rbacv1.PolicyRule, cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	// Service Account
	serviceAccount := &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cr.Namespace}}
	if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, serviceAccount, func() error {
		if err := controllerutil.SetControllerReference(cr, serviceAccount, scheme); err != nil {
			return err
		}
		// Actually nothing to do here.
		return nil
	}); err == nil {
		logCreateOrUpdateOperationResult(result, cr, serviceAccount)
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
		logCreateOrUpdateOperationResult(result, cr, serviceAccount)
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
			rbacv1.Subject{
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
		logCreateOrUpdateOperationResult(result, cr, serviceAccount)
	} else {
		return err
	}

	return nil
}

func logCreateOrUpdateOperationResult(result controllerutil.OperationResult, cr *apiv1alpha1.Astarte, obj metav1.Object) {
	misc.LogCreateOrUpdateOperationResult(log, result, cr, obj)
}

func getAstarteImageFromChannel(name, tag string, cr *apiv1alpha1.Astarte) string {
	distributionChannel := "astarte"
	if cr.Spec.DistributionChannel != "" {
		distributionChannel = cr.Spec.DistributionChannel
	}

	return fmt.Sprintf("%s/%s:%s", distributionChannel, name, tag)
}

func getImagePullPolicy(cr *apiv1alpha1.Astarte) v1.PullPolicy {
	if cr.Spec.ImagePullPolicy != nil {
		return *cr.Spec.ImagePullPolicy
	}

	return v1.PullIfNotPresent
}

func ensureErlangCookieSecret(secretName string, cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	theCookie := &v1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: cr.Namespace}, theCookie); err != nil {
		if kerrors.IsNotFound(err) {
			// Create it.
			// TODO: Throw a reconcile error and/or delete the persistent volume if we are in that situation.
			reqLogger.Info("Creating new Cookie", "cookie-name", secretName)
			cookie := make([]byte, 32)
			rand.Read(cookie)

			cookieSecret := v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: cr.Namespace,
				},
				StringData: map[string]string{"erlang-cookie": base32.StdEncoding.EncodeToString(cookie)},
			}
			if err := controllerutil.SetControllerReference(cr, &cookieSecret, scheme); err != nil {
				return err
			}
			// We force creation as for no reason in the world we want to even think about updating this.
			if err = c.Create(context.TODO(), &cookieSecret); err != nil {
				return err
			}
		} else {
			// Return here
			return err
		}
	}

	// All went well
	return nil
}

func computePersistentVolumeClaim(defaultName string, defaultSize *resource.Quantity, storageSpec *apiv1alpha1.AstartePersistentStorageSpec,
	cr *apiv1alpha1.Astarte) (string, *v1.PersistentVolumeClaim) {
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
		Resources:   v1.ResourceRequirements{Requests: v1.ResourceList{v1.ResourceStorage: *dataVolumeSize}},
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

func getAstarteCommonEnvVars(deploymentName string, cr *apiv1alpha1.Astarte, component apiv1alpha1.AstarteComponent) []v1.EnvVar {
	rabbitMQHost, rabbitMQPort := misc.GetRabbitMQHostnameAndPort(cr)
	userCredentialsSecretName, userCredentialsSecretUsernameKey, userCredentialsSecretPasswordKey := misc.GetRabbitMQUserCredentialsSecret(cr)
	ret := []v1.EnvVar{
		v1.EnvVar{
			Name:  "RELEASE_CONFIG_DIR",
			Value: "/beamconfig",
		},
		v1.EnvVar{
			Name:  "REPLACE_OS_VARS",
			Value: "true",
		},
		v1.EnvVar{
			Name:      "MY_POD_IP",
			ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "status.podIP"}},
		},
		v1.EnvVar{
			Name:  "RELEASE_NAME",
			Value: component.DockerImageName(),
		},
		v1.EnvVar{
			Name: "ERLANG_COOKIE",
			ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: deploymentName + "-cookie"},
				Key:                  "erlang-cookie",
			}},
		},
		v1.EnvVar{
			Name:  "ASTARTE_RPC_AMQP_CONNECTION_HOST",
			Value: rabbitMQHost,
		},
		v1.EnvVar{
			Name:  "ASTARTE_RPC_AMQP_CONNECTION_PORT",
			Value: strconv.Itoa(int(rabbitMQPort)),
		},
		v1.EnvVar{
			Name: "ASTARTE_RPC_AMQP_CONNECTION_USERNAME",
			ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
				Key:                  userCredentialsSecretUsernameKey,
			}},
		},
		v1.EnvVar{
			Name: "ASTARTE_RPC_AMQP_CONNECTION_PASSWORD",
			ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
				Key:                  userCredentialsSecretPasswordKey,
			}},
		},
	}

	return ret
}

func getAstarteCommonVolumes(cr *apiv1alpha1.Astarte) []v1.Volume {
	ret := []v1.Volume{
		v1.Volume{
			Name: "beam-config",
			VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{Name: fmt.Sprintf("%s-generic-erlang-configuration", cr.Name)},
				Items:                []v1.KeyToPath{v1.KeyToPath{Key: "vm.args", Path: "vm.args"}},
			}},
		},
	}

	return ret
}

func getVersionForAstarteComponent(cr *apiv1alpha1.Astarte, componentVersion string) string {
	if componentVersion != "" {
		return componentVersion
	}
	return cr.Spec.Version
}

func getSemanticVersionForAstarteComponent(cr *apiv1alpha1.Astarte, componentVersion string) *semver.Version {
	semVer, _ := semver.NewVersion(getVersionForAstarteComponent(cr, componentVersion))
	return semVer
}

func getAstarteCommonVolumeMounts() []v1.VolumeMount {
	ret := []v1.VolumeMount{
		v1.VolumeMount{
			Name:      "beam-config",
			MountPath: "/beamconfig",
			ReadOnly:  true,
		},
	}

	return ret
}

func getAffinityForClusteredResource(appLabel string, resource apiv1alpha1.AstarteGenericClusteredResource) *v1.Affinity {
	affinity := resource.CustomAffinity
	if affinity == nil && pointy.BoolValue(resource.AntiAffinity, true) {
		affinity = getStandardAntiAffinityForAppLabel(appLabel)
	}
	return affinity
}

func getAstarteImageForClusteredResource(defaultImageName string, resource apiv1alpha1.AstarteGenericClusteredResource, cr *apiv1alpha1.Astarte) string {
	if resource.Image != "" {
		return resource.Image
	}

	return getAstarteImageFromChannel(defaultImageName, getVersionForAstarteComponent(cr, resource.Version), cr)
}

func getImageForClusteredResource(defaultImageName, defaultImageTag string, resource apiv1alpha1.AstarteGenericClusteredResource) string {
	image := fmt.Sprintf("%s:%s", defaultImageName, defaultImageTag)
	if resource.Image != "" {
		image = resource.Image
	} else if resource.Version != "" {
		image = fmt.Sprintf("%s:%s", defaultImageName, resource.Version)
	}

	return image
}

func getDeploymentStrategyForClusteredResource(cr *apiv1alpha1.Astarte, resource apiv1alpha1.AstarteGenericClusteredResource) appsv1.DeploymentStrategy {
	if resource.DeploymentStrategy != nil {
		return *resource.DeploymentStrategy
	}
	return cr.Spec.DeploymentStrategy
}

func getDataQueueCount(cr *apiv1alpha1.Astarte) int {
	return pointy.IntValue(cr.Spec.Components.DataUpdaterPlant.DataQueueCount, 128)
}

func getAppEngineAPIMaxResultslimit(cr *apiv1alpha1.Astarte) int {
	return pointy.IntValue(cr.Spec.Components.AppengineAPI.MaxResultsLimit, 10000)
}

func getBaseAstarteAPIURL(cr *apiv1alpha1.Astarte) string {
	scheme := "https"
	if !pointy.BoolValue(cr.Spec.API.SSL, true) {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s", scheme, cr.Spec.API.Host)
}
