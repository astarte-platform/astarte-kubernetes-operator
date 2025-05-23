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
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"go.openly.dev/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/misc"
)

// EnsureAstarteGenericAPI reconciles any component compatible with AstarteGenericAPISpec with a custom Probe
func EnsureAstarteGenericAPI(cr *apiv1alpha2.Astarte, api apiv1alpha2.AstarteGenericAPISpec, component apiv1alpha2.AstarteComponent,
	c client.Client, scheme *runtime.Scheme) error {
	return EnsureAstarteGenericAPIWithCustomProbe(cr, api, component, c, scheme, nil)
}

// EnsureAstarteGenericAPIWithCustomProbe reconciles any component compatible with AstarteGenericAPISpec with a custom Probe
func EnsureAstarteGenericAPIWithCustomProbe(cr *apiv1alpha2.Astarte, api apiv1alpha2.AstarteGenericAPISpec, component apiv1alpha2.AstarteComponent,
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
	if !checkShouldDeployAPI(reqLogger, deploymentName, cr, api, component, c) {
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

	// Service Account?
	if component == apiv1alpha2.FlowComponent {
		if err := reconcileRBACForFlow(cr.Name+"-"+component.ServiceName(), cr, c, scheme); err != nil {
			return err
		}
	}

	deploymentSpec := appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: matchLabels,
		},
		Strategy: getDeploymentStrategyForClusteredResource(cr, api.AstarteGenericClusteredResource, component),
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: computePodLabels(api.AstarteGenericClusteredResource, labels),
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
		deployment.Spec.Replicas = getReplicaCountForResource(&api.AstarteGenericClusteredResource, cr, c, reqLogger)

		return nil
	})
	if err != nil {
		return err
	}

	misc.LogCreateOrUpdateOperationResult(log, result, cr, deployment)
	return nil
}

func checkShouldDeployAPI(reqLogger logr.Logger, deploymentName string, cr *apiv1alpha2.Astarte, api apiv1alpha2.AstarteGenericAPISpec,
	component apiv1alpha2.AstarteComponent, c client.Client) bool {
	defaultDeployValue := true
	// Flow should be deployed only if explicitly requested
	if component == apiv1alpha2.FlowComponent {
		defaultDeployValue = false
	}

	if !pointy.BoolValue(api.Deploy, defaultDeployValue) {
		reqLogger.V(1).Info("Skipping Astarte Component Deployment")
		// Before returning - check if we shall clean up the Deployment.
		// It is the only thing actually requiring resources, the rest will be cleaned up eventually when the
		// Astarte resource is deleted.
		theDeployment := &appsv1.Deployment{}
		if err := c.Get(context.TODO(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, theDeployment); err == nil {
			reqLogger.Info("Deleting previously existing Component Deployment, which is no longer needed")
			if err = c.Delete(context.TODO(), theDeployment); err != nil {
				reqLogger.Error(err, "Could not delete previously existing Component Deployment")
			}
		}

		// In any case, we should not deploy
		return false
	}

	return true
}

func getAstarteGenericAPIPodSpec(deploymentName string, cr *apiv1alpha2.Astarte, api apiv1alpha2.AstarteGenericAPISpec,
	component apiv1alpha2.AstarteComponent, customProbe *v1.Probe) v1.PodSpec {
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
				VolumeMounts:    getAstarteGenericAPIVolumeMounts(cr, component),
				Image:           getAstarteImageForClusteredResource(component.DockerImageName(), api.AstarteGenericClusteredResource, cr),
				ImagePullPolicy: getImagePullPolicy(cr),
				Resources:       misc.GetResourcesForAstarteComponent(cr, api.Resources, component),
				Env:             getAstarteGenericAPIEnvVars(deploymentName, cr, api, component),
				ReadinessProbe:  getAstarteAPIProbe(customProbe),
				LivenessProbe:   getAstarteAPIProbe(customProbe),
			},
		},
		Volumes: getAstarteGenericAPIVolumes(cr, component),
	}

	if component == apiv1alpha2.FlowComponent {
		ps.ServiceAccountName = cr.Name + "-" + component.ServiceName()
	}

	// do we want priorities?
	if cr.Spec.Features.AstartePodPriorities.IsEnabled() {
		// is a priorityClass specified in the Astarte CR?
		switch api.PriorityClass {
		case highPriority:
			ps.PriorityClassName = AstarteHighPriorityName
		case midPriority:
			ps.PriorityClassName = AstarteMidPriorityName
		case lowPriority:
			ps.PriorityClassName = AstarteLowPriorityName
		default:
			ps.PriorityClassName = GetDefaultAstartePriorityClassNameForComponent(component)
		}
	}

	return ps
}

