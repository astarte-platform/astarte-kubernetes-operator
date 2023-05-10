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
	"fmt"
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha2"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/migrate"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
	recon "github.com/astarte-platform/astarte-kubernetes-operator/lib/reconcile"
	"github.com/astarte-platform/astarte-kubernetes-operator/version"
)

// ReconcileHelper contains all needed objects to carry over reconciliation of an Astarte-like resource
type ReconcileHelper struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Client   client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// CheckAndPerformUpgrade carries over an upgrade, if needed, of an Astarte resource
func (r *ReconcileHelper) CheckAndPerformUpgrade(reqLogger logr.Logger, instance *apiv1alpha2.Astarte, newAstarteSemVersion *semver.Version) (ctrl.Result, error) {
	// TODO: This should go in the Admission Webhook too, going forward, to prevent deadlocks.
	// Given we're at a high chance of deadlocking here, we want to compute the status again and don't trust what
	// was reported in a previous reconciliation exclusively. On the other hand, in some scenarios (e.g.: failed upgrade
	// due to a temporary issue) we don't want to trust the computed health exclusively if the upgrade started at a time
	// when the cluster was healthy. As such, proceed if one among the computed health and the reported health are green.
	computedClusterHealth := r.ComputeClusterHealth(reqLogger, instance)
	if computedClusterHealth != apiv1alpha2.AstarteClusterHealthGreen && instance.Status.Health != apiv1alpha2.AstarteClusterHealthGreen {
		reqLogger.Error(fmt.Errorf("Astarte Upgrade requested, but the cluster isn't reporting stable Health. Refusing to upgrade"),
			"Cluster health is unstable, refusing to upgrade. Please revert to the previous version and wait for the cluster to settle.",
			"Reported Health", instance.Status.Health, "Computed Health", computedClusterHealth)
		r.Recorder.Event(instance, "Warning", apiv1alpha2.AstarteResourceEventCriticalError.String(),
			fmt.Sprintf("Cluster health is %s, refusing to upgrade. Please revert to the previous version and wait for the cluster to settle", computedClusterHealth))
		return ctrl.Result{Requeue: false}, fmt.Errorf("Astarte Upgrade requested, but the cluster isn't reporting stable Health. Refusing to upgrade")
	}
	// We need to check for upgrades.
	versionString := instance.Status.AstarteVersion
	// Are we on a release snapshot?
	if strings.Contains(instance.Status.AstarteVersion, "-snapshot") {
		// We're running on a release snapshot. Assume it's .0
		versionString = strings.ReplaceAll(versionString, "-snapshot", ".0")
		reqLogger.Info("You are running a Release snapshot. This is generally not a good idea in production. Assuming a Release version", "Version", versionString)
		r.Recorder.Eventf(instance, "Normal", apiv1alpha2.AstarteResourceEventUpgrade.String(),
			"Requested an upgrade from a Release snapshot. Assuming the base Release version is %v", versionString)
	}

	// All good!
	return ctrl.Result{}, nil
}

// ComputeClusterHealth computes, given an Astarte instance, the Health of the cluster
func (r *ReconcileHelper) ComputeClusterHealth(reqLogger logr.Logger, instance *apiv1alpha2.Astarte) apiv1alpha2.AstarteClusterHealth {
	// Compute overall Readiness for Astarte deployments
	astarteDeployments := &appsv1.DeploymentList{}
	nonReadyDeployments := 0
	if err := r.Client.List(context.TODO(), astarteDeployments, client.InNamespace(instance.Namespace),
		client.MatchingLabels{"component": "astarte"}); err == nil {
		for _, deployment := range astarteDeployments.Items {
			if deployment.Status.ReadyReplicas == 0 && pointy.Int32Value(deployment.Spec.Replicas, 0) > 0 {
				nonReadyDeployments++
			}
		}
	} else {
		reqLogger.Info("Could not list Astarte deployments to compute health.")
		// Set it high enough to turn red
		nonReadyDeployments = 5
	}

	// Now compute readiness for the other components we want to check: VerneMQ, RabbitMQ and CFSSL
	astarteStatefulSet := &appsv1.StatefulSet{}
	if pointy.BoolValue(instance.Spec.VerneMQ.Deploy, true) {
		if err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name + "-vernemq"},
			astarteStatefulSet); err == nil {
			if astarteStatefulSet.Status.ReadyReplicas == 0 {
				nonReadyDeployments++
			}
		} else {
			// Just increase the count - it might be a temporary error as the StatefulSet is being created.
			reqLogger.V(1).Info("Could not Get Astarte VerneMQ StatefulSet to compute health.")
			nonReadyDeployments++
		}
	}
	if !r.computeCFSSLHealth(reqLogger, instance) {
		nonReadyDeployments++
	}
	if !r.computeRabbitMQHealth(reqLogger, instance) {
		nonReadyDeployments++
	}

	if nonReadyDeployments == 0 {
		return apiv1alpha2.AstarteClusterHealthGreen
	} else if nonReadyDeployments == 1 {
		return apiv1alpha2.AstarteClusterHealthYellow
	}
	return apiv1alpha2.AstarteClusterHealthRed
}

