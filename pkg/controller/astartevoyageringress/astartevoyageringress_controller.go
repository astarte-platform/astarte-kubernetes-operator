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

package astartevoyageringress

import (
	"context"
	"fmt"
	"time"

	voyager "github.com/astarte-platform/astarte-kubernetes-operator/external/voyager/v1beta1"
	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_astartevoyageringress")

// Add creates a new AstarteVoyagerIngress Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAstarteVoyagerIngress{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("astartevoyageringress-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AstarteVoyagerIngress
	if err = c.Watch(&source.Kind{Type: &apiv1alpha1.AstarteVoyagerIngress{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// Additional checks: is the Voyager CRD installed?
	// The Manager Client is not yet available here, we have to resort to a custom Client for now.
	scheme := runtime.NewScheme()
	// Setup Scheme for API Extensions v1beta1
	if e := apiextensionsv1beta1.AddToScheme(scheme); e != nil {
		return e
	}
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}
	client, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return err
	}

	realReconciler := r.(*ReconcileAstarteVoyagerIngress)
	voyagerCRD := &apiextensionsv1beta1.CustomResourceDefinition{}
	if err = client.Get(context.TODO(), types.NamespacedName{Name: "ingresses.voyager.appscode.com"}, voyagerCRD); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Voyager is apparently not installed in this cluster. AstarteVoyagerIngress won't be available")
			realReconciler.shouldReconcile = false
		} else {
			return err
		}
	} else {
		log.Info("Voyager found in the Cluster. Enabling AstarteVoyagerIngress")
		realReconciler.shouldReconcile = true
		// Watch for changes to secondary resource Ingress and requeue the owner AstarteVoyagerIngress
		if err = c.Watch(&source.Kind{Type: &voyager.Ingress{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &apiv1alpha1.AstarteVoyagerIngress{},
		}); err != nil {
			c = nil
			return nil
		}

		// Watch for changes to secondary resource Ingress and requeue the owner AstarteVoyagerIngress
		if err = c.Watch(&source.Kind{Type: &voyager.Certificate{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &apiv1alpha1.AstarteVoyagerIngress{},
		}); err != nil {
			c = nil
			return nil
		}
	}

	return nil
}

// blank assignment to verify that ReconcileAstarteVoyagerIngress implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileAstarteVoyagerIngress{}

// ReconcileAstarteVoyagerIngress reconciles a AstarteVoyagerIngress object
type ReconcileAstarteVoyagerIngress struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client          client.Client
	scheme          *runtime.Scheme
	shouldReconcile bool
}

// Reconcile reads that state of the cluster for a AstarteVoyagerIngress object and makes changes based on the state read
// and what is in the AstarteVoyagerIngress.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAstarteVoyagerIngress) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	if !r.shouldReconcile {
		reqLogger.Info("Not handling reconcile requests for AstarteVoyagerIngress, as Voyager isn't installed")
		return reconcile.Result{}, nil
	}
	reqLogger.Info("Reconciling AstarteVoyagerIngress")

	// Fetch the AstarteVoyagerIngress instance
	instance := &apiv1alpha1.AstarteVoyagerIngress{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Get the Astarte instance
	astarte := &apiv1alpha1.Astarte{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.Astarte, Namespace: instance.Namespace}, astarte); err != nil {
		if errors.IsNotFound(err) {
			d, _ := time.ParseDuration("30s")
			return reconcile.Result{Requeue: true, RequeueAfter: d},
				fmt.Errorf("The Astarte Instance %s associated to this Voyager Ingress object cannot be found", instance.Spec.Astarte)
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Start by reconciling the Certificate (if needed)
	if err := ensureCertificate(instance, astarte, r.client, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Reconcile the API Ingress
	if err := ensureAPIIngress(instance, astarte, r.client, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Reconcile the Broker Ingress
	if err := ensureBrokerIngress(instance, astarte, r.client, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Done
	return reconcile.Result{}, nil
}
