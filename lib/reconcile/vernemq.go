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

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
	"github.com/astarte-platform/astarte-kubernetes-operator/version"
)

// EnsureVerneMQ reconciles VerneMQ
func EnsureVerneMQ(cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	//reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	statefulSetName := cr.Name + "-vernemq"
	labels := map[string]string{"app": statefulSetName}

	// Validate where necessary
	if err := validateVerneMQDefinition(&cr.Spec.VerneMQ); err != nil {
		return err
	}

	// Ok. Shall we deploy?
	if !pointy.BoolValue(cr.Spec.VerneMQ.Deploy, true) {
		log.Info("Skipping VerneMQ Deployment")
		// Before returning - check if we shall clean up the StatefulSet.
		// It is the only thing actually requiring resources, the rest will be cleaned up eventually when the
		// Astarte resource is deleted.
		theStatefulSet := &appsv1.StatefulSet{}
		err := c.Get(context.TODO(), types.NamespacedName{Name: statefulSetName, Namespace: cr.Namespace}, theStatefulSet)
		if err == nil {
			log.Info("Deleting previously existing VerneMQ StatefulSet, which is no longer needed")
			if err = c.Delete(context.TODO(), theStatefulSet); err != nil {
				return err
			}
		}

		// That would be all for today.
		return nil
	}

	// Ensure we reconcile with the RBAC Roles, if needed.
	if pointy.BoolValue(cr.Spec.RBAC, true) {
		if err := reconcileStandardRBACForClusteringForApp(statefulSetName, getVerneMQPolicyRules(), cr, c, scheme); err != nil {
			return err
		}
	}

	// Good. Now, reconcile the service first of all.
	service := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: statefulSetName, Namespace: cr.Namespace}}
	if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, service, func() error {
		if err := controllerutil.SetControllerReference(cr, service, scheme); err != nil {
			return err
		}
		// Always set everything to what we require.
		service.Spec.Type = v1.ServiceTypeClusterIP
		service.Spec.ClusterIP = noneClusterIP
		service.Spec.Ports = []v1.ServicePort{
			{
				Name:       "mqtt",
				Port:       1883,
				TargetPort: intstr.FromString("mqtt"),
				Protocol:   v1.ProtocolTCP,
			},
			{
				Name:       "mqtt-reverse",
				Port:       1885,
				TargetPort: intstr.FromString("mqtt-reverse"),
				Protocol:   v1.ProtocolTCP,
			},
			{
				Name:       "webadmin",
				Port:       8888,
				TargetPort: intstr.FromString("webadmin"),
				Protocol:   v1.ProtocolTCP,
			},
		}
		service.Spec.Selector = labels
		// Add Annotations for Voyager (when deployed)
		service.Annotations = map[string]string{
			"ingress.appscode.com/send-proxy": "v2-ssl-cn",
			"ingress.appscode.com/check":      strconv.FormatBool(true),
		}
		return nil
	}); err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, service)
	} else {
		return err
	}

	// Let's check upon Storage now.
	dataVolumeName, persistentVolumeClaim := computePersistentVolumeClaim(statefulSetName+"-data", resource.NewScaledQuantity(4, resource.Giga),
		cr.Spec.VerneMQ.Storage, cr)

	// Compute and prepare all data for building the StatefulSet
	statefulSetSpec := appsv1.StatefulSetSpec{
		ServiceName: statefulSetName,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: computePodLabels(cr.Spec.VerneMQ.AstarteGenericClusteredResource, labels),
			},
			Spec: getVerneMQPodSpec(statefulSetName, dataVolumeName, cr),
		},
	}

	if persistentVolumeClaim != nil {
		statefulSetSpec.VolumeClaimTemplates = []v1.PersistentVolumeClaim{*persistentVolumeClaim}
	}

	// Build the StatefulSet
	vmqStatefulSet := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: statefulSetName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, vmqStatefulSet, func() error {
		if err := controllerutil.SetControllerReference(cr, vmqStatefulSet, scheme); err != nil {
			return err
		}

		// Assign the Spec.
		vmqStatefulSet.ObjectMeta.Labels = map[string]string{"component": "astarte"}
		vmqStatefulSet.Spec = statefulSetSpec
		vmqStatefulSet.Spec.Replicas = cr.Spec.VerneMQ.Replicas

		return nil
	})
	if err != nil {
		return err
	}

	misc.LogCreateOrUpdateOperationResult(log, result, cr, service)
	return nil
}

func validateVerneMQDefinition(vmq *apiv1alpha1.AstarteVerneMQSpec) error {
	if vmq == nil {
		return nil
	}
	// All is good.
	return nil
}

func getVerneMQProbe() *v1.Probe {
	// Start checking after 1 minute, every 20 seconds, fail after the 3rd attempt
	return &v1.Probe{
		ProbeHandler:        v1.ProbeHandler{HTTPGet: &v1.HTTPGetAction{Path: "/metrics", Port: intstr.FromInt(8888)}},
		InitialDelaySeconds: 60,
		TimeoutSeconds:      10,
		PeriodSeconds:       20,
		FailureThreshold:    3,
	}
}

