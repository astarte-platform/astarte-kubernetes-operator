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
	"strconv"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/misc"
	"github.com/astarte-platform/astarte-kubernetes-operator/version"
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

// EnsureAstarteGenericBackend reconciles any component compatible with AstarteGenericClusteredResource
func EnsureAstarteGenericBackend(cr *apiv1alpha1.Astarte, backend apiv1alpha1.AstarteGenericClusteredResource, component apiv1alpha1.AstarteComponent,
	c client.Client, scheme *runtime.Scheme) error {
	return EnsureAstarteGenericBackendWithCustomProbe(cr, backend, component, c, scheme, nil)
}

// EnsureAstarteGenericBackendWithCustomProbe reconciles any component compatible with AstarteGenericClusteredResource adding a custom probe
func EnsureAstarteGenericBackendWithCustomProbe(cr *apiv1alpha1.Astarte, backend apiv1alpha1.AstarteGenericClusteredResource,
	component apiv1alpha1.AstarteComponent, c client.Client, scheme *runtime.Scheme, customProbe *v1.Probe) error {
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
	if !pointy.BoolValue(backend.Deploy, true) {
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
	if err := createOrUpdateService(cr, c, serviceName, scheme, matchLabels, labels); err != nil {
		return err
	}

	deploymentSpec := appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: matchLabels,
		},
		Strategy: getDeploymentStrategyForClusteredResource(cr, backend),
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: getAstarteGenericBackendPodSpec(deploymentName, cr, backend, component, customProbe),
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
		deployment.Spec.Replicas = backend.Replicas

		return nil
	})
	if err != nil {
		return err
	}

	misc.LogCreateOrUpdateOperationResult(log, result, cr, deployment)
	return nil
}

func getAstarteGenericBackendPodSpec(deploymentName string, cr *apiv1alpha1.Astarte, backend apiv1alpha1.AstarteGenericClusteredResource,
	component apiv1alpha1.AstarteComponent, customProbe *v1.Probe) v1.PodSpec {
	ps := v1.PodSpec{
		TerminationGracePeriodSeconds: pointy.Int64(30),
		ImagePullSecrets:              cr.Spec.ImagePullSecrets,
		Affinity:                      getAffinityForClusteredResource(deploymentName, backend),
		Containers: []v1.Container{
			{
				Name: component.DashedString(),
				Ports: []v1.ContainerPort{
					// This port is not exposed through any service - it is just used for health checks and the likes.
					{Name: "http", ContainerPort: astarteServicesPort},
				},
				VolumeMounts:    getAstarteGenericBackendVolumeMounts(cr),
				Image:           getAstarteImageForClusteredResource(component.DockerImageName(), backend, cr),
				ImagePullPolicy: getImagePullPolicy(cr),
				Resources:       misc.GetResourcesForAstarteComponent(cr, backend.Resources, component),
				Env:             getAstarteGenericBackendEnvVars(deploymentName, cr, backend, component),
				ReadinessProbe:  getAstarteBackendProbe(cr, backend, component, customProbe),
				LivenessProbe:   getAstarteBackendProbe(cr, backend, component, customProbe),
			},
		},
		Volumes: getAstarteGenericBackendVolumes(cr),
	}

	return ps
}

func getAstarteGenericBackendVolumes(cr *apiv1alpha1.Astarte) []v1.Volume {
	ret := getAstarteCommonVolumes(cr)

	return ret
}

func getAstarteGenericBackendVolumeMounts(cr *apiv1alpha1.Astarte) []v1.VolumeMount {
	ret := getAstarteCommonVolumeMounts(cr)

	return ret
}

