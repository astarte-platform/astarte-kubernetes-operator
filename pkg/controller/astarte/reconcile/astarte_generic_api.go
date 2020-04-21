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

// EnsureAstarteGenericAPI reconciles any component compatible with AstarteGenericAPISpec with a custom Probe
func EnsureAstarteGenericAPI(cr *apiv1alpha1.Astarte, api apiv1alpha1.AstarteGenericAPISpec, component apiv1alpha1.AstarteComponent,
	c client.Client, scheme *runtime.Scheme) error {
	return EnsureAstarteGenericAPIWithCustomProbe(cr, api, component, c, scheme, nil)
}

// EnsureAstarteGenericAPIWithCustomProbe reconciles any component compatible with AstarteGenericAPISpec with a custom Probe
func EnsureAstarteGenericAPIWithCustomProbe(cr *apiv1alpha1.Astarte, api apiv1alpha1.AstarteGenericAPISpec, component apiv1alpha1.AstarteComponent,
	c client.Client, scheme *runtime.Scheme, customProbe *v1.Probe) error {
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
	if !pointy.BoolValue(api.Deploy, true) {
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
	}); err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, service)
	} else {
		return err
	}

	deploymentSpec := appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: matchLabels,
		},
		Strategy: getDeploymentStrategyForClusteredResource(cr, api.AstarteGenericClusteredResource),
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: getAstarteGenericAPIPodSpec(deploymentName, cr, api, component, customProbe),
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
		deployment.Spec.Replicas = api.Replicas

		return nil
	})
	if err != nil {
		return err
	}

	misc.LogCreateOrUpdateOperationResult(log, result, cr, deployment)
	return nil
}

func getAstarteGenericAPIPodSpec(deploymentName string, cr *apiv1alpha1.Astarte, api apiv1alpha1.AstarteGenericAPISpec,
	component apiv1alpha1.AstarteComponent, customProbe *v1.Probe) v1.PodSpec {
	ps := v1.PodSpec{
		TerminationGracePeriodSeconds: pointy.Int64(30),
		ImagePullSecrets:              cr.Spec.ImagePullSecrets,
		Affinity:                      getAffinityForClusteredResource(deploymentName, api.AstarteGenericClusteredResource),
		Containers: []v1.Container{
			{
				Name: component.DashedString(),
				Ports: []v1.ContainerPort{
					{Name: "http", ContainerPort: astarteServicesPort},
				},
				VolumeMounts:    getAstarteGenericAPIVolumeMounts(component),
				Image:           getAstarteImageForClusteredResource(component.DockerImageName(), api.AstarteGenericClusteredResource, cr),
				ImagePullPolicy: getImagePullPolicy(cr),
				Resources:       misc.GetResourcesForAstarteComponent(cr, api.Resources, component),
				Env:             getAstarteGenericAPIEnvVars(deploymentName, cr, api, component),
				ReadinessProbe:  getAstarteAPIProbe(cr, api, component, customProbe),
				LivenessProbe:   getAstarteAPIProbe(cr, api, component, customProbe),
			},
		},
		Volumes: getAstarteGenericAPIVolumes(cr, component),
	}

	return ps
}

func getAstarteGenericAPIVolumes(cr *apiv1alpha1.Astarte, component apiv1alpha1.AstarteComponent) []v1.Volume {
	ret := getAstarteCommonVolumes(cr)

	// Depending on the component, we might need to add some more stuff.
	if component == apiv1alpha1.HousekeepingAPI {
		ret = append(ret, v1.Volume{
			Name: "jwtpubkey",
			VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{
				SecretName: fmt.Sprintf("%s-housekeeping-public-key", cr.Name),
			}},
		})
	}

	return ret
}

func getAstarteGenericAPIVolumeMounts(component apiv1alpha1.AstarteComponent) []v1.VolumeMount {
	ret := getAstarteCommonVolumeMounts()

	// Depending on the component, we might need to add some more stuff.
	if component == apiv1alpha1.HousekeepingAPI {
		ret = append(ret, v1.VolumeMount{
			Name:      "jwtpubkey",
			MountPath: "/jwtpubkey",
			ReadOnly:  true,
		})
	}

	return ret
}

