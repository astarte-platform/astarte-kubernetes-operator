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

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const landing10Version string = "1.0-snapshot"

// blindly upgrades to 1.0. Invokable only by the upgrade logic
func upgradeTo10(cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) error {
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	// Follow the script!
	reqLogger.Info("Upgrading Astarte to the 1.0.x series. The cluster might become partially unresponsive during the process")
	reqLogger.Info("At some point, the broker will be brought down during reconciliation. Devices won't be able to connect for a short period of time.")

	// Step 1: Migrate the Database
	// 1.0 allows us to do this "live". So simply upgrade Housekeeping and wait for it to settle while the cluster remains up.
	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
		"Starting Database Migration")
	reqLogger.Info("Upgrading Housekeeping and migrating the Database...")
	if _, err := upgradeHousekeeping(landing10Version, false, cr, c, scheme, recorder); err != nil {
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

	// Step 2: Stop the Broker. We need to upgrade its RPC
	if err := shutdownVerneMQ(cr, c, recorder); err != nil {
		return err
	}

	// Step 3: Drain RabbitMQ Queues, to ensure nothing is left before we move forward.
	if err := drainRabbitMQQueues(cr, c, recorder); err != nil {
		return err
	}

	reqLogger.Info("Your Astarte cluster has been successfully upgraded to the 1.0.x series!")

	// This is it. Do not bring up VerneMQ or anything: the reconciliation will now do the right thing with the right versions.
	// On the other hand, as the update successfully completed, increase the Astarte version in the status to ensure we don't
	// go through this twice.
	cr.Status.AstarteVersion = landing10Version
	if err := c.Status().Update(context.TODO(), cr); err != nil {
		reqLogger.Error(err, "Failed to update Astarte status. The Operator might misbehave")
		return err
	}

	// All done! Upgraded successfully. Now let the standard reconciliation workflow do the rest.
	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
		"Astarte upgraded successfully to the 1.0.x series")
	return nil
}
