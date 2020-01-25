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
	"fmt"
	"strconv"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/misc"
	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EnsureAstarteGenericAPI reconciles any component compatible with AstarteGenericAPISpec
func EnsureAstarteGenericAPI(cr *apiv1alpha1.Astarte, api apiv1alpha1.AstarteGenericAPISpec, component apiv1alpha1.AstarteComponent, c client.Client, scheme *runtime.Scheme) error {
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name, "Astarte.Component", component)
	deploymentName := cr.Name + "-" + component.DashedString()
	serviceName := cr.Name + "-" + component.ServiceName()
	labels := map[string]string{
		"app":               deploymentName,
		"component":         "astarte",
		"astarte-component": component.DashedString(),
	}
	matchLabels := map[string]string{"app": deploymentName}

	// Ok. Shall we deploy?
	if !pointy.BoolValue(api.GenericClusteredResource.Deploy, true) {
		reqLogger.V(1).Info("Skipping Astarte Component Deployment")
		// Before returning - check if we shall clean up the Deployment.
		// It is the only thing actually requiring resources, the rest will be cleaned up eventually when the
		// Astarte resource is deleted.
		theDeployment := &appsv1.Deployment{}
		err := c.Get(context.TODO(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, theDeployment)
		if err == nil {
			reqLogger.Info("Deleting previously existing Component Deployment, which is no longer needed")
			if err = c.Delete(context.TODO(), theDeployment); err != nil {
				return err
			}
		}

		// That would be all for today.
		return nil
	}

	// First of all, check if we need to regenerate the cookie.
	if err := ensureErlangCookieSecret(deploymentName+"-cookie", cr, c, scheme); err != nil {
		return err
	}

	// Good. Now, reconcile the service first of all.
	service := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: serviceName, Namespace: cr.Namespace}}
	if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, service, func() error {
		if err := controllerutil.SetControllerReference(cr, service, scheme); err != nil {
			return err
		}
		// Always set everything to what we require.
		service.ObjectMeta.Labels = labels
		service.Spec.Type = v1.ServiceTypeClusterIP
		service.Spec.ClusterIP = "None"
		service.Spec.Ports = []v1.ServicePort{
			v1.ServicePort{
				Name:       "http",
				Port:       4000,
				TargetPort: intstr.FromString("http"),
				Protocol:   v1.ProtocolTCP,
			},
		}
		service.Spec.Selector = matchLabels
		return nil
	}); err == nil {
		logCreateOrUpdateOperationResult(result, cr, service)
	} else {
		return err
	}

	deploymentSpec := appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: matchLabels,
		},
		Strategy: getDeploymentStrategyForClusteredResource(cr, api.GenericClusteredResource),
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: getAstarteGenericAPIPodSpec(deploymentName, cr, api, component),
		},
	}

	// Build the Deployment
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: deploymentName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, deployment, func() error {
		if err := controllerutil.SetControllerReference(cr, deployment, scheme); err != nil {
			return err
		}

		// Assign the Spec.
		deployment.ObjectMeta.Labels = labels
		deployment.Spec = deploymentSpec
		deployment.Spec.Replicas = api.GenericClusteredResource.Replicas

		return nil
	})
	if err != nil {
		return err
	}

	logCreateOrUpdateOperationResult(result, cr, deployment)
	return nil
}

func getAstarteGenericAPIPodSpec(deploymentName string, cr *apiv1alpha1.Astarte, api apiv1alpha1.AstarteGenericAPISpec, component apiv1alpha1.AstarteComponent) v1.PodSpec {
	ps := v1.PodSpec{
		TerminationGracePeriodSeconds: pointy.Int64(30),
		ImagePullSecrets:              cr.Spec.ImagePullSecrets,
		Affinity:                      getAffinityForClusteredResource(deploymentName, api.GenericClusteredResource),
		Containers: []v1.Container{
			v1.Container{
				Name: component.DashedString(),
				Ports: []v1.ContainerPort{
					v1.ContainerPort{Name: "http", ContainerPort: 4000},
				},
				VolumeMounts:    getAstarteGenericAPIVolumeMounts(deploymentName, cr, api, component),
				Image:           getAstarteImageForClusteredResource(component.DockerImageName(), api.GenericClusteredResource, cr),
				ImagePullPolicy: getImagePullPolicy(cr),
				Resources:       misc.GetResourcesForAstarteComponent(cr, api.GenericClusteredResource.Resources, component),
				Env:             getAstarteGenericAPIEnvVars(deploymentName, cr, api, component),
				ReadinessProbe:  getAstarteAPIProbe(cr, api, component),
				LivenessProbe:   getAstarteAPIProbe(cr, api, component),
			},
		},
		Volumes: getAstarteGenericAPIVolumes(deploymentName, cr, api, component),
	}

	return ps
}