func (r *ReconcileHelper) computeCFSSLHealth(reqLogger logr.Logger, instance *apiv1alpha2.Astarte) bool {
	if !pointy.BoolValue(instance.Spec.CFSSL.Deploy, true) {
		return true
	}

	// Statefulset or Deployment?
	switch version.CheckConstraintAgainstAstarteVersion("< 1.0.0", instance.Spec.Version) {
	case nil:
		cfsslStatefulSet := &appsv1.StatefulSet{}
		if err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name + "-cfssl"},
			cfsslStatefulSet); err == nil {
			if cfsslStatefulSet.Status.ReadyReplicas == 0 {
				return false
			}
		} else {
			// Just increase the count - it might be a temporary error as the StatefulSet is being created.
			reqLogger.V(1).Info("Could not Get Astarte CFSSL StatefulSet to compute health.")
			return false
		}
	case version.ErrConstraintNotSatisfied:
		cfsslDeployment := &appsv1.Deployment{}
		if err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name + "-cfssl"},
			cfsslDeployment); err == nil {
			if cfsslDeployment.Status.ReadyReplicas == 0 {
				return false
			}
		} else {
			// Just increase the count - it might be a temporary error as the Deployment is being created.
			reqLogger.V(1).Info("Could not Get Astarte CFSSL Deployment to compute health.")
			return false
		}
	default:
		return false
	}

	return true
}

func (r *ReconcileHelper) computeRabbitMQHealth(reqLogger logr.Logger, instance *apiv1alpha2.Astarte) bool {
	if !pointy.BoolValue(instance.Spec.RabbitMQ.Deploy, true) {
		return true
	}

	rabbitMQStatefulSet := &appsv1.StatefulSet{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name + "-rabbitmq"}, rabbitMQStatefulSet); err == nil {
		if rabbitMQStatefulSet.Status.ReadyReplicas == 0 {
			return false
		}
	} else {
		// Just increase the count - it might be a temporary error as the StatefulSet is being created.
		reqLogger.V(1).Info("Could not Get Astarte RabbitMQ StatefulSet to compute health.")
		return false
	}

	return true
}

