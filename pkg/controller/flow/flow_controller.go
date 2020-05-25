package flow

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_flow")

// Add creates a new Flow Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileFlow{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("flow-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Flow
	err = c.Watch(&source.Kind{Type: &apiv1alpha1.Flow{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployments and requeue the owner Flow
	if err := c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &apiv1alpha1.Flow{},
	}); err != nil {
		return err
	}
	// Watch for changes to secondary resource StatefulSet and requeue the owner Flow
	if err := c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &apiv1alpha1.Flow{},
	}); err != nil {
		return err
	}
	// Watch for changes to secondary resource Secret and requeue the owner Flow
	if err := c.Watch(&source.Kind{Type: &v1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &apiv1alpha1.Flow{},
	}); err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileFlow implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileFlow{}

// ReconcileFlow reconciles a Flow object
type ReconcileFlow struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Flow object and makes changes based on the state read
// and what is in the Flow.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileFlow) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Flow")

	// Fetch the Flow instance
	instance := &apiv1alpha1.Flow{}
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

	// Get the rest
	astarte, existingBlocks, reconcileResult, err := r.getResourcesForReconciliationFor(instance)
	if err != nil {
		return reconcileResult, err
	}

	// Reconcile all Blocks
	for _, block := range instance.Spec.ContainerBlocks {
		if e := ensureBlock(instance, block, astarte, r.client, r.scheme); e != nil {
			return reconcile.Result{}, e
		}

		delete(existingBlocks, generateBlockName(instance, block, astarte))
	}

	// Any leftovers we should delete?
	for _, b := range existingBlocks {
		if e := r.client.Delete(context.TODO(), &b); e != nil {
			return reconcile.Result{}, e
		}
	}

	// Compute the Status by fetching the Block Deployments, after the cleanup.
	blockList, err := r.getAllBlocksDeploymentsForFlow(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	totalBlocks, readyBlocks, failingBlocks, resList, unrecoverableFailures := r.computeBlocksState(reqLogger, blockList, instance)

	// Set status defaults
	instance.Status.TotalBlocks = totalBlocks
	instance.Status.ReadyBlocks = readyBlocks
	instance.Status.FailingBlocks = failingBlocks
	instance.Status.Resources = resList
	instance.Status.UnrecoverableFailures = unrecoverableFailures

	switch {
	case totalBlocks == 0:
		instance.Status.State = apiv1alpha1.FlowStateUnknown
	case failingBlocks > 0:
		instance.Status.State = apiv1alpha1.FlowStateUnhealthy
	case totalBlocks != readyBlocks:
		instance.Status.State = apiv1alpha1.FlowStateUnstable
	case totalBlocks == readyBlocks:
		instance.Status.State = apiv1alpha1.FlowStateFlowing
	}

	// Update the Status and finish the reconciliation
	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		reqLogger.Error(err, "Failed to update Flow status.")
		return reconcile.Result{}, err
	}

	reqLogger.Info("Flow Reconciled successfully")
	return reconcile.Result{}, nil
}

func (r *ReconcileFlow) getResourcesForReconciliationFor(instance *apiv1alpha1.Flow) (*apiv1alpha1.Astarte, map[string]appsv1.Deployment, reconcile.Result, error) {
	// Get the Astarte instance for the Flow
	astarte := &apiv1alpha1.Astarte{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.Astarte.Name, Namespace: instance.Namespace}, astarte); err != nil {
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

func (r *ReconcileFlow) getAllBlocksForFlow(instance *apiv1alpha1.Flow) (map[string]appsv1.Deployment, error) {
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

func (r *ReconcileFlow) getAllBlocksDeploymentsForFlow(instance *apiv1alpha1.Flow) (appsv1.DeploymentList, error) {
	blockLabels := map[string]string{
		"component":      "astarte-flow",
		"flow-component": "block",
		"flow-name":      instance.Name,
	}
	blockList := appsv1.DeploymentList{}
	err := r.client.List(context.TODO(), &blockList, client.InNamespace(instance.Namespace), client.MatchingLabels(blockLabels))

	return blockList, err
}

func (r *ReconcileFlow) computeBlocksState(reqLogger logr.Logger, blockList appsv1.DeploymentList, instance *apiv1alpha1.Flow) (int, int, int, v1.ResourceList, []v1.ContainerState) {
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
			if err := r.client.List(context.TODO(), podList, client.InNamespace(instance.Namespace),
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
