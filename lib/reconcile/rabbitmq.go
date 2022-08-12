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
	"errors"
	"fmt"
	"strconv"
	"strings"

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
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/deps"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
)

func getRabbitMQUserAndPassword(conn *apiv1alpha1.AstarteRabbitMQConnectionSpec) (string, string) {
	if conn != nil {
		return conn.Username, conn.Password
	}
	return "", ""
}

func getRabbitMQUserAndPasswordKeys(conn *apiv1alpha1.AstarteRabbitMQConnectionSpec) (string, string) {
	if conn != nil {
		if conn.Secret != nil {
			return conn.Secret.UsernameKey, conn.Secret.PasswordKey
		}
	}
	return misc.RabbitMQDefaultUserCredentialsUsernameKey, misc.RabbitMQDefaultUserCredentialsPasswordKey
}

func getRabbitMQSecret(cr *apiv1alpha1.Astarte) *apiv1alpha1.LoginCredentialsSecret {
	if cr.Spec.RabbitMQ.Connection != nil {
		return cr.Spec.RabbitMQ.Connection.Secret
	}
	return nil
}

// EnsureRabbitMQ reconciles the state of RabbitMQ
// nolint: funlen
func EnsureRabbitMQ(cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	statefulSetName := cr.Name + "-rabbitmq"
	labels := map[string]string{"app": statefulSetName}

	// Validate where necessary
	if err := validateRabbitMQDefinition(cr.Spec.RabbitMQ); err != nil {
		return err
	}

	// Depending on the situation, we need to take action on the credentials.
	secretName := cr.Name + "-rabbitmq-user-credentials"
	username, password := getRabbitMQUserAndPassword(cr.Spec.RabbitMQ.Connection)
	usernameKey, passwordKey := getRabbitMQUserAndPasswordKeys(cr.Spec.RabbitMQ.Connection)
	secret := getRabbitMQSecret(cr)
	forceCredentialsCreation := true

	if err := handleGenericUserCredentialsSecret(username, password, usernameKey, passwordKey, secretName, forceCredentialsCreation, secret, cr, c, scheme); err != nil {
		return err
	}

	// Ok. Shall we deploy?
	if !pointy.BoolValue(cr.Spec.RabbitMQ.Deploy, true) {
		log.Info("Skipping RabbitMQ Deployment")
		// Before returning - check if we shall clean up the StatefulSet.
		// It is the only thing actually requiring resources, the rest will be cleaned up eventually when the
		// Astarte resource is deleted.
		theStatefulSet := &appsv1.StatefulSet{}
		err := c.Get(context.TODO(), types.NamespacedName{Name: statefulSetName, Namespace: cr.Namespace}, theStatefulSet)
		if err == nil {
			log.Info("Deleting previously existing RabbitMQ StatefulSet, which is no longer needed")
			if err = c.Delete(context.TODO(), theStatefulSet); err != nil {
				return err
			}
		}
		// That would be all for today.
		return nil
	}

	// First of all, check if we need to regenerate the cookie.
	if err := ensureErlangCookieSecret(statefulSetName+"-cookie", cr, c, scheme); err != nil {
		return err
	}

	// Ensure we reconcile with the RBAC Roles, if needed.
	if pointy.BoolValue(cr.Spec.RBAC, true) {
		if err := reconcileStandardRBACForClusteringForApp(statefulSetName, getRabbitMQPolicyRules(), cr, c, scheme); err != nil {
			return err
		}
	}

	// Good. Now, reconcile the service first of all.
	service := &v1.Service{ObjectMeta: getCommonRabbitMQObjectMeta(statefulSetName, cr)}
	if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, service, func() error {
		if err := controllerutil.SetControllerReference(cr, service, scheme); err != nil {
			return err
		}
		// Always set everything to what we require.
		service.Spec.Type = v1.ServiceTypeClusterIP
		service.Spec.ClusterIP = noneClusterIP
		service.Spec.Ports = []v1.ServicePort{
			{
				Name:       "amqp",
				Port:       5672,
				TargetPort: intstr.FromString("amqp"),
				Protocol:   v1.ProtocolTCP,
			},
			{
				Name:       "management",
				Port:       15672,
				TargetPort: intstr.FromString("management"),
				Protocol:   v1.ProtocolTCP,
			},
			{
				Name:       "metrics",
				Port:       15692,
				TargetPort: intstr.FromString("metrics"),
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

	// Good. Reconcile the ConfigMap.
	if _, err := misc.ReconcileConfigMap(statefulSetName+"-config", getRabbitMQConfigMapData(statefulSetName, cr), cr, c, scheme, log); err != nil {
		return err
	}

	// Let's check upon Storage now.
	dataVolumeName, persistentVolumeClaim := computePersistentVolumeClaim(statefulSetName+"-data", resource.NewScaledQuantity(4, resource.Giga),
		cr.Spec.RabbitMQ.Storage, cr)

	// Compute and prepare all data for building the StatefulSet
	statefulSetSpec := appsv1.StatefulSetSpec{
		ServiceName: statefulSetName,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: computePodLabels(cr.Spec.RabbitMQ.AstarteGenericClusteredResource, labels),
			},
			Spec: getRabbitMQPodSpec(statefulSetName, dataVolumeName, cr),
		},
	}

	if persistentVolumeClaim != nil {
		statefulSetSpec.VolumeClaimTemplates = []v1.PersistentVolumeClaim{*persistentVolumeClaim}
	}

	// Build the StatefulSet
	rmqStatefulSet := &appsv1.StatefulSet{ObjectMeta: getCommonRabbitMQObjectMeta(statefulSetName, cr)}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, rmqStatefulSet, func() error {
		if err := controllerutil.SetControllerReference(cr, rmqStatefulSet, scheme); err != nil {
			return err
		}

		// Assign the Spec.
		rmqStatefulSet.Spec = statefulSetSpec
		rmqStatefulSet.Spec.Replicas = cr.Spec.RabbitMQ.Replicas

		return nil
	})
	if err != nil {
		return err
	}

	misc.LogCreateOrUpdateOperationResult(log, result, cr, service)
	return nil
}

