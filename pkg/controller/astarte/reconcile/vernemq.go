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

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	semver "github.com/Masterminds/semver/v3"
	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/misc"
	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
		service.Spec.ClusterIP = "None"
		service.Spec.Ports = []v1.ServicePort{
			v1.ServicePort{
				Name:       "mqtt",
				Port:       1883,
				TargetPort: intstr.FromString("mqtt"),
				Protocol:   v1.ProtocolTCP,
			},
			v1.ServicePort{
				Name:       "mqtt-reverse",
				Port:       1885,
				TargetPort: intstr.FromString("mqtt-reverse"),
				Protocol:   v1.ProtocolTCP,
			},
		}
		service.Spec.Selector = labels
		// Add Annotations for Voyager (when deployed)
		service.Annotations = map[string]string{
			"ingress.appscode.com/send-proxy": "v2-ssl-cn",
			"ingress.appscode.com/check":      "true",
		}
		return nil
	}); err == nil {
		logCreateOrUpdateOperationResult(result, cr, service)
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
				Labels: labels,
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

	logCreateOrUpdateOperationResult(result, cr, service)
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
		Handler:             v1.Handler{HTTPGet: &v1.HTTPGetAction{Path: "/metrics", Port: intstr.FromInt(8888)}},
		InitialDelaySeconds: 60,
		TimeoutSeconds:      10,
		PeriodSeconds:       20,
		FailureThreshold:    3,
	}
}

func getVerneMQEnvVars(statefulSetName string, cr *apiv1alpha1.Astarte) []v1.EnvVar {
	userCredentialsSecretName, userCredentialsSecretUsernameKey, userCredentialsSecretPasswordKey := misc.GetRabbitMQUserCredentialsSecret(cr)
	rabbitMQHost, _ := misc.GetRabbitMQHostnameAndPort(cr)
	dataQueueCount := getDataQueueCount(cr)

	envVars := []v1.EnvVar{
		v1.EnvVar{
			Name:      "MY_POD_NAME",
			ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.name"}},
		},
		v1.EnvVar{
			Name:      "MY_POD_IP",
			ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "status.podIP"}},
		},
		v1.EnvVar{
			Name:  "DOCKER_VERNEMQ_DISCOVERY_KUBERNETES",
			Value: "1",
		},
		v1.EnvVar{
			Name:  "DOCKER_VERNEMQ_KUBERNETES_LABEL_SELECTOR",
			Value: "app=" + statefulSetName,
		},
		v1.EnvVar{
			Name:  "DOCKER_VERNEMQ_ASTARTE_VMQ_PLUGIN__AMQP__HOST",
			Value: rabbitMQHost,
		},
		v1.EnvVar{
			Name: "DOCKER_VERNEMQ_ASTARTE_VMQ_PLUGIN__AMQP__USERNAME",
			ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
				Key:                  userCredentialsSecretUsernameKey,
			}},
		},
		v1.EnvVar{
			Name: "DOCKER_VERNEMQ_ASTARTE_VMQ_PLUGIN__AMQP__PASSWORD",
			ValueFrom: &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: userCredentialsSecretName},
				Key:                  userCredentialsSecretPasswordKey,
			}},
		},
	}

	c, _ := semver.NewConstraint(">= 0.11.0")
	ver, _ := semver.NewVersion(getVersionForAstarteComponent(cr, cr.Spec.VerneMQ.Version))
	checkVersion, _ := ver.SetPrerelease("")

	if c.Check(&checkVersion) {
		// When installing Astarte >= 0.11, add the data queue count
		envVars = append(envVars, v1.EnvVar{
			Name:  "DOCKER_VERNEMQ_ASTARTE_VMQ_PLUGIN__DATA_QUEUE_COUNT",
			Value: strconv.Itoa(dataQueueCount),
		})
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
			v1.Container{
				Name: "vernemq",
				VolumeMounts: []v1.VolumeMount{
					v1.VolumeMount{
						Name:      dataVolumeName,
						MountPath: "/opt/vernemq/data",
					},
				},
				// Defaults to the custom image built in Astarte
				Image:           getAstarteImageForClusteredResource("vernemq", cr.Spec.VerneMQ.AstarteGenericClusteredResource, cr),
				ImagePullPolicy: getImagePullPolicy(cr),
				Ports: []v1.ContainerPort{
					v1.ContainerPort{Name: "mqtt-ssl", ContainerPort: 8883},
					v1.ContainerPort{Name: "acme-verify", ContainerPort: 80},
					v1.ContainerPort{Name: "mqtt", ContainerPort: 1883},
					v1.ContainerPort{Name: "mqtt-reverse", ContainerPort: 1885},
					v1.ContainerPort{Name: "vmq-msg-dist", ContainerPort: 44053},
					v1.ContainerPort{Name: "epmd", ContainerPort: 4369},
					v1.ContainerPort{Name: "metrics", ContainerPort: 8888},
					v1.ContainerPort{ContainerPort: 9100},
					v1.ContainerPort{ContainerPort: 9101},
					v1.ContainerPort{ContainerPort: 9102},
					v1.ContainerPort{ContainerPort: 9103},
					v1.ContainerPort{ContainerPort: 9104},
					v1.ContainerPort{ContainerPort: 9105},
					v1.ContainerPort{ContainerPort: 9106},
					v1.ContainerPort{ContainerPort: 9107},
					v1.ContainerPort{ContainerPort: 9108},
					v1.ContainerPort{ContainerPort: 9109},
				},
				LivenessProbe:  getVerneMQProbe(),
				ReadinessProbe: getVerneMQProbe(),
				Resources:      resources,
				Env:            getVerneMQEnvVars(statefulSetName, cr),
			},
		},
	}

	return ps
}

func getVerneMQPolicyRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"pods", "services"},
			Verbs:     []string{"list", "get"},
		},
	}
}
