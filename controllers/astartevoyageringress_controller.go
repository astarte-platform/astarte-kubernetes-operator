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
	"fmt"
	"time"

	voyagercrd "github.com/astarte-platform/astarte-kubernetes-operator/external/voyager/v1beta1"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/voyager"
	"github.com/go-logr/logr"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/v1alpha1"
)

// AstarteVoyagerIngressReconciler reconciles a AstarteVoyagerIngress object
type AstarteVoyagerIngressReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	shouldReconcile bool
}

// +kubebuilder:rbac:groups=api.astarte-platform.org,resources=astartevoyageringresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.astarte-platform.org,resources=astartevoyageringresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=voyager.appscode.com,resources=*,verbs=get;list;watch;create;update;patch;delete

func (r *AstarteVoyagerIngressReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("astartevoyageringress", req.NamespacedName)

	if !r.shouldReconcile {
		reqLogger.Info("Not handling reconcile requests for AstarteVoyagerIngress, as Voyager isn't installed")
		return ctrl.Result{}, nil
	}
	reqLogger.Info("Reconciling AstarteVoyagerIngress")

	// Fetch the AstarteVoyagerIngress instance
	instance := &apiv1alpha1.AstarteVoyagerIngress{}
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
	astarte := &apiv1alpha1.Astarte{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.Astarte, Namespace: instance.Namespace}, astarte); err != nil {
		if errors.IsNotFound(err) {
			d, _ := time.ParseDuration("30s")
			return ctrl.Result{Requeue: true, RequeueAfter: d},
				fmt.Errorf("The Astarte Instance %s associated to this Voyager Ingress object cannot be found", instance.Spec.Astarte)
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Start by reconciling the Certificate (if needed)
	if err := voyager.EnsureCertificate(instance, astarte, r.Client, r.Scheme, reqLogger); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile the API Ingress
	if err := voyager.EnsureAPIIngress(instance, astarte, r.Client, r.Scheme, reqLogger); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile the Broker Ingress
	if err := voyager.EnsureBrokerIngress(instance, astarte, r.Client, r.Scheme, reqLogger); err != nil {
		return ctrl.Result{}, err
	}

	// Done
	return ctrl.Result{}, nil
}

func (r *AstarteVoyagerIngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Additional checks: is the Voyager CRD installed?
	// The Manager Client is not yet available here, we have to resort to a custom Client for now.
	scheme := runtime.NewScheme()
	// Setup Scheme for API Extensions v1beta1
	if err := apiextensionsv1beta1.AddToScheme(scheme); err != nil {
		return err
	}

	cfg := mgr.GetConfig()
	client, e := client.New(cfg, client.Options{Scheme: scheme})
	if e != nil {
		return e
	}

	voyagerCRD := &apiextensionsv1beta1.CustomResourceDefinition{}
	if err := client.Get(context.TODO(), types.NamespacedName{Name: "ingresses.voyager.appscode.com"}, voyagerCRD); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("Voyager is apparently not installed in this cluster. AstarteVoyagerIngress won't be available")
			r.shouldReconcile = false
		} else {
			return err
		}

		return ctrl.NewControllerManagedBy(mgr).
			For(&apiv1alpha1.AstarteVoyagerIngress{}).
			Complete(r)
	}

	r.Log.Info("Voyager found in the Cluster. Enabling AstarteVoyagerIngress")
	r.shouldReconcile = true

	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	}

	// Watch for changes to secondary resource Ingress and requeue the owner AstarteVoyagerIngress
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.AstarteVoyagerIngress{}, builder.WithPredicates(pred)).
		Owns(&voyagercrd.Certificate{}).
		Owns(&voyagercrd.Ingress{}).
		Complete(r)
}