func getAstarteGenericAPIVolumes(deploymentName string, cr *apiv1alpha1.Astarte, api apiv1alpha1.AstarteGenericAPISpec, component apiv1alpha1.AstarteComponent) []v1.Volume {
	ret := getAstarteCommonVolumes(cr)

	// Depending on the component, we might need to add some more stuff.
	switch component {
	case apiv1alpha1.HousekeepingAPI:
		ret = append(ret, v1.Volume{
			Name: "jwtpubkey",
			VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{
				SecretName: fmt.Sprintf("%s-housekeeping-public-key", cr.Name),
			}},
		})
	}

	return ret
}

func getAstarteGenericAPIVolumeMounts(deploymentName string, cr *apiv1alpha1.Astarte, api apiv1alpha1.AstarteGenericAPISpec, component apiv1alpha1.AstarteComponent) []v1.VolumeMount {
	ret := getAstarteCommonVolumeMounts()

	// Depending on the component, we might need to add some more stuff.
	switch component {
	case apiv1alpha1.HousekeepingAPI:
		ret = append(ret, v1.VolumeMount{
			Name:      "jwtpubkey",
			MountPath: "/jwtpubkey",
			ReadOnly:  true,
		})
	}

	return ret
}

func getAstarteGenericAPIEnvVars(deploymentName string, cr *apiv1alpha1.Astarte, api apiv1alpha1.AstarteGenericAPISpec, component apiv1alpha1.AstarteComponent) []v1.EnvVar {
	ret := getAstarteCommonEnvVars(deploymentName, cr, component)
	// Add Port
	ret = append(ret, v1.EnvVar{
		Name:  strings.ToUpper(component.String()) + "_PORT",
		Value: "4000",
	})

	// Should we disable authentication?
	if pointy.BoolValue(api.DisableAuthentication, false) {
		ret = append(ret, v1.EnvVar{
			Name:  strings.ToUpper(component.String()) + "_DISABLE_AUTHENTICATION",
			Value: "true",
		})
	}

	// Depending on the component, we might need to add some more stuff.
	switch component {
	case apiv1alpha1.AppEngineAPI:
		// Add Cassandra Nodes, AMQP information and Max results count
		rabbitMQHost, rabbitMQPort := misc.GetRabbitMQHostnameAndPort(cr)
		userCredentialsSecretName, userCredentialsSecretUsernameKey, userCredentialsSecretPasswordKey := misc.GetRabbitMQUserCredentialsSecret(cr)
		ret = append(ret,
			v1.EnvVar{
				Name:  "ASTARTE_CASSANDRA_NODES",
				Value: getCassandraNodes(cr),
			},
			v1.EnvVar{
				Name:  "APPENGINE_API_MAX_RESULTS_LIMIT",
				Value: strconv.Itoa(getAppEngineAPIMaxResultslimit(cr)),
			},
			v1.EnvVar{
				Name:  "APPENGINE_API_ROOMS_AMQP_CLIENT_HOST",
				Value: rabbitMQHost,
			},
			v1.EnvVar{
				Name:  "APPENGINE_API_ROOMS_AMQP_CLIENT_PORT",
				Value: strconv.Itoa(int(rabbitMQPort)),
			},
			v1.EnvVar{
				Name: "APPENGINE_API_ROOMS_AMQP_CLIENT_USERNAME",
				ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
					Key:                  userCredentialsSecretUsernameKey,
				}},
			},
			v1.EnvVar{
				Name: "APPENGINE_API_ROOMS_AMQP_CLIENT_PASSWORD",
				ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
					Key:                  userCredentialsSecretPasswordKey,
				}},
			},
		)
	case apiv1alpha1.HousekeepingAPI:
		// Add Public Key Information
		ret = append(ret, v1.EnvVar{
			Name:  "HOUSEKEEPING_API_JWT_PUBLIC_KEY_PATH",
			Value: "/jwtpubkey/public-key",
		})
	}

	return ret
}

func getAstarteAPIProbe(cr *apiv1alpha1.Astarte, api apiv1alpha1.AstarteGenericAPISpec, component apiv1alpha1.AstarteComponent) *v1.Probe {
	// Parse the version first
	v := getSemanticVersionForAstarteComponent(cr, api.GenericClusteredResource.Version)
	checkVersion, _ := v.SetPrerelease("")
	constraint, _ := semver.NewConstraint("< 0.11.0")

	if constraint.Check(&checkVersion) {
		// Only Housekeeping has a health check in 0.10.x
		if component != apiv1alpha1.HousekeepingAPI {
			return nil
		}

		return getAstarteAPIGenericProbe("/v1/health")
	}

	// 0.11, up to Beta 2, doesn't have Realm Management and Pairing API probes
	constraint, _ = semver.NewConstraint("<= 0.11.0-beta.2")
	if constraint.Check(v) {
		if component == apiv1alpha1.RealmManagementAPI || component == apiv1alpha1.PairingAPI {
			return nil
		}
	}

	// The rest are generic probes on /health
	return getAstarteAPIGenericProbe("/health")
}

func getAstarteAPIGenericProbe(path string) *v1.Probe {
	return &v1.Probe{
		Handler: v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Path: path,
				Port: intstr.FromString("http"),
			},
		},
		InitialDelaySeconds: 10,
		TimeoutSeconds:      5,
		PeriodSeconds:       30,
		FailureThreshold:    5,
	}
}
