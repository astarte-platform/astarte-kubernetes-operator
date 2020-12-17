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

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const landing011Version string = "0.11.0"

// blindly upgrades to 0.11. Invokable only by the upgrade logic
func upgradeTo011(cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) error {
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	// Follow the script!
	reqLogger.Info("Upgrading Astarte to the 0.11.x series. The cluster might become partially unresponsive during the process")
	reqLogger.Info("The broker will be brought down during reconciliation. Over the upgrade process, Devices won't be able to connect")

	// Step 1: Stop the Broker
	if err := shutdownVerneMQ(cr, c, recorder); err != nil {
		return err
	}

	// Step 2: Drain RabbitMQ Queues
	if err := drainRabbitMQQueues(cr, c, recorder); err != nil {
		return err
	}

	// Step 3: Migrate the Database
	// It is now time to reconcile selectively Housekeeping and Housekeeping API to a safe landing (0.11.0).
	// Also, we want to bring up exactly one Replica of each at this time.
	// By doing so, Cassandra will be migrated and the cluster will be ready to be reconciled entirely.
	// Version enforcement is done to ensure that jump upgrades will be performed sequentially.
	// Also, given VerneMQ is shutdown at the moment, we give Housekeeping (backend) more juice temporarily by adding to its resource pool
	// VerneMQ's resources. When reconciling later, everything should just settle automagically.
	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
		"Starting Database Migration")
	reqLogger.Info("Upgrading Housekeeping and migrating the Database...")
	housekeepingBackend, err := upgradeHousekeeping(landing011Version, true, cr, c, scheme, recorder)
	if err != nil {
		return err
	}

	// Now we wait indefinitely until this is done. Upgrading the Database might take *a lot* of time, so unless we enter in
	// weird states such as CrashLoopBackoff, we wait almost forever
	if err := waitForHousekeepingUpgrade(cr, c, recorder); err != nil {
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
	if err := waitForQueueLayoutMigration(landing011Version, cr, c, scheme); err != nil {
		recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
			"Could not migrate data queues. Upgrade will be retried, but manual intervention is likely required")
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
	if err := scaleDownHousekeeping(housekeepingBackend, cr, c, scheme); err != nil {
		reqLogger.Error(err, "Failed in waiting for Housekeeping to go down. Continuing anyway.")
	}

	// All done! Upgraded successfully. Now let the standard reconciliation workflow do the rest.
	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
		"Astarte upgraded successfully to the 0.11.x series")
	return nil
}
