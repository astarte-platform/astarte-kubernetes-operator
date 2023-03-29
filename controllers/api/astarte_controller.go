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

package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/astarte-platform/astarte-kubernetes-operator/lib/controllerutils"
	"github.com/astarte-platform/astarte-kubernetes-operator/version"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha2"
)

// AstarteReconciler reconciles a Astarte object
type AstarteReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=api.astarte-platform.org,resources=astartes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.astarte-platform.org,resources=astartes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=services;services/finalizers;endpoints;persistentvolumeclaims;configmaps;secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=apps,resourceNames=astarte-operator,resources=deployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;create
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch
// +kubebuilder:rbac:groups=scheduling.k8s.io,resources=priorityclasses,verbs=get;list;watch;create;delete

//nolint:funlen,gocyclo
func (r *AstarteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("astarte", req.NamespacedName)

	// Fetch the Astarte instance
	instance := &apiv1alpha2.Astarte{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	reqLogger.Info("Reconciling Astarte")

	reconciler := controllerutils.ReconcileHelper{
		Client:   r.Client,
		Recorder: r.Recorder,
		Scheme:   r.Scheme,
	}

	// Are we in manual maintenance mode?
	if instance.Spec.ManualMaintenanceMode {
		// If that is so, compute the status and quit.
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			instance = &apiv1alpha2.Astarte{}
			if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
				return err
			}

			instance.Status = reconciler.ComputeAstarteStatusResource(reqLogger, instance)

			if err := r.Client.Status().Update(ctx, instance); err != nil {
				reqLogger.Error(err, "Failed to update Astarte status.")
				return err
			}
			return nil
		}); err != nil {
			return ctrl.Result{}, err
		}

		// Notify and return
		reqLogger.Info("Astarte Reconciliation skipped due to Manual Maintenance Mode. Hope you know what you're doing!")
		return ctrl.Result{}, nil
	}

	// Are we capable of handling the requested version?
	newAstarteSemVersion, err := version.GetAstarteSemanticVersionFrom(instance.Spec.Version)
	if err != nil {
		// Reconcile every minute if we're here
		r.Recorder.Eventf(instance, "Warning", apiv1alpha2.AstarteResourceEventInconsistentVersion.String(),
			err.Error(), instance.Spec.Version)
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Check if the Astarte instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set. In that case, handle finalization.
	if instance.GetDeletionTimestamp() != nil {
		return r.handleFinalization(instance)
	}

	// Ensure status is coeherent
	if result, err := reconciler.EnsureStatusCoherency(reqLogger, instance, req); err != nil {
		return result, err
	}

	// Add finalizer for this CR
	if !contains(instance.GetFinalizers(), astarteFinalizer) {
		if e := r.addFinalizer(instance); e != nil {
			return ctrl.Result{}, e
		}
	}

	// Check the current version and see if we need to transition to an upgrade.
	switch {
	case instance.Status.AstarteVersion == "":
		reqLogger.Info("Could not determine an existing Astarte version for this Resource. Assuming this is a new installation.")
		r.Recorder.Event(instance, "Normal", apiv1alpha2.AstarteResourceEventStatus.String(), "Starting a brand new Astarte Cluster setup")
	case instance.Status.AstarteVersion == version.SnapshotVersion:
		reqLogger.Info("You are running an Astarte snapshot. Any upgrade phase will be skipped, you hopefully know what you're doing")
	case instance.Status.AstarteVersion != instance.Spec.Version:
		reqLogger.Info("Requested Version and Status Version are different, checking for upgrades...",
			"Version.Old", instance.Status.AstarteVersion, "Version.New", instance.Spec.Version)
		if result, e := reconciler.CheckAndPerformUpgrade(reqLogger, instance, newAstarteSemVersion); e != nil {
			return result, e
		}
	}

	// Run actual reconciliation.
	if err := reconciler.ReconcileAstarteResources(instance); err != nil {
		return ctrl.Result{}, err
	}

	// Update the status
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance := &apiv1alpha2.Astarte{}
		if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
			return err
		}

		instance.Status = reconciler.ComputeAstarteStatusResource(reqLogger, instance)

		if err := r.Client.Status().Update(ctx, instance); err != nil {
			reqLogger.Error(err, "Failed to update Astarte status.")
			return err
		}
		return nil
	}); err != nil {
		return ctrl.Result{}, err
	}

	// Reconciliation was successful. Log a message and return
	reqLogger.Info("Astarte Reconciled successfully")
	return ctrl.Result{}, nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func (r *AstarteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha2.Astarte{}, builder.WithPredicates(pred)).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
