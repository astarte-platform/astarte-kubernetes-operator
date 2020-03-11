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

	semver "github.com/Masterminds/semver/v3"
	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/controller/astarte/deps"
	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EnsureCFSSL reconciles CFSSL
func EnsureCFSSL(cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	//reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	statefulSetName := cr.Name + "-cfssl"
	labels := map[string]string{"app": statefulSetName}

	// Validate where necessary
	if err := validateCFSSLDefinition(cr.Spec.CFSSL); err != nil {
		return err
	}

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

	// Good. Now, reconcile the service first of all.
	service := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: statefulSetName, Namespace: cr.Namespace}}
	if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, service, func() error {
		if err := controllerutil.SetControllerReference(cr, service, scheme); err != nil {
			return err
		}
		// Always set everything to what we require.
		service.Spec.Type = v1.ServiceTypeClusterIP
		service.Spec.Ports = []v1.ServicePort{
			v1.ServicePort{
				Name:       "http",
				Port:       80,
				TargetPort: intstr.FromString("http"),
				Protocol:   v1.ProtocolTCP,
			},
		}
		service.Spec.Selector = labels
		return nil
	}); err == nil {
		logCreateOrUpdateOperationResult(result, cr, service)
	} else {
		return err
	}

	// Reconcile the ConfigMap
	configMap, err := getCFSSLConfigMapData(statefulSetName, cr)
	if err != nil {
		return err
	}
	if _, err := reconcileConfigMap(statefulSetName+"-config", configMap, cr, c, scheme); err != nil {
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
				Labels: labels,
			},
			Spec: getCFSSLPodSpec(statefulSetName, dataVolumeName, cr),
		},
	}

	if persistentVolumeClaim != nil {
		statefulSetSpec.VolumeClaimTemplates = []v1.PersistentVolumeClaim{*persistentVolumeClaim}
	}

	// Build the StatefulSet
	cfsslStatefulSet := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: statefulSetName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, cfsslStatefulSet, func() error {
		if err := controllerutil.SetControllerReference(cr, cfsslStatefulSet, scheme); err != nil {
			return err
		}

		// Assign the Spec.
		cfsslStatefulSet.Spec = statefulSetSpec
		cfsslStatefulSet.Spec.Replicas = pointy.Int32(1)

		return nil
	})
	if err != nil {
		return err
	}

	logCreateOrUpdateOperationResult(result, cr, service)
	return nil
}

func validateCFSSLDefinition(cfssl apiv1alpha1.AstarteCFSSLSpec) error {
	if !pointy.BoolValue(cfssl.Deploy, true) && cfssl.URL == "" {
		return errors.New("When not deploying CFSSL, the 'url' must be specified")
	}

	// All is good.
	return nil
}

func getCFSSLProbe(cr *apiv1alpha1.Astarte) *v1.Probe {
	c, _ := semver.NewConstraint("< 0.11.0")
	ver, _ := semver.NewVersion(getVersionForAstarteComponent(cr, cr.Spec.CFSSL.Version))
	checkVersion, _ := ver.SetPrerelease("")

	// HTTP Health is supported only from 0.11 on
	if c.Check(&checkVersion) {
		return &v1.Probe{
			Handler:             v1.Handler{TCPSocket: &v1.TCPSocketAction{Port: intstr.FromString("http")}},
			InitialDelaySeconds: 10,
			TimeoutSeconds:      5,
			PeriodSeconds:       30,
			FailureThreshold:    3,
		}
	}

	// Start checking after 10 seconds, every 20 seconds, fail after the 3rd attempt
	return &v1.Probe{
		Handler:             v1.Handler{HTTPGet: &v1.HTTPGetAction{Path: "/api/v1/cfssl/health", Port: intstr.FromString("http")}},
		InitialDelaySeconds: 10,
		TimeoutSeconds:      5,
		PeriodSeconds:       20,
		FailureThreshold:    3,
	}
}

