/*
This file is part of Astarte.

Copyright 2024 SECO Mind Srl.

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

package controller

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/controllerutils"
)

const astarteFinalizer = "astarte.astarte-platform.org/finalizer"

func (r *AstarteReconciler) handleFinalization(instance *v1alpha2.Astarte) (ctrl.Result, error) {
	if contains(instance.GetFinalizers(), astarteFinalizer) {
		// Run finalization logic for astarteFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if e := controllerutils.FinalizeAstarte(r.Client, instance.Name, instance.Namespace,
			r.Log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name)); e != nil {
			return ctrl.Result{}, e
		}

		// Remove astarteFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		instance.SetFinalizers(remove(instance.GetFinalizers(), astarteFinalizer))
		if e := r.Client.Update(context.TODO(), instance); e != nil {
			return ctrl.Result{}, e
		}
	}
	return ctrl.Result{}, nil
}

func (r *AstarteReconciler) addFinalizer(cr *v1alpha2.Astarte) error {
	reqLogger := r.Log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	reqLogger.Info("Adding Astarte Finalizer")
	cr.SetFinalizers(append(cr.GetFinalizers(), astarteFinalizer))

	// Update CR
	err := r.Client.Update(context.TODO(), cr)
	if err != nil {
		reqLogger.Error(err, "Failed to update Astarte with finalizer")
		return err
	}
	return nil
}
