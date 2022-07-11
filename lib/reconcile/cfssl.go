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
	"encoding/json"
	"errors"
	"fmt"

	cfsslcsr "github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/deps"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
	"github.com/astarte-platform/astarte-kubernetes-operator/version"
)

// EnsureCFSSL reconciles CFSSL
func EnsureCFSSL(cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	// Validate where necessary
	if err := validateCFSSLDefinition(cr.Spec.CFSSL); err != nil {
		return err
	}

	if version.CheckConstraintAgainstAstarteVersion("< 1.0.0", cr.Spec.Version) == nil {
		// Then it's a statefulset
		return ensureCFSSLStatefulSet(cr, c, scheme)
	}

	// Otherwise, reconcile the deployment
	return ensureCFSSLDeployment(cr, c, scheme)
}

func ensureCFSSLDeployment(cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	deploymentName := cr.Name + "-cfssl"
	labels := map[string]string{"app": deploymentName}

	// Ok. Shall we deploy?
	if !pointy.BoolValue(cr.Spec.CFSSL.Deploy, true) {
		log.Info("Skipping CFSSL Deployment")
		// Before returning - check if we shall clean up the Deployment.
		// It is the only thing actually requiring resources, the rest will be cleaned up eventually when the
		// Astarte resource is deleted.
		theDeployment := &appsv1.Deployment{}
		err := c.Get(context.TODO(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, theDeployment)
		if err == nil {
			log.Info("Deleting previously existing CFSSL Deployment, which is no longer needed")
			if err = c.Delete(context.TODO(), theDeployment); err != nil {
				return err
			}
		}

		// That would be all for today.
		return nil
	}

	// Add common sidecars
	if err := ensureCFSSLCommonSidecars(deploymentName, labels, cr, c, scheme); err != nil {
		return err
	}

	caSecretName := cr.Name + "-devices-ca"
	if cr.Spec.CFSSL.CASecret.Name != "" {
		// Don't even try creating it
		caSecretName = cr.Spec.CFSSL.CASecret.Name
	} else if err := ensureCFSSLCASecret(caSecretName, cr, c, scheme); err != nil {
		return err
	}

	// Ensure the proxy secret for TLS authentication
	if err := ensureCFSSLCAProxySecret(caSecretName, cr.Name+"-cfssl-ca", cr, c, scheme); err != nil {
		return err
	}

	// Compute and prepare all data for building the StatefulSet
	deploymentSpec := appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: getCFSSLPodSpec(deploymentName, "", caSecretName, cr),
		},
	}

	// Build the Deployment
	cfsslDeployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: deploymentName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, cfsslDeployment, func() error {
		if e := controllerutil.SetControllerReference(cr, cfsslDeployment, scheme); e != nil {
			return e
		}

		// Assign the Spec.
		cfsslDeployment.Spec = deploymentSpec
		cfsslDeployment.Spec.Replicas = pointy.Int32(1)

		return nil
	})
	if err != nil {
		return err
	}

	misc.LogCreateOrUpdateOperationResult(log, result, cr, cfsslDeployment)
	return nil
}

func ensureCFSSLStatefulSet(cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	statefulSetName := cr.Name + "-cfssl"
	labels := map[string]string{"app": statefulSetName}

	// Ok. Shall we deploy?
	if !pointy.BoolValue(cr.Spec.CFSSL.Deploy, true) {
		log.Info("Skipping CFSSL Deployment")
		// Before returning - check if we shall clean up the StatefulSet.
		// It is the only thing actually requiring resources, the rest will be cleaned up eventually when the
		// Astarte resource is deleted.
		theStatefulSet := &appsv1.StatefulSet{}
		err := c.Get(context.TODO(), types.NamespacedName{Name: statefulSetName, Namespace: cr.Namespace}, theStatefulSet)
		if err == nil {
			log.Info("Deleting previously existing CFSSL StatefulSet, which is no longer needed")
			if err = c.Delete(context.TODO(), theStatefulSet); err != nil {
				return err
			}
		}

		// That would be all for today.
		return nil
	}

	// Add common sidecars
	if err := ensureCFSSLCommonSidecars(statefulSetName, labels, cr, c, scheme); err != nil {
		return err
	}

	// Let's check upon Storage now.
	dataVolumeName, persistentVolumeClaim := computePersistentVolumeClaim(statefulSetName+"-data", resource.NewScaledQuantity(4, resource.Giga),
		cr.Spec.CFSSL.Storage, cr)

	// Compute and prepare all data for building the StatefulSet
	statefulSetSpec := appsv1.StatefulSetSpec{
		ServiceName: statefulSetName,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: computePodLabels(cr.Spec.CFSSL, labels),
			},
			Spec: getCFSSLPodSpec(statefulSetName, dataVolumeName, "", cr),
		},
	}

	if persistentVolumeClaim != nil {
		statefulSetSpec.VolumeClaimTemplates = []v1.PersistentVolumeClaim{*persistentVolumeClaim}
	}

	// Build the StatefulSet
	cfsslStatefulSet := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: statefulSetName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, cfsslStatefulSet, func() error {
		if e := controllerutil.SetControllerReference(cr, cfsslStatefulSet, scheme); e != nil {
			return e
		}

		// Assign the Spec.
		cfsslStatefulSet.Spec = statefulSetSpec
		cfsslStatefulSet.Spec.Replicas = pointy.Int32(1)

		return nil
	})
	if err != nil {
		return err
	}

	misc.LogCreateOrUpdateOperationResult(log, result, cr, cfsslStatefulSet)
	return nil
}