func getCFSSLPodSpec(statefulSetName, dataVolumeName string, cr *apiv1alpha1.Astarte) v1.PodSpec {
	// Defaults to the custom image built in Astarte
	astarteVersion, _ := semver.NewVersion(cr.Spec.Version)
	cfsslImage := getAstarteImageFromChannel("cfssl", deps.GetDefaultVersionForCFSSL(astarteVersion), cr)
	if cr.Spec.CFSSL.Image != "" {
		cfsslImage = cr.Spec.CFSSL.Image
	} else if cr.Spec.CFSSL.Version != "" {
		cfsslImage = getAstarteImageFromChannel("cfssl", cr.Spec.CFSSL.Version, cr)
	}

	resources := v1.ResourceRequirements{}
	if cr.Spec.CFSSL.Resources != nil {
		resources = *cr.Spec.CFSSL.Resources
	}

	ps := v1.PodSpec{
		TerminationGracePeriodSeconds: pointy.Int64(30),
		ImagePullSecrets:              cr.Spec.ImagePullSecrets,
		Containers: []v1.Container{
			v1.Container{
				Name: "cfssl",
				VolumeMounts: []v1.VolumeMount{
					v1.VolumeMount{
						Name:      "config",
						MountPath: "/etc/cfssl",
					},
					v1.VolumeMount{
						Name:      dataVolumeName,
						MountPath: "/data",
					},
				},
				Image:           cfsslImage,
				ImagePullPolicy: getImagePullPolicy(cr),
				Ports: []v1.ContainerPort{
					v1.ContainerPort{Name: "http", ContainerPort: 8080},
				},
				ReadinessProbe: getCFSSLProbe(cr),
				LivenessProbe:  getCFSSLProbe(cr),
				Resources:      resources,
			},
		},
		Volumes: []v1.Volume{
			v1.Volume{
				Name: "config",
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{Name: statefulSetName + "-config"},
					},
				},
			},
		},
	}

	return ps
}

func getCFSSLConfigMapData(statefulSetName string, cr *apiv1alpha1.Astarte) (map[string]string, error) {
	// First of all, build the default maps.
	dbConfigDefaultJSON := `{"data_source": "/data/certs.db", "driver": "sqlite3"}`
	caRootConfigDefaultJSON := `{"signing": {"default": {"usages": ["digital signature", "cert sign", "crl sign", "signing"], "ca_constraint": {"max_path_len_zero": true, "max_path_len": 0, "is_ca": true}, "expiry": "2190h"}}}`
	csrRootCaDefaultJSON := `{"ca": {"expiry": "262800h"}, "CN": "Astarte Root CA", "key": {"algo": "rsa", "size": 2048}, "names": [{"C": "IT", "ST": "Lombardia", "L": "Milan", "O": "Astarte User", "OU": "IoT Division"}]}`
	var dbConfig, caRootConfig, csrRootCa map[string]interface{}

	if err := json.Unmarshal([]byte(dbConfigDefaultJSON), &dbConfig); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(caRootConfigDefaultJSON), &caRootConfig); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(csrRootCaDefaultJSON), &csrRootCa); err != nil {
		return nil, err
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
	if cr.Spec.CFSSL.DBConfig != nil {
		overriddenDBConfig, err := json.Marshal(cr.Spec.CFSSL.DBConfig)
		if err != nil {
			return nil, err
		}
		dbConfig = make(map[string]interface{})
		if err := json.Unmarshal([]byte(overriddenDBConfig), &dbConfig); err != nil {
			return nil, err
		}
		// Switch in for retrocompatibility
		dbConfig["data_source"] = dbConfig["dataSource"]
		delete(dbConfig, "dataSource")
	}
	if cr.Spec.CFSSL.CSRRootCa != nil {
		overriddenCSRRootCa, err := json.Marshal(cr.Spec.CFSSL.CSRRootCa)
		if err != nil {
			return nil, err
		}
		csrRootCa = make(map[string]interface{})
		if err := json.Unmarshal([]byte(overriddenCSRRootCa), &csrRootCa); err != nil {
			return nil, err
		}
	}
	if cr.Spec.CFSSL.CARootConfig != nil {
		overriddenCARootConfig, err := json.Marshal(cr.Spec.CFSSL.CARootConfig)
		if err != nil {
			return nil, err
		}
		caRootConfig = make(map[string]interface{})
		if err := json.Unmarshal([]byte(overriddenCARootConfig), &caRootConfig); err != nil {
			return nil, err
		}
	}

	// All good. Now, let's convert them to JSON and set the ConfigMap data.
	dbConfigJSON, err := json.Marshal(dbConfig)
	if err != nil {
		return nil, err
	}
	csrRootCaJSON, err := json.Marshal(csrRootCa)
	if err != nil {
		return nil, err
	}
	caRootConfigJSON, err := json.Marshal(caRootConfig)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"ca_root_config.json": string(caRootConfigJSON),
		"csr_root_ca.json":    string(csrRootCaJSON),
		"db_config.json":      string(dbConfigJSON),
	}, nil
}

// This stuff is useful for other components which need to interact with CFSSL
func getCFSSLURL(cr *apiv1alpha1.Astarte) string {
	if cr.Spec.CFSSL.URL != "" {
		return cr.Spec.CFSSL.URL
	}

	// We're on defaults then. Give the standard hostname + port for our service
	return fmt.Sprintf("http://%s-cfssl.%s.svc.cluster.local", cr.Name, cr.Namespace)
}