func getAstarteGenericAPIVolumes(cr *apiv1alpha2.Astarte, component apiv1alpha2.AstarteComponent) []v1.Volume {
	ret := getAstarteCommonVolumes(cr)

	// Depending on the component, we might need to add some more stuff.
	if component == apiv1alpha2.HousekeepingAPI {
		ret = append(ret, v1.Volume{
			Name: "jwtpubkey",
			VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{
				SecretName: fmt.Sprintf("%s-housekeeping-public-key", cr.Name),
			}},
		})
	}

	return ret
}

func getAstarteGenericAPIVolumeMounts(cr *apiv1alpha2.Astarte, component apiv1alpha2.AstarteComponent) []v1.VolumeMount {
	ret := getAstarteCommonVolumeMounts(cr)

	// Depending on the component, we might need to add some more stuff.
	if component == apiv1alpha2.HousekeepingAPI {
		ret = append(ret, v1.VolumeMount{
			Name:      "jwtpubkey",
			MountPath: "/jwtpubkey",
			ReadOnly:  true,
		})
	}

	return ret
}

func getAstarteGenericAPIEnvVars(deploymentName string, cr *apiv1alpha2.Astarte, api apiv1alpha2.AstarteGenericAPISpec, component apiv1alpha2.AstarteComponent) []v1.EnvVar {
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
	case apiv1alpha2.AppEngineAPI:
		if cr.Spec.AstarteInstanceID != "" {
			ret = append(ret, v1.EnvVar{
				Name:  "ASTARTE_INSTANCE_ID",
				Value: cr.Spec.AstarteInstanceID,
			})
		}

		ret = appendCassandraConnectionEnvVars(ret, cr)

		// Add Cassandra Nodes
		ret = append(ret, v1.EnvVar{
			Name:  "CASSANDRA_NODES",
			Value: getCassandraNodes(cr),
		})

		ret = append(ret,
			v1.EnvVar{
				Name:  "APPENGINE_API_MAX_RESULTS_LIMIT",
				Value: strconv.Itoa(getAppEngineAPIMaxResultslimit(cr)),
			},
		)

		// Append RabbitMQ variables
		ret = appendRabbitMQConnectionEnvVars(ret, "APPENGINE_API_ROOMS_AMQP_CLIENT", cr)

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
	case apiv1alpha2.HousekeepingAPI:
		// Add Public Key Information
		ret = append(ret, v1.EnvVar{
			Name:  "HOUSEKEEPING_API_JWT_PUBLIC_KEY_PATH",
			Value: "/jwtpubkey/public-key",
		})
	case apiv1alpha2.FlowComponent:
		ret = append(ret, getAstarteFlowEnvVars(cr)...)
	}

	return ret
}

func getAstarteFlowEnvVars(cr *apiv1alpha2.Astarte) []v1.EnvVar {
	// TODO: This assumes Flow runs paired with the rest of Astarte. Handle other cases.
	ret := []v1.EnvVar{
		{
			Name:  "FLOW_ASTARTE_INSTANCE",
			Value: cr.Name,
		},
		{
			Name:  "FLOW_TARGET_NAMESPACE",
			Value: cr.Namespace,
		},
		{
			Name:  "FLOW_PIPELINES_DIR",
			Value: "/pipelines",
		},
		{
			Name:  "CASSANDRA_NODES",
			Value: getCassandraNodes(cr),
		},
		{
			Name:  "FLOW_REALM_PUBLIC_KEY_PROVIDER",
			Value: "astarte",
		},
	}
	if cr.Spec.AstarteInstanceID != "" {
		ret = append(ret, v1.EnvVar{
			Name:  "ASTARTE_INSTANCE_ID",
			Value: cr.Spec.AstarteInstanceID,
		})
	}

	// Append RabbitMQ variables
	return appendRabbitMQConnectionEnvVars(ret, "FLOW_DEFAULT_AMQP_CONNECTION", cr)
}

func getAstarteAPIProbe(customProbe *v1.Probe) *v1.Probe {
	if customProbe != nil {
		return customProbe
	}

	// The rest are generic probes on /health
	return getAstarteAPIGenericProbe("/health")
}

func getAstarteAPIGenericProbe(path string) *v1.Probe {
	return &v1.Probe{
		ProbeHandler: v1.ProbeHandler{
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