func validateRabbitMQDefinition(rmq apiv1alpha1.AstarteRabbitMQSpec) error {
	if !pointy.BoolValue(rmq.Deploy, true) {
		// We need to make sure that we have all needed components
		if rmq.Connection == nil {
			return errors.New("When not deploying RabbitMQ, the 'connection' section is compulsory")
		}
		if rmq.Connection.Host == "" {
			return errors.New("When not deploying RabbitMQ, it is compulsory to specify at least a Host")
		}
		if (rmq.Connection.Username == "" || rmq.Connection.Password == "") && rmq.Connection.Secret == nil {
			return errors.New("When not deploying RabbitMQ, either a username/password combination or a Kubernetes secret must be provided")
		}
	}
	// All is good.
	return nil
}

func getRabbitMQInitContainers(dataVolumeName string) []v1.Container {
	return []v1.Container{
		{
			Name:  "setup-rabbitmq",
			Image: "busybox",
			Command: []string{
				"sh",
				"-c",
				"cp /configmap/* /etc/rabbitmq " +
					"&& cp /erlang-cookie-secret/.erlang.cookie /var/lib/rabbitmq/.erlang.cookie " +
					"&& chmod 400 /var/lib/rabbitmq/.erlang.cookie",
			},
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      "config-volume",
					MountPath: "/configmap",
				},
				{
					Name:      "config",
					MountPath: "/etc/rabbitmq",
				},
				{
					Name:      dataVolumeName,
					MountPath: "/var/lib/rabbitmq",
				},
				{
					Name:      "erlang-cookie-secret",
					MountPath: "/erlang-cookie-secret",
				},
			},
		},
	}
}

func getRabbitMQLivenessProbe() *v1.Probe {
	// rabbitmqctl status is pretty expensive. Don't run it more than once per minute.
	// Also, give it enough time to start.
	return &v1.Probe{
		ProbeHandler:        v1.ProbeHandler{Exec: &v1.ExecAction{Command: []string{"rabbitmqctl", "status"}}},
		InitialDelaySeconds: 300,
		TimeoutSeconds:      10,
		PeriodSeconds:       60,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
}

func getRabbitMQReadinessProbe() *v1.Probe {
	// Starting at least 3.7.21, rabbitmqctl status fails if the app hasn't started yet, and this could take *a lot*
	// of time. Increase the failure threshold to 15 before giving up.
	return &v1.Probe{
		ProbeHandler:        v1.ProbeHandler{Exec: &v1.ExecAction{Command: []string{"rabbitmqctl", "status"}}},
		InitialDelaySeconds: 30,
		TimeoutSeconds:      10,
		PeriodSeconds:       30,
		SuccessThreshold:    1,
		FailureThreshold:    15,
	}
}

func getRabbitMQEnvVars(statefulSetName string, cr *apiv1alpha1.Astarte) []v1.EnvVar {
	userCredentialsSecretName, userCredentialsSecretUsernameKey, userCredentialsSecretPasswordKey := misc.GetRabbitMQUserCredentialsSecret(cr)

	ret := []v1.EnvVar{
		{
			Name:  "RABBITMQ_USE_LONGNAME",
			Value: strconv.FormatBool(true),
		},
		{
			Name:      "MY_POD_NAME",
			ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.name"}},
		},
		{
			Name:  "RABBITMQ_NODENAME",
			Value: fmt.Sprintf("rabbit@$(MY_POD_NAME).%s.%s.svc.cluster.local", statefulSetName, cr.Namespace),
		},
		{
			Name:  "K8S_SERVICE_NAME",
			Value: statefulSetName,
		},
		{
			Name: "RABBITMQ_DEFAULT_USER",
			ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
				Key:                  userCredentialsSecretUsernameKey,
			}},
		},
		{
			Name: "RABBITMQ_DEFAULT_PASS",
			ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
				Key:                  userCredentialsSecretPasswordKey,
			}},
		},
	}

	// Add any explicit additional env
	if len(cr.Spec.RabbitMQ.AdditionalEnv) > 0 {
		ret = append(ret, cr.Spec.RabbitMQ.AdditionalEnv...)
	}

	return ret
}