// EnsureStatusCoherency ensures status coherency
func (r *ReconcileHelper) EnsureStatusCoherency(reqLogger logr.Logger, instance *apiv1alpha2.Astarte, request ctrl.Request) (ctrl.Result, error) {
	if instance.Status.AstarteVersion != "" {
		// It's simply ok.
		return ctrl.Result{}, nil
	}

	// Ok, in this case there's two potential situations: we're on our first reconcile, or the status is
	// messed up. Let's see if we can find the Housekeeping Deployment.
	hkDeployment := &appsv1.Deployment{}
	if err := r.Client.Get(context.TODO(),
		types.NamespacedName{Name: instance.Name + "-housekeeping", Namespace: instance.Namespace}, hkDeployment); err == nil {
		// In this case, we are in a weird state (e.g.: migrating from the old operator). Let's try and fix this.
		// First of all, try and migrate it.
		r.Recorder.Event(instance, "Warning", apiv1alpha2.AstarteResourceEventMigration.String(),
			"Found an invalid status. The resource will be migrated to latest format")
		reqLogger.Info("Found an invalid status. Attempting to migrate the resource.")
		if e := migrate.ToNewCR(instance, r.Client, r.Scheme); e != nil {
			return ctrl.Result{}, e
		}

		// Second of all, we want to ensure we have a clean start without losing anything. To do so, we need to bring it to a state
		// where it can always be reconciled.
		// Let's just ensure the Status struct is meaningful: reconstruct it from what we know/can access.
		instance.Status.ReconciliationPhase = apiv1alpha2.ReconciliationPhaseReconciling
		instance.Status.OperatorVersion = version.Version
		// red before anything else happens
		instance.Status.Health = apiv1alpha2.AstarteClusterHealthRed
		instance.Status.BaseAPIURL = instance.Spec.API.Host
		instance.Status.BrokerURL = instance.Spec.VerneMQ.Host

		reqLogger.Info("Reconciling Astarte Version from Housekeeping's image tag")
		hkImage := hkDeployment.Spec.Template.Spec.Containers[0].Image
		hkImageTokens := strings.Split(hkImage, ":")
		if len(hkImageTokens) != 2 {
			// Reconcile every minute if we're here
			r.Recorder.Eventf(instance, "Warning", apiv1alpha2.AstarteResourceEventCriticalError.String(),
				"Could not parse Astarte version from Housekeeping Image tag %s. Please fix your resource definition", hkImage)
			return ctrl.Result{RequeueAfter: time.Minute},
				fmt.Errorf("Could not parse Astarte version from Housekeeping Image tag %s. Refusing to proceed", hkImage)
		}

		// Update the status
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			instance = &apiv1alpha2.Astarte{}
			if err = r.Client.Get(context.TODO(), request.NamespacedName, instance); err != nil {
				// Error reading the object - requeue the request.
				return err
			}

			instance.Status.AstarteVersion = hkImageTokens[1]

			if err = r.Client.Status().Update(context.TODO(), instance); err != nil {
				r.Recorder.Event(instance, "Warning", apiv1alpha2.AstarteResourceEventReconciliationFailed.String(),
					"Failed to update Astarte status - will retry reconciliation")
				reqLogger.Error(err, "Failed to update Astarte status.")
				return err
			}
			return nil
		}); err != nil {
			return ctrl.Result{}, err
		}

		r.Recorder.Eventf(instance, "Normal", apiv1alpha2.AstarteResourceEventMigration.String(),
			"Resource version reconciled to %v from Housekeeping", hkImageTokens[1])

		// If we got here, we want to Get the instance again. Given the modifications we made, we want to ensure we're in sync.
		instance = &apiv1alpha2.Astarte{}
		if e := r.Client.Get(context.TODO(), request.NamespacedName, instance); e != nil {
			return ctrl.Result{}, e
		}
	} else if !errors.IsNotFound(err) {
		// There was some issue in reading the Object - requeue
		return ctrl.Result{}, err
	}

	r.Recorder.Event(instance, "Normal", apiv1alpha2.AstarteResourceEventStatus.String(),
		"Running first resource reconciliation")
	reqLogger.V(1).Info("Apparently running first reconciliation.")

	return ctrl.Result{}, nil
}

// ComputeAstarteStatusResource computes an AstarteStatus resource
func (r *ReconcileHelper) ComputeAstarteStatusResource(reqLogger logr.Logger, instance *apiv1alpha2.Astarte) apiv1alpha2.AstarteStatus {
	oldAstarteHealth := instance.Status.Health
	newAstarteStatus := instance.Status
	newAstarteStatus.Health = r.ComputeClusterHealth(reqLogger, instance)

	// Cast an event in case the health changed
	if oldAstarteHealth != newAstarteStatus.Health && oldAstarteHealth != "" {
		eventtype := "Normal"
		// Notify as a warning if the health degraded compared to the previous reconciliation.
		if oldAstarteHealth == apiv1alpha2.AstarteClusterHealthGreen {
			eventtype = "Warning"
		}
		r.Recorder.Eventf(instance, eventtype, apiv1alpha2.AstarteResourceEventStatus.String(),
			"Astarte Cluster status changed from %v to %v", oldAstarteHealth, newAstarteStatus.Health)
	}

	// Update status
	newAstarteStatus.AstarteVersion = instance.Spec.Version
	newAstarteStatus.OperatorVersion = version.Version
	newAstarteStatus.ReconciliationPhase = apiv1alpha2.ReconciliationPhaseReconciled
	newAstarteStatus.BaseAPIURL = "https://" + instance.Spec.API.Host
	newAstarteStatus.BrokerURL = misc.GetVerneMQBrokerURL(instance)

	if instance.Spec.ManualMaintenanceMode {
		newAstarteStatus.ReconciliationPhase = apiv1alpha2.ReconciliationPhaseManualMaintenanceMode
	}

	// Return the Astarte status
	return newAstarteStatus
}