func ensureCFSSLCommonSidecars(resourceName string, labels map[string]string, cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	// Good. Now, reconcile the service first of all.
	service := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: cr.Namespace}}
	if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, service, func() error {
		if err := controllerutil.SetControllerReference(cr, service, scheme); err != nil {
			return err
		}
		// Always set everything to what we require.
		service.Spec.Type = v1.ServiceTypeClusterIP
		service.Spec.Ports = []v1.ServicePort{
			{
				Name:       "http",
				Port:       80,
				TargetPort: intstr.FromString("http"),
				Protocol:   v1.ProtocolTCP,
			},
		}
		service.Spec.Selector = labels
		return nil
	}); err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, service)
	} else {
		return err
	}

	// Reconcile the ConfigMap
	configMap, err := getCFSSLConfigMapData(cr)
	if err != nil {
		return err
	}
	if _, e := misc.ReconcileConfigMap(resourceName+"-config", configMap, cr, c, scheme, log); e != nil {
		return e
	}

	// All good!
	return nil
}

func validateCFSSLDefinition(cfssl apiv1alpha1.AstarteCFSSLSpec) error {
	if !pointy.BoolValue(cfssl.Deploy, true) && cfssl.URL == "" {
		return errors.New("When not deploying CFSSL, the 'url' must be specified")
	}

	// All is good.
	return nil
}

func getCFSSLProbe() *v1.Probe {
	// Start checking after 10 seconds, every 20 seconds, fail after the 3rd attempt
	return &v1.Probe{
		ProbeHandler:        v1.ProbeHandler{HTTPGet: &v1.HTTPGetAction{Path: "/api/v1/cfssl/health", Port: intstr.FromString("http")}},
		InitialDelaySeconds: 10,
		TimeoutSeconds:      5,
		PeriodSeconds:       20,
		FailureThreshold:    3,
	}
}

