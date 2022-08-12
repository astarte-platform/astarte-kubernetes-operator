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

	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
)

// EnsureAstarteDataUpdaterPlant manages multiple deployments for Astarte Data Updater Plant based on scalability requirements
func EnsureAstarteDataUpdaterPlant(cr *apiv1alpha1.Astarte, dup apiv1alpha1.AstarteDataUpdaterPlantSpec, c client.Client, scheme *runtime.Scheme) error {
	replicas := pointy.Int32Value(dup.AstarteGenericClusteredResource.Replicas, 1)
	component := apiv1alpha1.DataUpdaterPlant

	// Let's list the existing deployments labeled DUP
	currentDUPDeployments := &appsv1.DeploymentList{}
	if err := c.List(context.TODO(), currentDUPDeployments, client.InNamespace(cr.Namespace),
		client.MatchingLabels{"astarte-component": component.DashedString()}); err != nil {
		return err
	}

	if len(currentDUPDeployments.Items) > int(replicas) || !pointy.BoolValue(dup.AstarteGenericClusteredResource.Deploy, true) {
		// In this case, we should schedule for immediate deletion all of the deployments and recreate them.
		if err := c.DeleteAllOf(context.Background(), &appsv1.Deployment{}, client.InNamespace(cr.Namespace),
			client.MatchingLabels{"astarte-component": component.DashedString()}); err != nil {
			return err
		}
		// If we shouldn't deploy, just return now then.
		if !pointy.BoolValue(dup.AstarteGenericClusteredResource.Deploy, true) {
			return nil
		}
	}

	if replicas == 1 {
		// We should treat it as a standard case
		return EnsureAstarteGenericBackend(cr, dup.AstarteGenericClusteredResource, component, c, scheme)
	}

	// If we got here, we need to apply some custom logic to our deployments
	// Reconcile the service first
	serviceName := cr.Name + "-" + component.ServiceName()
	labels := map[string]string{
		"app":                   cr.Name + "-" + component.DashedString(),
		"component":             "astarte",
		"astarte-component":     component.DashedString(),
		"astarte-instance-name": cr.Name,
	}
	matchLabels := map[string]string{"astarte-component": component.DashedString(), "astarte-instance-name": cr.Name}
	if err := createOrUpdateService(cr, c, serviceName, scheme, matchLabels, labels); err != nil {
		return err
	}

	// Now proceed in creating a deployment for each DUP replica with its own set of queues
	for i := 0; i < int(replicas); i++ {
		if err := createIndexedDataUpdaterPlantDeployment(i, int(replicas), cr, dup, c, scheme); err != nil {
			return err
		}
	}

	return nil
}

func createIndexedDataUpdaterPlantDeployment(replicaIndex, replicas int, cr *apiv1alpha1.Astarte, dup apiv1alpha1.AstarteDataUpdaterPlantSpec, c client.Client, scheme *runtime.Scheme) error {
	component := apiv1alpha1.DataUpdaterPlant

	deploymentName := cr.Name + "-" + component.DashedString()
	if replicaIndex > 0 {
		deploymentName = deploymentName + "-" + strconv.Itoa(replicaIndex)
	}

	labels := map[string]string{
		"app":                   deploymentName,
		"component":             "astarte",
		"astarte-component":     component.DashedString(),
		"astarte-instance-name": cr.Name,
	}

	matchLabels := map[string]string{"app": deploymentName}

	// First of all, check if we need to regenerate the cookie.
	if err := ensureErlangCookieSecret(deploymentName+"-cookie", cr, c, scheme); err != nil {
		return err
	}

	deploymentSpec := appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: matchLabels,
		},
		Strategy: getDeploymentStrategyForClusteredResource(cr, dup.AstarteGenericClusteredResource, component),
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: computePodLabels(dup.AstarteGenericClusteredResource, labels),
			},
			Spec: getAstarteGenericBackendPodSpec(deploymentName, replicaIndex, replicas, cr, dup.AstarteGenericClusteredResource, component, nil),
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
		// Always force to 1
		deployment.Spec.Replicas = pointy.Int32(1)

		return nil
	})
	if err != nil {
		return err
	}

	misc.LogCreateOrUpdateOperationResult(log, result, cr, deployment)
	return nil
}