func getVerneMQEnvVars(statefulSetName string, cr *apiv1alpha1.Astarte) []v1.EnvVar {
	dataQueueCount := getDataQueueCount(cr)
	mirrorQueue := getMirrorQueue(cr)

	envVars := []v1.EnvVar{
		{
			Name:      "MY_POD_NAME",
			ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.name"}},
		},
		{
			Name:      "MY_POD_IP",
			ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "status.podIP"}},
		},
		{
			Name:  "DOCKER_VERNEMQ_DISCOVERY_KUBERNETES",
			Value: "1",
		},
		{
			Name:  "DOCKER_VERNEMQ_KUBERNETES_LABEL_SELECTOR",
			Value: "app=" + statefulSetName,
		},
	}

	// Append RabbitMQ variables (trailing _, as we need two)
	envVars = appendRabbitMQConnectionEnvVars(envVars, "DOCKER_VERNEMQ_ASTARTE_VMQ_PLUGIN__AMQP_", cr)
	// Also append env vars for RPC
	envVars = appendRabbitMQConnectionEnvVars(envVars, "RPC_AMQP_CONNECTION", cr)

	// Add the data queue count
	envVars = append(envVars, v1.EnvVar{
		Name:  "DOCKER_VERNEMQ_ASTARTE_VMQ_PLUGIN__AMQP__DATA_QUEUE_COUNT",
		Value: strconv.Itoa(dataQueueCount),
	})

	if mirrorQueue != "" {
		// If a mirror queue is defined, set the relevant environment variable
		envVars = append(envVars, v1.EnvVar{
			Name:  "DOCKER_VERNEMQ_ASTARTE_VMQ_PLUGIN__AMQP__MIRROR_QUEUE_NAME",
			Value: mirrorQueue,
		})
	}

	// 1.0+ variables
	if version.CheckConstraintAgainstAstarteComponentVersion(">= 1.0.0", cr.Spec.VerneMQ.Version, cr.Spec.Version) == nil {
		if cr.Spec.VerneMQ.DeviceHeartbeatSeconds > 0 {
			envVars = append(envVars,
				v1.EnvVar{
					Name:  "DOCKER_VERNEMQ_ASTARTE_VMQ_PLUGIN__DEVICE_HEARTBEAT_INTERVAL_MS",
					Value: strconv.Itoa(cr.Spec.VerneMQ.DeviceHeartbeatSeconds * 1000),
				})
		}

		if pointy.BoolValue(cr.Spec.VerneMQ.SSLListener, false) && cr.Spec.VerneMQ.SSLListenerCertSecretName != "" {
			// if we are here, SSL termination must be handled at VMQ level
			// thus, append the proper env variables
			envVars = append(envVars, v1.EnvVar{
				Name:  "VERNEMQ_ENABLE_SSL_LISTENER",
				Value: strconv.FormatBool(true),
			})

			envVars = append(envVars, v1.EnvVar{
				// to check where ca.pem comes from, have a look at this script
				// https://github.com/astarte-platform/astarte_vmq_plugin/blob/master/docker/bin/vernemq.sh#L141
				Name:  "DOCKER_VERNEMQ_LISTENER__SSL__DEFAULT__CAFILE",
				Value: "/opt/vernemq/etc/ca.pem",
			})

			envVars = append(envVars, v1.EnvVar{
				Name:  "DOCKER_VERNEMQ_LISTENER__SSL__DEFAULT__CERTFILE",
				Value: "/opt/vernemq/etc/cert.pem",
			})

			envVars = append(envVars, v1.EnvVar{
				Name:  "DOCKER_VERNEMQ_LISTENER__SSL__DEFAULT__KEYFILE",
				Value: "/opt/vernemq/etc/privkey.pem",
			})

			envVars = append(envVars, v1.EnvVar{
				Name:  "CFSSL_URL",
				Value: fmt.Sprintf("http://%s-cfssl.%s.svc.cluster.local", cr.Name, cr.Namespace),
			})
		}

		persistentClientExpiration := cr.Spec.VerneMQ.PersistentClientExpiration
		if persistentClientExpiration == "" {
			// Defaults to 1 year
			persistentClientExpiration = "1y"
		}

		envVars = append(envVars,
			v1.EnvVar{
				Name:  "DOCKER_VERNEMQ_PERSISTENT_CLIENT_EXPIRATION",
				Value: persistentClientExpiration,
			},
			v1.EnvVar{
				Name:  "DOCKER_VERNEMQ_MAX_OFFLINE_MESSAGES",
				Value: strconv.Itoa(pointy.IntValue(cr.Spec.VerneMQ.MaxOfflineMessages, 1000000)),
			})
	}

	// Add any explicit additional env
	if len(cr.Spec.VerneMQ.AdditionalEnv) > 0 {
		envVars = append(envVars, cr.Spec.VerneMQ.AdditionalEnv...)
	}

	return envVars
}

