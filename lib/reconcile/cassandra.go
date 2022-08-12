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
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/deps"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
)

func getCassandraUserAndPassword(conn *apiv1alpha1.AstarteCassandraConnectionSpec) (string, string) {
	if conn != nil {
		return conn.Username, conn.Password
	}
	return "", ""
}

func getCassandraUserAndPasswordKeys(conn *apiv1alpha1.AstarteCassandraConnectionSpec) (string, string) {
	if conn != nil {
		if conn.Secret != nil {
			return conn.Secret.UsernameKey, conn.Secret.PasswordKey
		}
	}
	return misc.CassandraDefaultUserCredentialsUsernameKey, misc.CassandraDefaultUserCredentialsPasswordKey
}

func getCassandraSecret(cr *apiv1alpha1.Astarte) *apiv1alpha1.LoginCredentialsSecret {
	if cr.Spec.Cassandra.Connection != nil {
		return cr.Spec.Cassandra.Connection.Secret
	}
	return nil
}

// EnsureCassandra reconciles Cassandra
func EnsureCassandra(cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	//reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	statefulSetName := cr.Name + "-cassandra"
	labels := map[string]string{"app": statefulSetName}

	// Validate where necessary
	if err := validateCassandraDefinition(cr.Spec.Cassandra); err != nil {
		return err
	}

	// Depending on the situation, we need to take action on the credentials.
	secretName := cr.Name + "-cassandra-user-credentials"
	username, password := getCassandraUserAndPassword(cr.Spec.Cassandra.Connection)
	usernameKey, passwordKey := getCassandraUserAndPasswordKeys(cr.Spec.Cassandra.Connection)
	secret := getCassandraSecret(cr)
	forceCredentialsCreation := false

	if err := handleGenericUserCredentialsSecret(username, password, usernameKey, passwordKey, secretName, forceCredentialsCreation, secret, cr, c, scheme); err != nil {
		return err
	}

	// Ok. Shall we deploy?
	if !pointy.BoolValue(cr.Spec.Cassandra.Deploy, true) {
		log.Info("Skipping Cassandra Deployment")
		// Before returning - check if we shall clean up the StatefulSet.
		// It is the only thing actually requiring resources, the rest will be cleaned up eventually when the
		// Astarte resource is deleted.
		theStatefulSet := &appsv1.StatefulSet{}
		err := c.Get(context.TODO(), types.NamespacedName{Name: statefulSetName, Namespace: cr.Namespace}, theStatefulSet)
		if err == nil {
			log.Info("Deleting previously existing Cassandra StatefulSet, which is no longer needed")
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
		service.Spec.ClusterIP = noneClusterIP
		service.Spec.Ports = []v1.ServicePort{
			{
				Name:       "cql",
				Port:       9042,
				TargetPort: intstr.FromString("cql"),
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

	// Let's check upon Storage now.
	dataVolumeName, persistentVolumeClaim := computePersistentVolumeClaim(statefulSetName+"-data", resource.NewScaledQuantity(30, resource.Giga),
		cr.Spec.Cassandra.Storage, cr)

	// Compute and prepare all data for building the StatefulSet
	statefulSetSpec := appsv1.StatefulSetSpec{
		ServiceName: statefulSetName,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: computePodLabels(cr.Spec.Cassandra.AstarteGenericClusteredResource, labels),
			},
			Spec: getCassandraPodSpec(statefulSetName, dataVolumeName, cr),
		},
	}

	if persistentVolumeClaim != nil {
		statefulSetSpec.VolumeClaimTemplates = []v1.PersistentVolumeClaim{*persistentVolumeClaim}
	}

	// Build the StatefulSet
	cassandraStatefulSet := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: statefulSetName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, cassandraStatefulSet, func() error {
		if err := controllerutil.SetControllerReference(cr, cassandraStatefulSet, scheme); err != nil {
			return err
		}

		// Assign the Spec.
		cassandraStatefulSet.Spec = statefulSetSpec
		cassandraStatefulSet.Spec.Replicas = cr.Spec.Cassandra.Replicas

		return nil
	})
	if err != nil {
		return err
	}

	misc.LogCreateOrUpdateOperationResult(log, result, cr, service)
	return nil
}

func validateCassandraDefinition(cassandra apiv1alpha1.AstarteCassandraSpec) error {
	if !pointy.BoolValue(cassandra.Deploy, true) && cassandra.Nodes == "" {
		return errors.New("When not deploying Cassandra, the 'nodes' must be specified")
	}

	// All is good.
	return nil
}

func getCassandraProbe() *v1.Probe {
	// Start checking after 1 minute, every 20 seconds, fail after the 3rd attempt
	return &v1.Probe{
		ProbeHandler:        v1.ProbeHandler{Exec: &v1.ExecAction{Command: []string{"/bin/bash", "-c", "/ready-probe.sh"}}},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      5,
		PeriodSeconds:       20,
		FailureThreshold:    3,
	}
}

func getCassandraEnvVars(statefulSetName string, cr *apiv1alpha1.Astarte) []v1.EnvVar {
	maxHeapSize := "1024M"
	heapNewSize := "256M"
	if cr.Spec.Cassandra.MaxHeapSize != "" {
		maxHeapSize = cr.Spec.Cassandra.MaxHeapSize
	}
	if cr.Spec.Cassandra.HeapNewSize != "" {
		heapNewSize = cr.Spec.Cassandra.HeapNewSize
	}

	envVars := []v1.EnvVar{
		{
			Name:      "POD_NAME",
			ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.name"}},
		},
		{
			Name:      "POD_IP",
			ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "status.podIP"}},
		},
		{
			Name:  "CASSANDRA_SEEDS",
			Value: fmt.Sprintf("%s-0.%s.%s.svc.cluster.local", statefulSetName, statefulSetName, cr.Namespace),
		},
		{
			Name:  "CASSANDRA_CLUSTER_NAME",
			Value: "AstarteCassandra",
		},
		{
			Name:  "CASSANDRA_DC",
			Value: "DC1-AstarteCassandra",
		},
		{
			Name:  "CASSANDRA_RACK",
			Value: "Rack1-AstarteCassandra",
		},
		{
			Name:  "MAX_HEAP_SIZE",
			Value: maxHeapSize,
		},
		{
			Name:  "HEAP_NEWSIZE",
			Value: heapNewSize,
		},
	}

	// Add any explicit additional env
	if len(cr.Spec.Cassandra.AdditionalEnv) > 0 {
		envVars = append(envVars, cr.Spec.Cassandra.AdditionalEnv...)
	}

	return envVars
}