func getAstarteGenericBackendEnvVars(deploymentName string, cr *apiv1alpha1.Astarte, backend apiv1alpha1.AstarteGenericClusteredResource, component apiv1alpha1.AstarteComponent) []v1.EnvVar {
	ret := getAstarteCommonEnvVars(deploymentName, cr, backend, component)

	cassandraPrefix := ""
	if version.CheckConstraintAgainstAstarteComponentVersion("< 1.0.0", backend.Version, cr) == nil {
		cassandraPrefix = oldAstartePrefix
	} else {
		// Append Cassandra connection env vars only if version >= 1.0.0
		ret = appendCassandraConnectionEnvVars(ret, cr)
	}

	// Add Cassandra Nodes
	ret = append(ret, v1.EnvVar{
		Name:  cassandraPrefix + "CASSANDRA_NODES",
		Value: getCassandraNodes(cr),
	})

	eventsExchangeName := cr.Spec.RabbitMQ.EventsExchangeName

	// Depending on the component, we might need to add some more stuff.
	switch component {
	case apiv1alpha1.Housekeeping:
		if cr.Spec.AstarteSystemKeyspace.ReplicationFactor > 1 {
			ret = append(ret,
				v1.EnvVar{
					Name:  "HOUSEKEEPING_ASTARTE_KEYSPACE_REPLICATION_FACTOR",
					Value: strconv.Itoa(cr.Spec.AstarteSystemKeyspace.ReplicationFactor),
				})
		}
		if cr.Spec.Features.RealmDeletion {
			ret = append(ret,
				v1.EnvVar{
					Name:  "HOUSEKEEPING_ENABLE_REALM_DELETION",
					Value: "true",
				})
		}
	case apiv1alpha1.Pairing:
		ret = append(ret,
			v1.EnvVar{
				Name:  "PAIRING_CFSSL_URL",
				Value: getCFSSLURL(cr),
			},
			v1.EnvVar{
				Name:  "PAIRING_BROKER_URL",
				Value: misc.GetVerneMQBrokerURL(cr),
			})
	case apiv1alpha1.DataUpdaterPlant:
		ret = append(ret, getAstarteDataUpdaterPlantBackendEnvVars(eventsExchangeName, cr, backend)...)
	case apiv1alpha1.TriggerEngine:
		// Add RabbitMQ variables
		ret = appendRabbitMQConnectionEnvVars(ret, "TRIGGER_ENGINE_AMQP_CONSUMER", cr)

		if eventsExchangeName != "" {
			ret = append(ret,
				v1.EnvVar{
					Name:  "TRIGGER_ENGINE_AMQP_EVENTS_EXCHANGE_NAME",
					Value: eventsExchangeName,
				})
		}

		if cr.Spec.Components.AppengineAPI.RoomEventsQueueName != "" {
			ret = append(ret,
				v1.EnvVar{
					Name:  "TRIGGER_ENGINE_AMQP_EVENTS_QUEUE_NAME",
					Value: cr.Spec.Components.AppengineAPI.RoomEventsQueueName,
				})
		}

		if cr.Spec.Components.TriggerEngine.EventsRoutingKey != "" {
			ret = append(ret,
				v1.EnvVar{
					Name:  "TRIGGER_ENGINE_AMQP_EVENTS_ROUTING_KEY",
					Value: cr.Spec.Components.TriggerEngine.EventsRoutingKey,
				})
		}
	}

	return ret
}

