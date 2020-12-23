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
	"time"

	semver "github.com/Masterminds/semver/v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
)

var log = logf.Log.WithName("controller_astarte")

// Various constants useful all around the update package.
const (
	// Retry every 20 seconds for all API calls
	retryInterval = time.Second * 20
	// Time out after 5 minutes
	timeout = time.Second * 180
)

// EnsureAstarteUpgrade ensures that CR with requested newVersion will be upgraded from oldVersion, if needed.
func EnsureAstarteUpgrade(oldVersion, newVersion *semver.Version, cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) error {
	// Check 0.10.x -> 0.11.x constraint
	transitionCheck, err := validateConstraintAndPrepareUpgrade(oldVersion, newVersion, "~0.10.0", ">= 0.11.0", cr, c)
	if err != nil {
		return err
	}
	if transitionCheck {
		// Perform upgrade
		recorder.Eventf(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
			"Initiating Astarte 0.11 Upgrade, from version %v to version %v", cr.Status.AstarteVersion, cr.Spec.Version)
		if e := upgradeTo011(cr, c, scheme, recorder); e != nil {
			return e
		}
	} else {
		recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
			"Requested Astarte Upgrade does not require any special action, continuing standard reconciliation")
	}

	// Check 0.11.x -> 1.0.x constraint
	transitionCheck, err = validateConstraintAndPrepareUpgrade(oldVersion, newVersion, "~0.11.0", ">= 1.0.0", cr, c)
	if err != nil {
		return err
	}
	if transitionCheck {
		// Perform upgrade
		recorder.Eventf(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
			"Initiating Astarte 1.0 Upgrade, from version %v to version %v", cr.Status.AstarteVersion, cr.Spec.Version)
		if e := upgradeTo10(cr, c, scheme, recorder); e != nil {
			return e
		}
	} else {
		recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
			"Requested Astarte Upgrade does not require any special action, continuing standard reconciliation")
	}

	// All good if we're here!
	return nil
}

func validateConstraintAndPrepareUpgrade(oldVersion, newVersion *semver.Version, oldConstraintString, newConstraintString string, cr *apiv1alpha1.Astarte, c client.Client) (bool, error) {
	oldConstraint, err := semver.NewConstraint(oldConstraintString)
	if err != nil {
		return false, err
	}
	newConstraint, err := semver.NewConstraint(newConstraintString)
	if err != nil {
		return false, err
	}

	// Remove pre-releases, if part of the version, to enable constraint comparison
	if oldVersion.Prerelease() != "" {
		*oldVersion, _ = oldVersion.SetPrerelease("")
	}
	if newVersion.Prerelease() != "" {
		*newVersion, _ = newVersion.SetPrerelease("")
	}

	if oldConstraint.Check(oldVersion) && newConstraint.Check(newVersion) {
		// Set the Reconciliation Phase to Upgrading
		reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
		reqLogger.Info("Upgrade found, will start Upgrade routine")
		cr.Status.ReconciliationPhase = apiv1alpha1.ReconciliationPhaseUpgrading
		// Update the status
		if err := c.Status().Update(context.TODO(), cr); err != nil {
			reqLogger.Error(err, "Failed to update Astarte Reconciliation Phase status. Not dying for this, though")
			// That's it - no point in failing here.
		}

		return true, nil
	}

	return false, nil
}
