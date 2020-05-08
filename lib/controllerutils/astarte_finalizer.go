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

package controllerutils

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
					if e := c.Delete(context.TODO(), &pvc); e != nil {
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

	// That's it. So long, and thanks for all the fish.
	reqLogger.Info("Successfully finalized astarte")
	return nil
}
