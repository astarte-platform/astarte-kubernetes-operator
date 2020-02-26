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

package upgrade

import (
	"context"
	"fmt"
	"time"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/controller/astarte/reconcile"
	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/misc"
	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: Change this to a stable release as soon as it is generally available.
const landing011Version string = "0.11.0-rc.0"

// blindly upgrades to 0.11. Invokable only by the upgrade logic
func upgradeTo011(cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) error {
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	// Follow the script!
	reqLogger.Info("Upgrading Astarte to the 0.11.x series. The cluster might become partially unresponsive during the process")
	reqLogger.Info("The broker will be brought down during reconciliation. Over the upgrade process, Devices won't be able to connect")

	// Step 1: Stop the Broker
	if err := shutdownVerneMQ(cr, c, scheme, recorder); err != nil {
		return err
	}

	// Step 2: Drain RabbitMQ Queues
	if err := drainRabbitMQQueues(cr, c, scheme, recorder); err != nil {
		return err
	}

	// Step 3: Migrate the Database
	// It is now time to reconcile selectively Housekeeping and Housekeeping API to a safe landing (0.11.0-rc.0 now).
	// Also, we want to bring up exactly one Replica of each at this time.
	// By doing so, Cassandra will be migrated and the cluster will be ready to be reconciled entirely.
	// Version enforcement is done to ensure that jump upgrades will be performed sequentially.
	// Also, given VerneMQ is shutdown at the moment, we give Housekeeping (backend) more juice temporarily by adding to its resource pool
	// VerneMQ's resources. When reconciling later, everything should just settle automagically.
	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
		"Starting Database Migration")
	reqLogger.Info("Upgrading Housekeeping and migrating the Database...")
	housekeepingBackend := cr.Spec.Components.Housekeeping.Backend.DeepCopy()
	housekeepingBackend.Version = landing011Version
	housekeepingBackend.Replicas = pointy.Int32(1)
	// Ensure the policy is Replace. We don't want to have old pods hanging around.
	housekeepingBackend.DeploymentStrategy = &appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
	if misc.IsResourceRequirementsExplicit(cr.Spec.VerneMQ.Resources) {
		resourceRequirements := misc.GetResourcesForAstarteComponent(cr, housekeepingBackend.Resources, apiv1alpha1.Housekeeping)
		resourceRequirements.Requests.Cpu().Add(*cr.Spec.VerneMQ.Resources.Requests.Cpu())
		resourceRequirements.Requests.Memory().Add(*cr.Spec.VerneMQ.Resources.Requests.Memory())
		resourceRequirements.Limits.Cpu().Add(*cr.Spec.VerneMQ.Resources.Limits.Cpu())
		resourceRequirements.Limits.Memory().Add(*cr.Spec.VerneMQ.Resources.Limits.Memory())

		// This way, on the next call to GetResourcesForAstarteComponent, these resources will be returned as explicitly stated
		// in the original spec.
		housekeepingBackend.Resources = resourceRequirements
	}
	// TODO: When we move to 0.11.0-beta3 or above, add a Probe
	if err := reconcile.EnsureAstarteGenericBackend(cr, *housekeepingBackend, apiv1alpha1.Housekeeping, c, scheme); err != nil {
		recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventUpgradeError.String(),
			"Could not initiate Database Migration. Upgrade will be retried")
		return err
	}
	housekeepingAPI := cr.Spec.Components.Housekeeping.API.DeepCopy()
	housekeepingAPI.Replicas = pointy.Int32(1)
	housekeepingAPI.Version = landing011Version
	// Ensure the policy is Replace. We don't want to have old pods hanging around.
	housekeepingAPI.DeploymentStrategy = &appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
	if err := reconcile.EnsureAstarteGenericAPIWithCustomProbe(cr, *housekeepingAPI, apiv1alpha1.HousekeepingAPI, c,
		scheme, getSpecialHousekeepingMigrationProbe("/health")); err != nil {
		recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventUpgradeError.String(),
			"Could not initiate Database Migration. Upgrade will be retried")
		return err
	}

	// Now we wait indefinitely until this is done. Upgrading the Database might take *a lot* of time, so unless we enter in
	// weird states such as CrashLoopBackoff, we wait almost forever
	weirdFailuresCount := 0
	weirdFailuresThreshold := 10
	if err := wait.Poll(retryInterval, time.Hour, func() (done bool, err error) {
		deployment := &appsv1.Deployment{}
		if err = c.Get(context.TODO(), types.NamespacedName{Name: cr.Name + "-housekeeping-api", Namespace: cr.Namespace}, deployment); err != nil {
			weirdFailuresCount++
			if weirdFailuresCount > weirdFailuresThreshold {
				// Something is off.
				recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
					"Repeated errors in monitoring Database Migration. Manual intervention is likely required")
				return false, fmt.Errorf("Failed in looking up Housekeeping API Deployment. Most likely, manual intervention is required. %v", err)
			}
			// Something is off.
			log.Error(err, "Failed in looking up Housekeeping API Deployment. This might be a temporary problem - will retry")
			return false, nil
		}

		if deployment.Status.ReadyReplicas >= 1 {
			// That's it bros.
			return true, nil
		}

		// Ensure we aren't in the position where Housekeeping itself is crashing.
		housekeepingComponent := apiv1alpha1.Housekeeping
		podList := &v1.PodList{}
		if err = c.List(context.TODO(), podList, client.InNamespace(cr.Namespace),
			client.MatchingLabels{"astarte-component": housekeepingComponent.DashedString()}); err != nil {
			weirdFailuresCount++
			if weirdFailuresCount > weirdFailuresThreshold {
				// Something is off.
				recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
					"Repeated errors in monitoring Database Migration. Manual intervention is likely required")
				return false, fmt.Errorf("Failed in looking up Housekeeping pods. Most likely, manual intervention is required. %v", err)
			}
			// Something is off.
			log.Error(err, "Failed in looking up Housekeeping pods. This might be a temporary problem - will retry")
			return false, nil
		}

		// Inspect the list!
		if len(podList.Items) != 1 {
			weirdFailuresCount++
			if weirdFailuresCount > weirdFailuresThreshold {
				// Something is off.
				recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
					"Repeated errors in monitoring Database Migration. Manual intervention is likely required")
				return false, fmt.Errorf("%v Housekeeping pods found. Most likely, manual intervention is required. %v", len(podList.Items), err)
			}
			// Something is off.
			log.Error(err, fmt.Sprintf("%v Housekeeping pods found. This might be a temporary problem - will retry", len(podList.Items)))
			return false, nil
		}

		if len(podList.Items[0].Status.ContainerStatuses) != 1 {
			weirdFailuresCount++
			if weirdFailuresCount > weirdFailuresThreshold {
				// Something is off.
				recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
					"Repeated errors in monitoring Database Migration. Manual intervention is likely required")
				return false, fmt.Errorf("%v Container Statuses retrieved. Most likely, manual intervention is required. %v", len(podList.Items[0].Status.ContainerStatuses), err)
			}
			// Something is off.
			log.Error(err, fmt.Sprintf("%v Container Statuses retrieved. This might be a temporary problem - will retry", len(podList.Items[0].Status.ContainerStatuses)))
			return false, nil
		}

		if podList.Items[0].Status.ContainerStatuses[0].State.Waiting != nil {
			if podList.Items[0].Status.ContainerStatuses[0].State.Waiting.Reason == "CrashLoopBackoff" {
				recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
					"Database Migration failed. Manual intervention is likely required")
				return true, fmt.Errorf("Housekeeping is crashing repeatedly. There has to be a problem in handling Database migrations. Please take manual action as soon as possible")
			}
		}

		return false, nil
	}); err != nil {
		recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
			"Timed out waiting for Database Migration. Upgrade will be retried, but manual intervention is likely required")
		return fmt.Errorf("Failed in waiting for Housekeeping deployment and migrations to go up: %v", err)
	}
	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
		"Database migrated successfully")
	reqLogger.Info("Database successfully migrated!")

	// Step 4: Migrate the Queue Layout
	// If we got here, we're almost there. Now we need to bring up Data Updater Plant and wait for it to become ready
	// to ensure the consistency of RabbitMQ queues.
	// Again, same thing as before: hook to a known version. There's no need to add more
	// resources to DUP as it doesn't need them to perform this operation, and most of all it should have enough sauce already.
	reqLogger.Info("Ensuring new RabbitMQ Queue Layout through Data Updater Plant...")
	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
		"Starting Queue layout migration")
	dataUpdaterPlant := cr.Spec.Components.DataUpdaterPlant.DeepCopy()
	dataUpdaterPlant.Version = landing011Version
	// Ensure the policy is Replace. We don't want to have old pods hanging around.
	dataUpdaterPlant.DeploymentStrategy = &appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
	if err := reconcile.EnsureAstarteGenericBackend(cr, dataUpdaterPlant.AstarteGenericClusteredResource, apiv1alpha1.DataUpdaterPlant, c, scheme); err != nil {
		return err
	}
	// Again, the operation should be pretty normal. Wait with standard timeouts here
	if err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		deployment := &appsv1.Deployment{}
		if err = c.Get(context.TODO(), types.NamespacedName{Name: cr.Name + "-data-updater-plant", Namespace: cr.Namespace}, deployment); err != nil {
			return false, err
		}

		if deployment.Status.ReadyReplicas > 0 {
			return true, nil
		}

		return false, nil
	}); err != nil {
		recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
			"Timed out while waiting for queues to be migrated. Upgrade will be retried, but manual intervention is likely required")
		return fmt.Errorf("Failed in waiting for Data Updater Plant to come up: %v", err)
	}
	reqLogger.Info("RabbitMQ queues layout upgrade successful!")
	reqLogger.Info("Your Astarte cluster has been successfully upgraded to the 0.11.x series!")

	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
		"Queues migrated successfully")

	// This is it. Do not bring up VerneMQ or anything: the reconciliation will now do the right thing with the right versions.
	// On the other hand, as the update successfully completed, increase the Astarte version in the status to ensure we don't
	// go through this twice.
	cr.Status.AstarteVersion = landing011Version
	if err := c.Status().Update(context.TODO(), cr); err != nil {
		reqLogger.Error(err, "Failed to update Astarte status. The Operator might misbehave")
		return err
	}

	// Just to be sure, scale down Housekeeping to 0 replicas. If we're *really* tight on resources, it might be that
	// the additional pool prevents other pods from coming up.
	reqLogger.Info("Restoring original environment and waiting for cluster to settle...")
	housekeepingBackend.Replicas = pointy.Int32(0)
	if err := reconcile.EnsureAstarteGenericBackend(cr, *housekeepingBackend, apiv1alpha1.Housekeeping, c, scheme); err != nil {
		return err
	}
	// Wait for it to go down, then we should be good to go.
	if err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		deployment := &appsv1.Deployment{}
		if err = c.Get(context.TODO(), types.NamespacedName{Name: cr.Name + "-housekeeping", Namespace: cr.Namespace}, deployment); err != nil {
			return false, err
		}

		if deployment.Status.ReadyReplicas > 0 {
			return false, nil
		}

		return true, nil
	}); err != nil {
		reqLogger.Error(err, "Failed in waiting for Housekeeping to go down. Continuing anyway.")
	}

	// All done! Upgraded successfully. Now let the standard reconciliation workflow do the rest.
	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
		"Astarte upgraded successfully to the 0.11.x series")
	return nil
}
