/*
Copyright 2024.

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
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/flow"
)

// FlowReconciler reconciles a Flow object
type FlowReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=api.astarte-platform.org,resources=flows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.astarte-platform.org,resources=flows/status,verbs=get;update;patch

func (r *FlowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("flow", req.NamespacedName)
	reqLogger.Info("Reconciling Flow")

	// Fetch the Flow instance
	instance := &apiv1alpha2.Flow{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Get the rest
	astarte, existingBlocks, reconcileResult, err := r.getResourcesForReconciliationFor(instance)
	if err != nil {
		return reconcileResult, err
	}

	// Reconcile all Blocks
	for _, block := range instance.Spec.ContainerBlocks {
		if e := flow.EnsureBlock(instance, block, astarte, r.Client, r.Scheme, reqLogger); e != nil {
			return reconcile.Result{}, e
		}

		delete(existingBlocks, flow.GenerateBlockName(instance, block, astarte))
	}

	// Any leftovers we should delete?
	for _, b := range existingBlocks {
		block := b
		if e := r.Client.Delete(ctx, &block); e != nil {
			return reconcile.Result{}, e
		}
	}

	// Update the Status and finish the reconciliation
	if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance = &apiv1alpha2.Flow{}
		if err = r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
			return err
		}

		instance.Status, err = r.computeFlowStatusResource(reqLogger, instance)
		if err != nil {
			reqLogger.Error(err, "Failed to compute Flow status.")
			return err
		}

		if err = r.Client.Status().Update(ctx, instance); err != nil {
			reqLogger.Error(err, "Failed to update Flow status.")
			return err
		}
		return nil
	}); err != nil {
		return ctrl.Result{}, err
	}

	reqLogger.Info("Flow Reconciled successfully")
	return ctrl.Result{}, nil
}

func (r *FlowReconciler) computeFlowStatusResource(reqLogger logr.Logger, instance *apiv1alpha2.Flow) (apiv1alpha2.FlowStatus, error) {
	newStatus := instance.Status

	// Compute the Status by fetching the Block Deployments, after the cleanup.
	blockList, err := r.getAllBlocksDeploymentsForFlow(instance)
	if err != nil {
		return apiv1alpha2.FlowStatus{}, err
	}

	newStatus.TotalContainerBlocks, newStatus.ReadyContainerBlocks, newStatus.FailingContainerBlocks,
		newStatus.Resources, newStatus.UnrecoverableFailures = r.computeBlocksState(reqLogger, blockList, instance)

	switch {
	case instance.Status.TotalContainerBlocks == 0:
		instance.Status.State = apiv1alpha2.FlowStateUnknown
	case instance.Status.FailingContainerBlocks > 0:
		instance.Status.State = apiv1alpha2.FlowStateUnhealthy
	case instance.Status.TotalContainerBlocks != instance.Status.ReadyContainerBlocks:
		instance.Status.State = apiv1alpha2.FlowStateUnstable
	case instance.Status.TotalContainerBlocks == instance.Status.ReadyContainerBlocks:
		instance.Status.State = apiv1alpha2.FlowStateFlowing
	}

	return newStatus, nil
}

func (r *FlowReconciler) getResourcesForReconciliationFor(instance *apiv1alpha2.Flow) (*apiv1alpha2.Astarte, map[string]appsv1.Deployment, reconcile.Result, error) {
	// Get the Astarte instance for the Flow
	astarte := &apiv1alpha2.Astarte{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.Astarte.Name, Namespace: instance.Namespace}, astarte); err != nil {
		if errors.IsNotFound(err) {
			d, _ := time.ParseDuration("30s")
			return nil, nil, reconcile.Result{Requeue: true, RequeueAfter: d},
				fmt.Errorf("The Astarte Instance %s associated to this Flow cannot be found", instance.Spec.Astarte)
		}
		// Error reading the object - requeue the request.
		return nil, nil, reconcile.Result{}, err
	}

	// Get all existing blocks belonging to this Flow
	existingBlocks, err := r.getAllBlocksForFlow(instance)
	if err != nil {
		return nil, nil, reconcile.Result{}, err
	}

	return astarte, existingBlocks, reconcile.Result{}, err
}

func (r *FlowReconciler) getAllBlocksForFlow(instance *apiv1alpha2.Flow) (map[string]appsv1.Deployment, error) {
	blockList, err := r.getAllBlocksDeploymentsForFlow(instance)
	if err != nil {
		return nil, err
	}

	existingBlocks := map[string]appsv1.Deployment{}
	for _, b := range blockList.Items {
		existingBlocks[b.Name] = b
	}

	return existingBlocks, nil
}

func (r *FlowReconciler) getAllBlocksDeploymentsForFlow(instance *apiv1alpha2.Flow) (appsv1.DeploymentList, error) {
	blockLabels := map[string]string{
		"component":      "astarte-flow",
		"flow-component": "block",
		"flow-name":      instance.Name,
	}
	blockList := appsv1.DeploymentList{}
	err := r.Client.List(context.TODO(), &blockList, client.InNamespace(instance.Namespace), client.MatchingLabels(blockLabels))

	return blockList, err
}

func (r *FlowReconciler) computeBlocksState(reqLogger logr.Logger, blockList appsv1.DeploymentList, instance *apiv1alpha2.Flow) (int, int, int, v1.ResourceList, []v1.ContainerState) {
	var totalBlocks, readyBlocks, failingBlocks int
	cpuResources := instance.Spec.NativeBlocksResources.Cpu().DeepCopy()
	memoryResources := instance.Spec.NativeBlocksResources.Memory().DeepCopy()
	unrecoverableFailures := []v1.ContainerState{}
	for _, b := range blockList.Items {
		totalBlocks++
		for _, c := range b.Spec.Template.Spec.Containers {
			if c.Resources.Requests.Cpu() != nil {
				cpuResources.Add(*c.Resources.Requests.Cpu())
			}
			if c.Resources.Requests.Memory() != nil {
				memoryResources.Add(*c.Resources.Requests.Memory())
			}
		}
		if b.Status.ReadyReplicas == b.Status.Replicas {
			readyBlocks++
		} else {
			podList := &v1.PodList{}
			if err := r.Client.List(context.TODO(), podList, client.InNamespace(instance.Namespace),
				client.MatchingLabels{"flow-block": b.Labels["flow-block"]}); err != nil {
				reqLogger.Error(err, "Could not fetch pods for Block", "block", b.Name)
				continue
			}

			// Grab failures and the likes
			podsFailure, unrecoverablePodsFailures := computePodsFailureForBlock(podList)
			if podsFailure {
				failingBlocks++
				unrecoverableFailures = append(unrecoverableFailures, unrecoverablePodsFailures...)
			}
		}
	}

	return totalBlocks, readyBlocks, failingBlocks, v1.ResourceList{v1.ResourceCPU: cpuResources, v1.ResourceMemory: cpuResources}, unrecoverableFailures
}

func computePodsFailureForBlock(podList *v1.PodList) (bool, []v1.ContainerState) {
	var podsFailure bool
	unrecoverableFailures := []v1.ContainerState{}

	// Inspect the list!
	for _, pod := range podList.Items {
		if pod.Status.Phase == v1.PodRunning {
			// If the pod is running, we're not in a scenario where we're facing a persistent failure.
			continue
		}
		for _, cS := range pod.Status.ContainerStatuses {
			switch {
			case cS.State.Terminated != nil:
				podsFailure = true
				unrecoverableFailures = append(unrecoverableFailures, cS.State)
			case cS.State.Waiting != nil:
				if strings.HasPrefix(cS.State.Waiting.Reason, "Err") {
					// Assume this is a unrecoverable error and the Waiting state is just there not to
					// overload the scheduler with unsatisfiable requests.
					podsFailure = true
					unrecoverableFailures = append(unrecoverableFailures, cS.State)
				}
			}
		}
	}

	return podsFailure, unrecoverableFailures
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha2.Flow{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&v1.Secret{}).
		Complete(r)
}
