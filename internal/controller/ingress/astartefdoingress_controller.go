/*
This file is part of Astarte.

Copyright 2020-26 SECO Mind Srl.

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

package ingress

import (
	"context"
	"fmt"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	ingressv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v2alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/controllerutils"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/fdoingress"
	"github.com/go-logr/logr"
)

// AstarteFDOIngressReconciler reconciles a AstarteFDOIngress object
type AstarteFDOIngressReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ingress.astarte-platform.org,resources=astartefdoingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ingress.astarte-platform.org,resources=astartefdoingresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ingress.astarte-platform.org,resources=astartefdoingresses/finalizers,verbs=update
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services;services/finalizers;configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *AstarteFDOIngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)
	reqLogger := r.Log.WithValues("astartefdoingress", req.NamespacedName)
	reqLogger.Info("Reconciling AstarteFDOIngress")

	// Fetch the AstarteFDOIngress instance
	instance := &ingressv2alpha1.AstarteFDOIngress{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Get the Astarte instance
	astarte := &apiv2alpha1.Astarte{}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.Astarte, Namespace: instance.Namespace}, astarte); err != nil {
		if errors.IsNotFound(err) {
			d, _ := time.ParseDuration("30s")
			return ctrl.Result{Requeue: true, RequeueAfter: d},
				fmt.Errorf("the Astarte Instance %s associated to this Ingress object cannot be found", instance.Spec.Astarte)
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	reconciler := controllerutils.ReconcileHelper{
		Client: r.Client,
		Scheme: r.Scheme,
	}

	// Check if Astarte is in manual maintenance mode
	if astarte.Spec.ManualMaintenanceMode {
		// If that is so, compute the status and quit.
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			instance = &ingressv2alpha1.AstarteFDOIngress{}
			if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
				return err
			}

			instance.Status = reconciler.ComputeFDOIngressStatusResource(reqLogger, instance)

			if err := r.Client.Status().Update(ctx, instance); err != nil {
				reqLogger.Error(err, "Failed to update AstarteFDOIngress status.")
				return err
			}
			return nil
		}); err != nil {
			return ctrl.Result{}, err
		}

		// Notify and return
		reqLogger.Info("AstarteFDOIngress Reconciliation skipped due to Manual Maintenance Mode set true in Astarte CR. Hope you know what you're doing!")
		return ctrl.Result{}, nil
	}

	// Reconcile the FDO Ingress
	if err := fdoingress.EnsureFDOIngress(instance, astarte, r.Client, r.Scheme, reqLogger); err != nil {
		return ctrl.Result{}, err
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance := &ingressv2alpha1.AstarteFDOIngress{}
		if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
			return err
		}

		instance.Status = reconciler.ComputeFDOIngressStatusResource(reqLogger, instance)

		if err := r.Client.Status().Update(ctx, instance); err != nil {
			reqLogger.Error(err, "Failed to update AstarteFDOIngress status.")
			return err
		}
		return nil
	}); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AstarteFDOIngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
	}

	// AstarteFDOIngress depends on information in the referenced Astarte CR (e.g., API URL).
	// With this watch, changing the Astarte object triggers reconciliation of the dependent AstarteFDOIngress
	astarteToADIReconcileRequestFunc := func(_ context.Context, obj client.Object) []reconcile.Request {
		astarteName := obj.GetName()
		req := []reconcile.Request{}
		list := &ingressv2alpha1.AstarteFDOIngressList{}
		_ = r.List(context.Background(), list, client.InNamespace(obj.GetNamespace()))

		if len(list.Items) == 0 {
			return req
		}

		for _, item := range list.Items {
			if item.Spec.Astarte == astarteName {
				req = append(req, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      item.GetName(),
						Namespace: item.GetNamespace(),
					},
				})
			}
		}
		return req
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&ingressv2alpha1.AstarteFDOIngress{}, builder.WithPredicates(pred)).
		Named("ingress-astartefdoingress").
		Owns(&networkingv1.Ingress{}).
		Watches(
			&apiv2alpha1.Astarte{},
			handler.EnqueueRequestsFromMapFunc(astarteToADIReconcileRequestFunc),
		).
		Complete(r)
}