func getRabbitMQPodSpec(statefulSetName, dataVolumeName string, cr *apiv1alpha1.Astarte) v1.PodSpec {
	serviceAccountName := statefulSetName
	if pointy.BoolValue(cr.Spec.RBAC, false) {
		serviceAccountName = ""
	}

	resources := v1.ResourceRequirements{}
	if cr.Spec.RabbitMQ.Resources != nil {
		resources = *cr.Spec.RabbitMQ.Resources
	}

	ps := v1.PodSpec{
		TerminationGracePeriodSeconds: pointy.Int64(30),
		ServiceAccountName:            serviceAccountName,
		InitContainers:                getRabbitMQInitContainers(dataVolumeName),
		ImagePullSecrets:              cr.Spec.ImagePullSecrets,
		Affinity:                      getAffinityForClusteredResource(statefulSetName, cr.Spec.RabbitMQ.AstarteGenericClusteredResource),
		Containers: []v1.Container{
			{
				Name: "rabbitmq",
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      "config",
						MountPath: "/etc/rabbitmq",
					},
					{
						Name:      dataVolumeName,
						MountPath: "/var/lib/rabbitmq",
					},
				},
				Image: getImageForClusteredResource("rabbitmq", deps.GetDefaultVersionForRabbitMQ(cr.Spec.Version),
					cr.Spec.RabbitMQ.AstarteGenericClusteredResource),
				ImagePullPolicy: getImagePullPolicy(cr),
				Ports: []v1.ContainerPort{
					{Name: "amqp", ContainerPort: 5672},
					{Name: "management", ContainerPort: 15672},
					{Name: "metrics", ContainerPort: 15692},
				},
				LivenessProbe:  getRabbitMQLivenessProbe(),
				ReadinessProbe: getRabbitMQReadinessProbe(),
				Resources:      resources,
				Env:            getRabbitMQEnvVars(statefulSetName, cr),
			},
		},
		Volumes: []v1.Volume{
			{
				Name:         "config",
				VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
			},
			{
				Name: "config-volume",
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{Name: statefulSetName + "-config"},
						Items: []v1.KeyToPath{
							{
								Key:  "rabbitmq.conf",
								Path: "rabbitmq.conf",
							},
							{
								Key:  "enabled_plugins",
								Path: "enabled_plugins",
							},
						},
					},
				},
			},
			{
				Name: "erlang-cookie-secret",
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: statefulSetName + "-cookie",
						Items: []v1.KeyToPath{
							{
								Key:  "erlang-cookie",
								Path: ".erlang.cookie",
							},
						},
					},
				},
			},
		},
	}

	return ps
}

func getRabbitMQConfigMapData(statefulSetName string, cr *apiv1alpha1.Astarte) map[string]string {
	rmqPlugins := []string{"rabbitmq_management", "rabbitmq_peer_discovery_k8s"}
	if len(cr.Spec.RabbitMQ.AdditionalPlugins) > 0 {
		rmqPlugins = append(rmqPlugins, cr.Spec.RabbitMQ.AdditionalPlugins...)
	}

	rmqConf := `## Clustering
cluster_formation.peer_discovery_backend  = rabbit_peer_discovery_k8s
cluster_formation.k8s.host = kubernetes.default.svc.cluster.local
cluster_formation.k8s.hostname_suffix = .%v.%v.svc.cluster.local
cluster_formation.k8s.address_type = hostname
cluster_formation.node_cleanup.interval = 10
cluster_formation.node_cleanup.only_log_warning = true
cluster_partition_handling = autoheal
## queue master locator 
queue_master_locator=min-masters
## enable guest user  
loopback_users.guest = false
`
	rmqConf = fmt.Sprintf(rmqConf, statefulSetName, cr.Namespace)

	return map[string]string{
		"enabled_plugins": fmt.Sprintf("[%s].\n", strings.Join(rmqPlugins, ",")),
		"rabbitmq.conf":   rmqConf,
	}
}

func getRabbitMQPolicyRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"endpoints"},
			Verbs:     []string{"get"},
		},
	}
}

func getCommonRabbitMQObjectMeta(statefulSetName string, cr *apiv1alpha1.Astarte) metav1.ObjectMeta {
	labels := map[string]string{"app": statefulSetName}
	return metav1.ObjectMeta{
		Name:      statefulSetName,
		Namespace: cr.Namespace,
		Labels:    labels,
	}
}
