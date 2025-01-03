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

package ingress

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
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
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
	ingressv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/controllerutils"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/defaultingress"
)

// AstarteDefaultIngressReconciler reconciles a AstarteDefaultIngress object
type AstarteDefaultIngressReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ingress.astarte-platform.org,resources=astartedefaultingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ingress.astarte-platform.org,resources=astartedefaultingresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services;services/finalizers;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ingress.astarte-platform.org,resources=astartedefaultingresses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *AstarteDefaultIngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("astartedefaultingress", req.NamespacedName)
	reqLogger.Info("Reconciling AstarteDefaultIngress")

	// Fetch the AstarteDefaultIngress instance
	instance := &ingressv1alpha1.AstarteDefaultIngress{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
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
	astarte := &apiv1alpha2.Astarte{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.Astarte, Namespace: instance.Namespace}, astarte); err != nil {
		if errors.IsNotFound(err) {
			d, _ := time.ParseDuration("30s")
			return ctrl.Result{Requeue: true, RequeueAfter: d},
				fmt.Errorf("the Astarte Instance %s associated to this Ingress object cannot be found", instance.Spec.Astarte)
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Reconcile the API Ingress
	if err := defaultingress.EnsureAPIIngress(instance, astarte, r.Client, r.Scheme, reqLogger); err != nil {
		return ctrl.Result{}, err
	}
	// Reconcile the Broker Ingress
	if err := defaultingress.EnsureBrokerIngress(instance, astarte, r.Client, r.Scheme, reqLogger); err != nil {
		return ctrl.Result{}, err
	}
	// And eventually reconcile the Metrics Ingress
	if err := defaultingress.EnsureMetricsIngress(instance, astarte, r.Client, r.Scheme, reqLogger); err != nil {
		return ctrl.Result{}, err
	}

	reconciler := controllerutils.ReconcileHelper{
		Client: r.Client,
		Scheme: r.Scheme,
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance := &ingressv1alpha1.AstarteDefaultIngress{}
		if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
			return err
		}

		instance.Status = reconciler.ComputeADIStatusResource(reqLogger, instance)

		if err := r.Client.Status().Update(ctx, instance); err != nil {
			reqLogger.Error(err, "Failed to update AstarteDefaultIngress status.")
			return err
		}
		return nil
	}); err != nil {
		return ctrl.Result{}, err
	}

	// Done
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AstarteDefaultIngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
	}

	astarteToADIReconcileRequestFunc := func(_ context.Context, obj client.Object) []reconcile.Request {
		astarteName := obj.GetName()
		ret := []reconcile.Request{}
		adiList := &ingressv1alpha1.AstarteDefaultIngressList{}
		_ = r.List(context.Background(), adiList, client.InNamespace(obj.GetNamespace()))

		if len(adiList.Items) == 0 {
			return ret
		}

		for _, item := range adiList.Items {
			if item.Spec.Astarte == astarteName {
				ret = append(ret, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      item.GetName(),
						Namespace: item.GetNamespace(),
					},
				})
			}
		}
		return ret
	}

	// Watch for changes to secondary resource Ingress and requeue the owner AstarteDefaultIngress
	return ctrl.NewControllerManagedBy(mgr).
		For(&ingressv1alpha1.AstarteDefaultIngress{}, builder.WithPredicates(pred)).
		Owns(&networkingv1.Ingress{}).
		Watches(
			&apiv1alpha2.Astarte{},
			handler.EnqueueRequestsFromMapFunc(astarteToADIReconcileRequestFunc),
		).
		Complete(r)
}