// ReconcileAstarteResources reconciles all third-party dependencies, when needed
func (r *ReconcileHelper) ReconcileAstarteResources(instance *apiv1alpha2.Astarte) error {
	// Start by ensuring the housekeeping key
	if err := recon.EnsureHousekeepingKey(instance, r.Client, r.Scheme); err != nil {
		return err
	}
	// Then, make sure we have an up to date Erlang Configuration for our Pods
	if err := recon.EnsureGenericErlangConfiguration(instance, r.Client, r.Scheme); err != nil {
		return err
	}

	// Give priority to PriorityClasses
	if err := recon.EnsureAstartePriorityClasses(instance, r.Client, r.Scheme); err != nil {
		return err
	}

	// Dependencies Dance!
	// RabbitMQ, first and foremost
	if err := recon.EnsureRabbitMQ(instance, r.Client, r.Scheme); err != nil {
		return err
	}

	// Cassandra
	if err := recon.EnsureCassandra(instance, r.Client, r.Scheme); err != nil {
		return err
	}

	// CFSSL
	if err := recon.EnsureCFSSL(instance, r.Client, r.Scheme); err != nil {
		return err
	}

	// CFSSL CA Secret - if we're < 1.0.0
	if version.CheckConstraintAgainstAstarteVersion("< 1.0.0", instance.Spec.Version) == nil {
		if err := recon.EnsureCFSSLCASecret(instance, r.Client, r.Scheme); err != nil {
			return err
		}
	}

	// OK! Now it's time to reconcile all of Astarte Services
	if err := r.EnsureAstarteMicroservices(instance); err != nil {
		return err
	}

	// Last but not least, VerneMQ
	if err := recon.EnsureVerneMQ(instance, r.Client, r.Scheme); err != nil {
		return err
	}

	// And Dashboard to close it down.
	if err := recon.EnsureAstarteDashboard(instance, instance.Spec.Components.Dashboard, r.Client, r.Scheme); err != nil {
		return err
	}

	// All good!
	return nil
}

// EnsureAstarteMicroservices reconciles all Astarte microservices
func (r *ReconcileHelper) EnsureAstarteMicroservices(instance *apiv1alpha2.Astarte) error {
	// OK! Now it's time to reconcile all of Astarte Services, in a specific order.
	// Housekeeping first - it creates/migrates the Database
	if err := r.ensureAstarteGenericComponent(instance, instance.Spec.Components.Housekeeping, apiv1alpha2.Housekeeping, apiv1alpha2.HousekeepingAPI); err != nil {
		return err
	}

	// Then, Realm Management
	if err := r.ensureAstarteGenericComponent(instance, instance.Spec.Components.RealmManagement, apiv1alpha2.RealmManagement, apiv1alpha2.RealmManagementAPI); err != nil {
		return err
	}

	// Then, Pairing
	if err := r.ensureAstarteGenericComponent(instance, instance.Spec.Components.Pairing, apiv1alpha2.Pairing, apiv1alpha2.PairingAPI); err != nil {
		return err
	}

	// Then, Flow
	if err := recon.EnsureAstarteGenericAPI(instance, instance.Spec.Components.Flow, apiv1alpha2.FlowComponent, r.Client, r.Scheme); err != nil {
		return err
	}

	// Trigger Engine right before DUP
	if err := recon.EnsureAstarteGenericBackend(instance, instance.Spec.Components.TriggerEngine.AstarteGenericClusteredResource, apiv1alpha2.TriggerEngine, r.Client, r.Scheme); err != nil {
		return err
	}

	// Now it's Data Updater plant turn
	if err := recon.EnsureAstarteDataUpdaterPlant(instance, instance.Spec.Components.DataUpdaterPlant, r.Client, r.Scheme); err != nil {
		return err
	}

	// Now it's AppEngine API turn
	if err := recon.EnsureAstarteGenericAPI(instance, instance.Spec.Components.AppengineAPI.AstarteGenericAPISpec, apiv1alpha2.AppEngineAPI, r.Client, r.Scheme); err != nil {
		return err
	}

	// All good!
	return nil
}

func (r *ReconcileHelper) ensureAstarteGenericComponent(instance *apiv1alpha2.Astarte, genericComponentSpec apiv1alpha2.AstarteGenericComponentSpec,
	backendComponent, apiComponent apiv1alpha2.AstarteComponent) error {
	if err := recon.EnsureAstarteGenericBackend(instance, genericComponentSpec.Backend, backendComponent, r.Client, r.Scheme); err != nil {
		return err
	}
	return recon.EnsureAstarteGenericAPI(instance, genericComponentSpec.API, apiComponent, r.Client, r.Scheme)
}