func getAstarteGenericAPIEnvVars(deploymentName string, cr *apiv1alpha1.Astarte, api apiv1alpha1.AstarteGenericAPISpec, component apiv1alpha1.AstarteComponent) []v1.EnvVar {
	ret := getAstarteCommonEnvVars(deploymentName, cr, api.AstarteGenericClusteredResource, component)

	// Should we disable authentication?
	if pointy.BoolValue(api.DisableAuthentication, false) {
		ret = append(ret, v1.EnvVar{
			Name:  strings.ToUpper(component.String()) + "_DISABLE_AUTHENTICATION",
			Value: strconv.FormatBool(true),
		})
	}

	// Depending on the component, we might need to add some more stuff.
	switch component {
	case apiv1alpha1.AppEngineAPI:
		// Add Cassandra Nodes, AMQP information and Max results count
		rabbitMQHost, rabbitMQPort := misc.GetRabbitMQHostnameAndPort(cr)
		userCredentialsSecretName, userCredentialsSecretUsernameKey, userCredentialsSecretPasswordKey := misc.GetRabbitMQUserCredentialsSecret(cr)

		cassandraPrefix := ""
		v := getSemanticVersionForAstarteComponent(cr, cr.Spec.Components.AppengineAPI.Version)
		checkVersion, _ := v.SetPrerelease("")
		constraint, _ := semver.NewConstraint("< 1.0.0")
		if constraint.Check(&checkVersion) {
			cassandraPrefix = oldAstartePrefix
		}

		// Add Cassandra Nodes
		ret = append(ret, v1.EnvVar{
			Name:  cassandraPrefix + "CASSANDRA_NODES",
			Value: getCassandraNodes(cr),
		})

		ret = append(ret,
			v1.EnvVar{
				Name:  cassandraPrefix + "CASSANDRA_NODES",
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

		if cr.Spec.RabbitMQ.Connection != nil {
			if cr.Spec.RabbitMQ.Connection.VirtualHost != "" {
				ret = append(ret,
					v1.EnvVar{
						Name:  "APPENGINE_API_ROOMS_AMQP_CLIENT_VIRTUAL_HOST",
						Value: cr.Spec.RabbitMQ.Connection.VirtualHost,
					})
			}
		}

		if cr.Spec.Components.AppengineAPI.RoomEventsQueueName != "" {
			ret = append(ret,
				v1.EnvVar{
					Name:  "APPENGINE_API_ROOMS_EVENTS_QUEUE_NAME",
					Value: cr.Spec.Components.AppengineAPI.RoomEventsQueueName,
				})
		}

		if cr.Spec.Components.AppengineAPI.RoomEventsExchangeName != "" {
			ret = append(ret,
				v1.EnvVar{
					Name:  "APPENGINE_API_ROOMS_EVENTS_EXCHANGE_NAME",
					Value: cr.Spec.Components.AppengineAPI.RoomEventsExchangeName,
				})
		}
	case apiv1alpha1.HousekeepingAPI:
		// Add Public Key Information
		ret = append(ret, v1.EnvVar{
			Name:  "HOUSEKEEPING_API_JWT_PUBLIC_KEY_PATH",
			Value: "/jwtpubkey/public-key",
		})
	}

	return ret
}

func getAstarteAPIProbe(cr *apiv1alpha1.Astarte, api apiv1alpha1.AstarteGenericAPISpec, component apiv1alpha1.AstarteComponent, customProbe *v1.Probe) *v1.Probe {
	if customProbe != nil {
		return customProbe
	}

	// Parse the version first
	v := getSemanticVersionForAstarteComponent(cr, api.Version)
	checkVersion, _ := v.SetPrerelease("")
	constraint, _ := semver.NewConstraint("< 0.11.0")

	if constraint.Check(&checkVersion) {
		// Only Housekeeping has a health check in 0.10.x
		if component != apiv1alpha1.HousekeepingAPI {
			return nil
		}

		return getAstarteAPIGenericProbe("/v1/health")
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