func getCassandraPodSpec(statefulSetName, dataVolumeName string, cr *apiv1alpha1.Astarte) v1.PodSpec {
	resources := v1.ResourceRequirements{}
	if cr.Spec.Cassandra.Resources != nil {
		resources = *cr.Spec.Cassandra.Resources
	}

	ps := v1.PodSpec{
		// Give it a lot of time to terminate to drain the node.
		TerminationGracePeriodSeconds: pointy.Int64(1800),
		ImagePullSecrets:              cr.Spec.ImagePullSecrets,
		Affinity:                      getAffinityForClusteredResource(statefulSetName, cr.Spec.Cassandra.AstarteGenericClusteredResource),
		Containers: []v1.Container{
			{
				Name: "cassandra",
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      dataVolumeName,
						MountPath: "/cassandra_data",
					},
				},
				Image: getImageForClusteredResource("gcr.io/google-samples/cassandra", deps.GetDefaultVersionForCassandra(cr.Spec.Version),
					cr.Spec.Cassandra.AstarteGenericClusteredResource),
				ImagePullPolicy: getImagePullPolicy(cr),
				Ports: []v1.ContainerPort{
					{Name: "intra-node", ContainerPort: 7000},
					{Name: "tls-intra-node", ContainerPort: 7001},
					{Name: "jmx", ContainerPort: 7199},
					{Name: "cql", ContainerPort: 9042},
				},
				ReadinessProbe: getCassandraProbe(),
				Resources:      resources,
				Env:            getCassandraEnvVars(statefulSetName, cr),
				SecurityContext: &v1.SecurityContext{
					Capabilities: &v1.Capabilities{Add: []v1.Capability{"IPC_LOCK"}},
				},
				Lifecycle: &v1.Lifecycle{PreStop: &v1.LifecycleHandler{Exec: &v1.ExecAction{Command: []string{"/bin/sh", "-c", "nodetool drain"}}}},
			},
		},
	}

	return ps
}

// This stuff is useful for other components which need to interact with Cassandra
func getCassandraNodes(cr *apiv1alpha1.Astarte) string {
	replicas := cr.Spec.Cassandra.Replicas
	if cr.Spec.Cassandra.Nodes != "" {
		return cr.Spec.Cassandra.Nodes
	}

	// We're on defaults then. Give all the fully qualified nodes, joined by a comma.
	nodes := []string{}
	nodeNumber := int(pointy.Int32Value(replicas, 1))
	for i := 0; i < nodeNumber; i++ {
		nodes = append(nodes, fmt.Sprintf("%s-cassandra-%d.%s-cassandra.%s.svc.cluster.local:9042", cr.Name, i, cr.Name, cr.Namespace))
	}

	return strings.Join(nodes, ",")
}