func getAstarteDataUpdaterPlantBackendEnvVars(eventsExchangeName string, cr *apiv1alpha1.Astarte, backend apiv1alpha1.AstarteGenericClusteredResource) []v1.EnvVar {
	ret := []v1.EnvVar{}

	// Append RabbitMQ variables for both Consumer and Producer
	ret = appendRabbitMQConnectionEnvVars(ret, "DATA_UPDATER_PLANT_AMQP_CONSUMER", cr)
	ret = appendRabbitMQConnectionEnvVars(ret, "DATA_UPDATER_PLANT_AMQP_PRODUCER", cr)

	if eventsExchangeName != "" {
		ret = append(ret,
			v1.EnvVar{
				Name:  "DATA_UPDATER_PLANT_AMQP_EVENTS_EXCHANGE_NAME",
				Value: eventsExchangeName,
			})
	}

	if cr.Spec.Components.DataUpdaterPlant.PrefetchCount != nil {
		ret = append(ret,
			v1.EnvVar{
				Name:  "DATA_UPDATER_PLANT_AMQP_CONSUMER_PREFETCH_COUNT",
				Value: strconv.Itoa(pointy.IntValue(cr.Spec.Components.DataUpdaterPlant.PrefetchCount, 300)),
			})
	}

	// 0.11+ variables
	if version.CheckConstraintAgainstAstarteComponentVersion(">= 0.11.0", backend.Version, cr) == nil {
		dataQueueCount := getDataQueueCount(cr)

		// When installing Astarte >= 0.11, add the data queue count
		ret = append(ret,
			v1.EnvVar{
				Name: "DATA_UPDATER_PLANT_AMQP_DATA_QUEUE_RANGE_START",
				// TODO: This actually binds DUP to be Replicated by 1. This will change in the future after 0.11, most likely.
				Value: "0",
			},
			v1.EnvVar{
				Name: "DATA_UPDATER_PLANT_AMQP_DATA_QUEUE_RANGE_END",
				// same as above, but fixed at queue count. Subtract 1 since the range ends at count - 1
				Value: strconv.Itoa(dataQueueCount - 1),
			})

		// 0.11.1+ variables
		if version.CheckConstraintAgainstAstarteComponentVersion(">= 0.11.1", backend.Version, cr) == nil {
			ret = append(ret,
				v1.EnvVar{
					Name: "DATA_UPDATER_PLANT_AMQP_DATA_QUEUE_TOTAL_COUNT",
					// This must always hold the total data queue count, not just the one this specific replica of DUP is using
					Value: strconv.Itoa(dataQueueCount),
				})
		}

		if cr.Spec.RabbitMQ.DataQueuesPrefix != "" {
			ret = append(ret,
				v1.EnvVar{
					Name:  "DATA_UPDATER_PLANT_AMQP_DATA_QUEUE_PREFIX",
					Value: cr.Spec.RabbitMQ.DataQueuesPrefix,
				})
		}
	}

	// 1.0+ variables
	if version.CheckConstraintAgainstAstarteComponentVersion(">= 1.0.0", backend.Version, cr) == nil {
		if cr.Spec.VerneMQ.DeviceHeartbeatSeconds > 0 {
			ret = append(ret,
				v1.EnvVar{
					Name:  "DATA_UPDATER_PLANT_DEVICE_HEARTBEAT_INTERVAL_MS",
					Value: strconv.Itoa(cr.Spec.VerneMQ.DeviceHeartbeatSeconds * 1000),
				})
		}
	}

	return ret
}

func getAstarteBackendProbe(cr *apiv1alpha1.Astarte, backend apiv1alpha1.AstarteGenericClusteredResource,
	component apiv1alpha1.AstarteComponent, customProbe *v1.Probe) *v1.Probe {
	if customProbe != nil {
		return customProbe
	}

	if version.CheckConstraintAgainstAstarteComponentVersion("< 0.11.0", backend.Version, cr) == nil {
		// 0.10.x has no such thing.
		return nil
	}

	// Custom components
	if component == apiv1alpha1.Housekeeping {
		// We need a much longer timeout, as we have an initialization which happens 3 times
		return getAstarteBackendGenericProbeWithThreshold("/health", 15)
	}

	// The rest are generic probes on /health
	return getAstarteBackendGenericProbe("/health")
}

func getAstarteBackendGenericProbe(path string) *v1.Probe {
	return getAstarteBackendGenericProbeWithThreshold(path, 5)
}

func getAstarteBackendGenericProbeWithThreshold(path string, threshold int32) *v1.Probe {
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
		FailureThreshold:    threshold,
	}
}
