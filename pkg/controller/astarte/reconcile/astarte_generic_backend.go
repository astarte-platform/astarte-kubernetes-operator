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

	semver "github.com/Masterminds/semver/v3"
	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/misc"
	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EnsureAstarteGenericBackend reconciles any component compatible with AstarteGenericClusteredResource
func EnsureAstarteGenericBackend(cr *apiv1alpha1.Astarte, backend apiv1alpha1.AstarteGenericClusteredResource, component apiv1alpha1.AstarteComponent, c client.Client, scheme *runtime.Scheme) error {
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name, "Astarte.Component", component)
	deploymentName := cr.Name + "-" + component.DashedString()
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

	deploymentSpec := appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: matchLabels,
		},
		Strategy: getDeploymentStrategyForClusteredResource(cr, backend),
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: getAstarteGenericBackendPodSpec(deploymentName, cr, backend, component),
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

	logCreateOrUpdateOperationResult(result, cr, deployment)
	return nil
}

func getAstarteGenericBackendPodSpec(deploymentName string, cr *apiv1alpha1.Astarte, backend apiv1alpha1.AstarteGenericClusteredResource, component apiv1alpha1.AstarteComponent) v1.PodSpec {
	ps := v1.PodSpec{
		TerminationGracePeriodSeconds: pointy.Int64(30),
		ImagePullSecrets:              cr.Spec.ImagePullSecrets,
		Affinity:                      getAffinityForClusteredResource(deploymentName, backend),
		Containers: []v1.Container{
			v1.Container{
				Name:            component.DashedString(),
				VolumeMounts:    getAstarteGenericBackendVolumeMounts(deploymentName, cr, backend, component),
				Image:           getAstarteImageForClusteredResource(component.DockerImageName(), backend, cr),
				ImagePullPolicy: getImagePullPolicy(cr),
				Resources:       misc.GetResourcesForAstarteComponent(cr, backend.Resources, component),
				Env:             getAstarteGenericBackendEnvVars(deploymentName, cr, backend, component),
			},
		},
		Volumes: getAstarteGenericBackendVolumes(deploymentName, cr, backend, component),
	}

	return ps
}

func getAstarteGenericBackendVolumes(deploymentName string, cr *apiv1alpha1.Astarte, backend apiv1alpha1.AstarteGenericClusteredResource, component apiv1alpha1.AstarteComponent) []v1.Volume {
	ret := getAstarteCommonVolumes(cr)

	// Depending on the component, we might need to add some more stuff.
	switch component {
	}

	return ret
}

func getAstarteGenericBackendVolumeMounts(deploymentName string, cr *apiv1alpha1.Astarte, backend apiv1alpha1.AstarteGenericClusteredResource, component apiv1alpha1.AstarteComponent) []v1.VolumeMount {
	ret := getAstarteCommonVolumeMounts()

	// Depending on the component, we might need to add some more stuff.
	switch component {
	}

	return ret
}

func getAstarteGenericBackendEnvVars(deploymentName string, cr *apiv1alpha1.Astarte, backend apiv1alpha1.AstarteGenericClusteredResource, component apiv1alpha1.AstarteComponent) []v1.EnvVar {
	ret := getAstarteCommonEnvVars(deploymentName, cr, component)
	// Add Cassandra Nodes
	ret = append(ret, v1.EnvVar{
		Name:  "ASTARTE_CASSANDRA_NODES",
		Value: getCassandraNodes(cr),
	})

	// Depending on the component, we might need to add some more stuff.
	switch component {
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
		rabbitMQHost, rabbitMQPort := misc.GetRabbitMQHostnameAndPort(cr)
		userCredentialsSecretName, userCredentialsSecretUsernameKey, userCredentialsSecretPasswordKey := misc.GetRabbitMQUserCredentialsSecret(cr)
		ret = append(ret,
			v1.EnvVar{
				Name:  "DATA_UPDATER_PLANT_AMQP_CONSUMER_HOST",
				Value: rabbitMQHost,
			},
			v1.EnvVar{
				Name:  "DATA_UPDATER_PLANT_AMQP_CONSUMER_PORT",
				Value: strconv.Itoa(int(rabbitMQPort)),
			},
			v1.EnvVar{
				Name: "DATA_UPDATER_PLANT_AMQP_CONSUMER_USERNAME",
				ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
					Key:                  userCredentialsSecretUsernameKey,
				}},
			},
			v1.EnvVar{
				Name: "DATA_UPDATER_PLANT_AMQP_CONSUMER_PASSWORD",
				ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
					Key:                  userCredentialsSecretPasswordKey,
				}},
			},
			v1.EnvVar{
				Name:  "DATA_UPDATER_PLANT_AMQP_PRODUCER_HOST",
				Value: rabbitMQHost,
			},
			v1.EnvVar{
				Name:  "DATA_UPDATER_PLANT_AMQP_PRODUCER_PORT",
				Value: strconv.Itoa(int(rabbitMQPort)),
			},
			v1.EnvVar{
				Name:  "DATA_UPDATER_PLANT_AMQP_PRODUCER_VIRTUAL_HOST",
				Value: "/",
			},
			v1.EnvVar{
				Name: "DATA_UPDATER_PLANT_AMQP_PRODUCER_USERNAME",
				ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
					Key:                  userCredentialsSecretUsernameKey,
				}},
			},
			v1.EnvVar{
				Name: "DATA_UPDATER_PLANT_AMQP_PRODUCER_PASSWORD",
				ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
					Key:                  userCredentialsSecretPasswordKey,
				}},
			})

		c, _ := semver.NewConstraint(">= 0.11.0")
		ver, _ := semver.NewVersion(getVersionForAstarteComponent(cr, backend.Version))
		checkVersion, _ := ver.SetPrerelease("")

		if c.Check(&checkVersion) {
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
					Value: strconv.Itoa(getDataQueueCount(cr) - 1),
				})
		}
	case apiv1alpha1.TriggerEngine:
		rabbitMQHost, rabbitMQPort := misc.GetRabbitMQHostnameAndPort(cr)
		userCredentialsSecretName, userCredentialsSecretUsernameKey, userCredentialsSecretPasswordKey := misc.GetRabbitMQUserCredentialsSecret(cr)
		ret = append(ret,
			v1.EnvVar{
				Name:  "TRIGGER_ENGINE_AMQP_CONSUMER_HOST",
				Value: rabbitMQHost,
			},
			v1.EnvVar{
				Name:  "TRIGGER_ENGINE_AMQP_CONSUMER_PORT",
				Value: strconv.Itoa(int(rabbitMQPort)),
			},
			v1.EnvVar{
				Name: "TRIGGER_ENGINE_AMQP_CONSUMER_USERNAME",
				ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
					Key:                  userCredentialsSecretUsernameKey,
				}},
			},
			v1.EnvVar{
				Name: "TRIGGER_ENGINE_AMQP_CONSUMER_PASSWORD",
				ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
					Key:                  userCredentialsSecretPasswordKey,
				}},
			})
	}

	return ret
}
