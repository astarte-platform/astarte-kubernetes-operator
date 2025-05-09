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
package controllerutils

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	scheduling "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/astarte-platform/astarte-kubernetes-operator/internal/reconcile"
)

// FinalizeAstarte handles the finalization logic for Astarte
func FinalizeAstarte(c client.Client, name, namespace string, reqLogger logr.Logger) error {
	reqLogger.Info("Finalizing Astarte")

	// First of all - do we have the CA Secret still around?
	theSecret := &v1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: name + "-cfssl-ca", Namespace: namespace}, theSecret); err == nil {
		// The secret is there. Delete it.
		if err := c.Delete(context.TODO(), theSecret); err != nil {
			reqLogger.Error(err, "Error while finalizing Astarte. CFSSL CA Secret will need to be manually removed.")
		}
	}

	// Now it's time for our persistent volume claims. Look up all volumes, and see if we need to clear them out.
	// Information in these volumes becomes meaningless after the instance deletion. If one wants to preserve Cassandra, he should take
	// different measures.
	erasePVCPrefixes := []string{
		name + "-vernemq-data",
		name + "-rabbitmq-data",
		name + "-cfssl-data",
		name + "-cassandra-data",
	}

	pvcs := &v1.PersistentVolumeClaimList{}
	if err := c.List(context.TODO(), pvcs, client.InNamespace(namespace)); err == nil {
		// Iterate and delete
		for _, pvc := range pvcs.Items {
			for _, prefix := range erasePVCPrefixes {
				if strings.HasPrefix(pvc.GetName(), prefix) {
					// Delete.
					pvcCopy := pvc
					if e := c.Delete(context.TODO(), &pvcCopy); e != nil {
						reqLogger.Error(e, "Error while finalizing Astarte. A PersistentVolumeClaim will need to be manually removed.", "PVC", pvc)
					}
					break
				}
			}
		}
	} else if !errors.IsNotFound(err) {
		// Notify
		return err
	}

	// Last but not least, we remove the PriorityClasses introduced by Astarte.
	if err := finalizePriorityClasses(c, reqLogger); err != nil {
		return err
	}

	// That's it. So long, and thanks for all the fish.
	reqLogger.Info("Successfully finalized astarte")
	return nil
}

func finalizePriorityClasses(c client.Client, reqLogger logr.Logger) error {
	priorityClasses := &scheduling.PriorityClassList{}
	err := c.List(context.TODO(), priorityClasses)
	if err == nil {
		// Iterate and delete
		for _, priorityClass := range priorityClasses.Items {
			if priorityClass.GetName() == reconcile.AstarteHighPriorityName ||
				priorityClass.GetName() == reconcile.AstarteMidPriorityName ||
				priorityClass.GetName() == reconcile.AstarteLowPriorityName {
				priorityClassCopy := priorityClass
				if err2 := c.Delete(context.TODO(), &priorityClassCopy); err2 != nil {
					reqLogger.Error(err2, "Error while finalizing Astarte. A PriorityClass will need to be manually removed.", "PriorityClass", priorityClass)
					return err2
				}
			}
		}
	} else if !errors.IsNotFound(err) {
		// Notify
		return err
	}
	// All good
	return nil
}