// TODO: Deprecate dataVolumeName and all the jargon when we won't support < 1.0 anymore
func getCFSSLPodSpec(statefulSetName, dataVolumeName, secretName string, cr *apiv1alpha1.Astarte) v1.PodSpec {
	// Defaults to the custom image built in Astarte
	cfsslImage := getAstarteImageFromChannel("cfssl", deps.GetDefaultVersionForCFSSL(cr.Spec.Version), cr)
	if cr.Spec.CFSSL.Image != "" {
		cfsslImage = cr.Spec.CFSSL.Image
	} else if cr.Spec.CFSSL.Version != "" {
		cfsslImage = getAstarteImageFromChannel("cfssl", cr.Spec.CFSSL.Version, cr)
	}

	resources := v1.ResourceRequirements{}
	if cr.Spec.CFSSL.Resources != nil {
		resources = *cr.Spec.CFSSL.Resources
	}

	volumeMounts := []v1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/etc/cfssl",
		},
	}
	volumes := []v1.Volume{
		{
			Name: "config",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{Name: statefulSetName + "-config"},
				},
			},
		},
	}
	env := []v1.EnvVar{}

	if dataVolumeName != "" {
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      dataVolumeName,
			MountPath: "/data",
		})
	} else {
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      "devices-ca",
			MountPath: "/devices-ca",
			ReadOnly:  true,
		})
		volumes = append(volumes, v1.Volume{
			Name: "devices-ca",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		})
		env = append(env, v1.EnvVar{
			Name:  "KUBERNETES",
			Value: "1",
		}, v1.EnvVar{
			Name:  "CFSSL_CA_CERTIFICATE",
			Value: "/devices-ca/" + v1.TLSCertKey,
		}, v1.EnvVar{
			Name:  "CFSSL_CA_PRIVATE_KEY",
			Value: "/devices-ca/" + v1.TLSPrivateKeyKey,
		})

		if cr.Spec.CFSSL.DBConfig != nil {
			env = append(env, v1.EnvVar{
				Name:  "CFSSL_USE_DB",
				Value: "1",
			})
		}
	}

	ps := v1.PodSpec{
		TerminationGracePeriodSeconds: pointy.Int64(30),
		ImagePullSecrets:              cr.Spec.ImagePullSecrets,
		Containers: []v1.Container{
			{
				Name:            "cfssl",
				VolumeMounts:    volumeMounts,
				Env:             env,
				Image:           cfsslImage,
				ImagePullPolicy: getImagePullPolicy(cr),
				Ports: []v1.ContainerPort{
					{Name: "http", ContainerPort: 8080},
				},
				ReadinessProbe: getCFSSLProbe(),
				LivenessProbe:  getCFSSLProbe(),
				Resources:      resources,
			},
		},
		Volumes: volumes,
	}

	return ps
}

func getCFSSLConfigMapData(cr *apiv1alpha1.Astarte) (map[string]string, error) {
	// First of all, build the default maps.
	caRootConfig, csrRootCa, e := getCFSSLConfigMapDataDefaults()
	if e != nil {
		return nil, e
	}
	dbConfig, e := getCFSSLDBConfig(cr)
	if e != nil {
		return nil, e
	}

	// Now. Do we have any overrides?
	if cr.Spec.CFSSL.CaExpiry != "" {
		csrRootCa["ca"] = map[string]string{"expiry": cr.Spec.CFSSL.CaExpiry}
	}
	if cr.Spec.CFSSL.CertificateExpiry != "" {
		defaultObj := caRootConfig["signing"].(map[string]interface{})["default"].(map[string]interface{})
		defaultObj["expiry"] = cr.Spec.CFSSL.CertificateExpiry
		caRootConfig["signing"] = map[string]interface{}{"default": defaultObj}
	}

	// Are there any full overrides?
	// CSR Root CA
	if overriddenCSRRootCa, err := getCFSSLFullOverride(cr.Spec.CFSSL.CSRRootCa); err == nil && overriddenCSRRootCa != nil {
		csrRootCa = overriddenCSRRootCa
	} else if err != nil {
		return nil, err
	}
	// Root CA Configuration
	if overriddenCARootConfig, err := getCFSSLFullOverride(cr.Spec.CFSSL.CARootConfig); err == nil && overriddenCARootConfig != nil {
		caRootConfig = overriddenCARootConfig
	} else if err != nil {
		return nil, err
	}

	// All good. Now, let's convert them to JSON and set the ConfigMap data.
	dbConfigJSON, csrRootCaJSON, caRootConfigJSON, err := getCFSSLJSONFormattedConfigMapData(dbConfig, csrRootCa, caRootConfig)
	if err != nil {
		return nil, err
	}

	configMapData := map[string]string{
		"ca_root_config.json": string(caRootConfigJSON),
		"csr_root_ca.json":    string(csrRootCaJSON),
	}

	// Add DB Config only if we have it, to adhere to all CFSSL variants
	if dbConfigJSON != nil {
		configMapData["db_config.json"] = string(dbConfigJSON)
	}

	return configMapData, nil
}

func getCFSSLDBConfig(cr *apiv1alpha1.Astarte) (map[string]interface{}, error) {
	// By default, this one has to be nil...
	var dbConfig map[string]interface{}

	// ...unless we're < 1.0.0.
	if version.CheckConstraintAgainstAstarteVersion("< 1.0.0", cr.Spec.Version) == nil {
		// Then it's a statefulset
		dbConfig = map[string]interface{}{"data_source": "/data/certs.db", "driver": "sqlite3"}
	}

	// DB Configuration
	if overriddenDBConfig, err := getCFSSLFullOverride(cr.Spec.CFSSL.DBConfig); err == nil && overriddenDBConfig != nil {
		dbConfig = overriddenDBConfig
		// Switch in for retrocompatibility
		if val, ok := dbConfig["dataSource"]; ok {
			dbConfig["data_source"] = val
			delete(dbConfig, "dataSource")
		}
	} else if err != nil {
		return nil, err
	}

	return dbConfig, nil
}