func getVerneMQPodSpec(statefulSetName, dataVolumeName string, cr *apiv1alpha1.Astarte) v1.PodSpec {
	serviceAccountName := statefulSetName
	if pointy.BoolValue(cr.Spec.RBAC, false) {
		serviceAccountName = ""
	}

	resources := v1.ResourceRequirements{}
	if cr.Spec.VerneMQ.Resources != nil {
		resources = *cr.Spec.VerneMQ.Resources
	}

	ps := v1.PodSpec{
		TerminationGracePeriodSeconds: pointy.Int64(30),
		ServiceAccountName:            serviceAccountName,
		ImagePullSecrets:              cr.Spec.ImagePullSecrets,
		Affinity:                      getAffinityForClusteredResource(statefulSetName, cr.Spec.VerneMQ.AstarteGenericClusteredResource),
		Containers: []v1.Container{
			{
				Name:         "vernemq",
				VolumeMounts: getVerneMQVolumeMounts(dataVolumeName, cr),
				// Defaults to the custom image built in Astarte
				Image:           getAstarteImageForClusteredResource("vernemq", cr.Spec.VerneMQ.AstarteGenericClusteredResource, cr),
				ImagePullPolicy: getImagePullPolicy(cr),
				Ports: []v1.ContainerPort{
					{Name: "mqtt-ssl", ContainerPort: 8883},
					{Name: "acme-verify", ContainerPort: 80},
					{Name: "mqtt", ContainerPort: 1883},
					{Name: "mqtt-reverse", ContainerPort: 1885},
					{Name: "vmq-msg-dist", ContainerPort: 44053},
					{Name: "epmd", ContainerPort: 4369},
					{Name: "metrics", ContainerPort: 8888},
					{ContainerPort: 9100},
					{ContainerPort: 9101},
					{ContainerPort: 9102},
					{ContainerPort: 9103},
					{ContainerPort: 9104},
					{ContainerPort: 9105},
					{ContainerPort: 9106},
					{ContainerPort: 9107},
					{ContainerPort: 9108},
					{ContainerPort: 9109},
				},
				LivenessProbe:  getVerneMQProbe(),
				ReadinessProbe: getVerneMQProbe(),
				Resources:      resources,
				Env:            getVerneMQEnvVars(statefulSetName, cr),
			},
		},
		Volumes: getVerneMQVolumes(cr),
	}

	return ps
}

func getVerneMQVolumes(cr *apiv1alpha1.Astarte) []v1.Volume {
	theVolumes := []v1.Volume{}

	// if SSL termination must be handled at VerneMQ level, create the volume to store the certificates
	if pointy.BoolValue(cr.Spec.VerneMQ.SSLListener, false) && cr.Spec.VerneMQ.SSLListenerCertSecretName != "" {
		// we don't check if the secret is already there as it is enforced by the validating webhook
		theVolumes = append(theVolumes, v1.Volume{
			Name: cr.Spec.VerneMQ.SSLListenerCertSecretName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					DefaultMode: pointy.Int32(420),
					SecretName:  cr.Spec.VerneMQ.SSLListenerCertSecretName,
					Items: []v1.KeyToPath{
						{
							Key:  "tls.crt",
							Path: "cert",
						},
						{
							Key:  "tls.key",
							Path: "privkey",
						},
					},
				},
			},
		})
	}

	return theVolumes
}

func getVerneMQVolumeMounts(dataVolumeName string, cr *apiv1alpha1.Astarte) []v1.VolumeMount {
	theVolumeMounts := []v1.VolumeMount{
		{
			Name:      dataVolumeName,
			MountPath: "/opt/vernemq/data",
		},
	}

	// if SSL termination must be handled at VerneMQ level, we have to mount the certificates
	if pointy.BoolValue(cr.Spec.VerneMQ.SSLListener, false) && cr.Spec.VerneMQ.SSLListenerCertSecretName != "" {
		// If we need to expose VerneMQ, let's append the secret as a volume in the pod.
		// The key and cert in the secret are copied to /opt/vernemq/etc according to
		// this script: https://github.com/astarte-platform/astarte_vmq_plugin/blob/master/docker/bin/vernemq.sh#L137
		theVolumeMounts = append(theVolumeMounts, v1.VolumeMount{
			Name:      cr.Spec.VerneMQ.SSLListenerCertSecretName,
			MountPath: "/etc/ssl/vernemq-certs",
			ReadOnly:  true,
		})
	}
	return theVolumeMounts
}

func getVerneMQPolicyRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods", "services"},
			Verbs:     []string{"list", "get"},
		},
	}
}

func getMirrorQueue(cr *apiv1alpha1.Astarte) string {
	return cr.Spec.VerneMQ.MirrorQueue
}
