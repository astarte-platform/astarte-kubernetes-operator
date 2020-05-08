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

package astarte

import (
	"context"
	"time"

	"github.com/astarte-platform/astarte-kubernetes-operator/lib/controllerutils"
	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/version"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_astarte")

// Add creates a new Astarte Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAstarte{client: mgr.GetClient(), scheme: mgr.GetScheme(), recorder: mgr.GetEventRecorderFor("astarte-controller")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("astarte-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Astarte
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	}
	if err := c.Watch(&source.Kind{Type: &apiv1alpha1.Astarte{}}, &handler.EnqueueRequestForObject{}, pred); err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployments and requeue the owner Astarte
	if err := c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &apiv1alpha1.Astarte{},
	}); err != nil {
		return err
	}
	// Watch for changes to secondary resource StatefulSet and requeue the owner Astarte
	if err := c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &apiv1alpha1.Astarte{},
	}); err != nil {
		return err
	}
	// Watch for changes to secondary resource Job and requeue the owner Astarte
	if err := c.Watch(&source.Kind{Type: &batchv1.Job{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &apiv1alpha1.Astarte{},
	}); err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileAstarte implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileAstarte{}

// ReconcileAstarte reconciles a Astarte object
type ReconcileAstarte struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a Astarte object and makes changes based on the state read
// and what is in the Astarte.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAstarte) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// Fetch the Astarte instance
	instance := &apiv1alpha1.Astarte{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	reqLogger.Info("Reconciling Astarte")

	reconciler := controllerutils.ReconcileHelper{
		Client:   r.client,
		Recorder: r.recorder,
		Scheme:   r.scheme,
	}

	// Are we capable of handling the requested version?
	newAstarteSemVersion, err := version.GetAstarteSemanticVersionFrom(instance.Spec.Version)
	if err != nil {
		// Reconcile every minute if we're here
		r.recorder.Eventf(instance, "Warning", apiv1alpha1.AstarteResourceEventInconsistentVersion.String(),
			err.Error(), instance.Spec.Version)
		return reconcile.Result{RequeueAfter: time.Minute}, err
	}

	// Check if the Astarte instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set. In that case, handle finalization.
	if instance.GetDeletionTimestamp() != nil {
		return r.handleFinalization(instance)
	}

	// Ensure status is coeherent
	if result, err := reconciler.EnsureStatusCoherency(reqLogger, instance, request); err != nil {
		return result, err
	}

	// Add finalizer for this CR
	if !contains(instance.GetFinalizers(), astarteFinalizer) {
		if e := r.addFinalizer(instance); e != nil {
			return reconcile.Result{}, e
		}
	}

	// Check the current version and see if we need to transition to an upgrade.
	switch {
	case instance.Status.AstarteVersion == "":
		reqLogger.Info("Could not determine an existing Astarte version for this Resource. Assuming this is a new installation.")
		r.recorder.Event(instance, "Normal", apiv1alpha1.AstarteResourceEventStatus.String(), "Starting a brand new Astarte Cluster setup")
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
		return reconcile.Result{}, err
	}

	// Update the status
	instance.Status = reconciler.ComputeAstarteStatusResource(reqLogger, instance)
	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		reqLogger.Error(err, "Failed to update Astarte status.")
		return reconcile.Result{}, err
	}

	// Reconciliation was successful. Log a message and return
	reqLogger.Info("Astarte Reconciled successfully")
	return reconcile.Result{}, nil
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