func getCFSSLJSONFormattedConfigMapData(dbConfig, csrRootCa, caRootConfig map[string]interface{}) ([]byte, []byte, []byte, error) {
	// dbConfig might be as well nil
	var dbConfigJSON []byte
	if dbConfig != nil {
		var err error
		dbConfigJSON, err = json.Marshal(dbConfig)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	csrRootCaJSON, err := json.Marshal(csrRootCa)
	if err != nil {
		return nil, nil, nil, err
	}
	caRootConfigJSON, err := json.Marshal(caRootConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	return dbConfigJSON, csrRootCaJSON, caRootConfigJSON, nil
}

func getCFSSLFullOverride(field interface{}) (map[string]interface{}, error) {
	if field != nil {
		overriddenPayload, err := json.Marshal(field)
		if err != nil {
			return nil, err
		}
		newMap := make(map[string]interface{})
		if err := json.Unmarshal(overriddenPayload, &newMap); err != nil {
			return nil, err
		}
		return newMap, nil
	}

	return nil, nil
}

func getCFSSLConfigMapDataDefaults() (map[string]interface{}, map[string]interface{}, error) {
	caRootConfigDefaultJSON := `{"signing": {"default": {"usages": ["digital signature", "cert sign", "crl sign", "signing"], "ca_constraint": {"max_path_len_zero": true, "max_path_len": 0, "is_ca": true}, "expiry": "2190h"}}}`
	csrRootCaDefaultJSON := `{"ca": {"expiry": "262800h"}, "CN": "Astarte Root CA", "key": {"algo": "rsa", "size": 2048}, "names": [{"C": "IT", "ST": "Lombardia", "L": "Milan", "O": "Astarte User", "OU": "IoT Division"}]}`
	var caRootConfig, csrRootCa map[string]interface{}

	if err := json.Unmarshal([]byte(caRootConfigDefaultJSON), &caRootConfig); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal([]byte(csrRootCaDefaultJSON), &csrRootCa); err != nil {
		return nil, nil, err
	}

	return caRootConfig, csrRootCa, nil
}

// This stuff is useful for other components which need to interact with CFSSL
func getCFSSLURL(cr *apiv1alpha1.Astarte) string {
	if cr.Spec.CFSSL.URL != "" {
		return cr.Spec.CFSSL.URL
	}

	// We're on defaults then. Give the standard hostname + port for our service
	return fmt.Sprintf("http://%s-cfssl.%s.svc.cluster.local", cr.Name, cr.Namespace)
}

func ensureCFSSLCAProxySecret(secretName, proxySecretName string, cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	// Grab the real secret (the TLS one)
	s := &v1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: cr.Namespace}, s); err != nil {
		return err
	}

	// Reconcile the proxy secret with ca.crt in place of our standard key
	_, err := misc.ReconcileSecret(proxySecretName, map[string][]byte{"ca.crt": s.Data[v1.TLSCertKey]}, cr, c, scheme, log)
	return err
}

func ensureCFSSLCASecret(secretName string, cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	// We want to ensure that we create / update the secret ONLY if it doesn't exist. So check that first.
	s := &v1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: cr.Namespace}, s); err == nil {
		// A Secret exists. So we're good.
		return nil
	} else if !kerrors.IsNotFound(err) {
		// An error of different nature occurred, so report it
		return err
	}

	// If we got here, it doesn't exist. So go for it.
	return createCFSSLCASecret(secretName, cr, c, scheme)
}

func createCFSSLCASecret(secretName string, cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	// Get our configuration
	cfsslConfig, err := getCFSSLConfigMapData(cr)
	if err != nil {
		return err
	}

	// Prepare the request
	req := cfsslcsr.CertificateRequest{
		KeyRequest: cfsslcsr.NewKeyRequest(),
	}
	if e := json.Unmarshal([]byte(cfsslConfig["csr_root_ca.json"]), &req); e != nil {
		return err
	}

	cert, _, key, err := initca.New(&req)
	if err != nil {
		return err
	}

	// Reconcile our secret
	_, err = misc.ReconcileTLSSecret(secretName, string(cert), string(key), cr, c, scheme, log)
	return err
}
