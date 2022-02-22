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

package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/reconcile"
)

// ForceRunModeEnv indicates if the operator should be forced to run in either local
// or cluster mode (currently only used for local mode)
var ForceRunModeEnv = "OSDK_FORCE_RUN_MODE"

type RunModeType string

const (
	LocalRunMode   RunModeType = "local"
	ClusterRunMode RunModeType = "cluster"
)

func isRunModeLocal() bool {
	return os.Getenv(ForceRunModeEnv) == string(LocalRunMode)
}

// ErrNoNamespace indicates that a namespace could not be found for the current
// environment
var ErrNoNamespace = fmt.Errorf("namespace not found for current environment")

// ErrRunLocal indicates that the operator is set to run in local mode (this error
// is returned by functions that only work on operators running in cluster mode)
var ErrRunLocal = fmt.Errorf("operator run mode forced to local")

// GetOperatorNamespace returns the namespace the operator should be running in.
func GetOperatorNamespace() (string, error) {
	if isRunModeLocal() {
		return "", ErrRunLocal
	}
	nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoNamespace
		}
		return "", err
	}
	ns := strings.TrimSpace(string(nsBytes))
	log.V(1).Info("Found namespace", "Namespace", ns)
	return ns, nil
}

func shutdownVerneMQ(cr *apiv1alpha1.Astarte, c client.Client, recorder record.EventRecorder) error {
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	// First, bring down VerneMQ by putting its replicas to 0, and wait until it is settled.
	verneMQStatefulSetName := cr.Name + "-vernemq"
	verneMQStatefulSet := &appsv1.StatefulSet{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: verneMQStatefulSetName, Namespace: cr.Namespace}, verneMQStatefulSet); err != nil {
		return fmt.Errorf("Could not retrieve VerneMQ statefulset: %v", err)
	}
	verneMQStatefulSet.Spec.Replicas = pointy.Int32(0)
	reqLogger.Info("Bringing down the broker to prevent data loss and mismatches. Devices won't be able to connect until the next reconciliation.")
	if err := c.Update(context.TODO(), verneMQStatefulSet); err != nil {
		return fmt.Errorf("Could not downscale VerneMQ statefulset: %v", err)
	}
	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
		"Bringing down the broker to prevent data loss and mismatches. Devices won't be able to connect until the next reconciliation")

	reqLogger.Info("Waiting for the broker to go down...")
	// Now wait
	if err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		statefulSet := &appsv1.StatefulSet{}
		if err = c.Get(context.TODO(), types.NamespacedName{Name: verneMQStatefulSetName, Namespace: cr.Namespace}, statefulSet); err != nil {
			return false, err
		}

		if statefulSet.Status.Replicas > 0 {
			return false, nil
		}

		return true, nil
	}); err != nil {
		recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventUpgradeError.String(),
			"Could not bring down the Broker. Upgrade will be retried")
		return fmt.Errorf("Failed in waiting for VerneMQ statefulset to shutdown: %v", err)
	}

	return nil
}

func drainRabbitMQQueues(cr *apiv1alpha1.Astarte, c client.Client, recorder record.EventRecorder) error {
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	// We might also find out whether the queue has been entirely drained, so we don't lose
	// data. If we're deployed externally, we have to initiate a port forward.
	rmqHost, _, rmqUser, rmqPass, err := misc.GetRabbitMQCredentialsFor(cr, c)
	if err != nil {
		reqLogger.Error(err, "Could not fetch RabbitMQ credentials. Skipping RabbitMQ queue checks.")
		return err
	}

	// If we need to port forward, a connection will be opened. Replace the host, in that case. We assume
	// RabbitMQ Management is enabled (it's a requirement, anyway)
	var stopChannel chan struct{}
	if newHost, theStopChannel, err := openRabbitMQPortForward(cr); err == nil && newHost != "" {
		stopChannel = theStopChannel
		rmqHost = newHost
	} else if err != nil {
		return err
	}

	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
		"Draining RabbitMQ Data Queues")

	// Get the queue state
	httpClient := &http.Client{}
	req, _ := http.NewRequest("GET", "http://"+rmqHost+":15672/api/queues", nil)
	req.SetBasicAuth(rmqUser, rmqPass)

	// Wait up to a minute, otherwise restart
	if err := wait.Poll(5*time.Second, time.Minute, func() (done bool, err error) {
		resp, e := httpClient.Do(req)
		if e != nil {
			reqLogger.Error(e, "Could not query RabbitMQ Management, retrying...")
			return false, nil
		}

		defer resp.Body.Close()
		respBody, _ := ioutil.ReadAll(resp.Body)
		respJSON := []map[string]interface{}{}
		if e2 := json.Unmarshal(respBody, &respJSON); e2 != nil {
			recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
				"Unrecoverable error in querying RabbitMQ Management. Upgrade will be retried, but manual intervention is likely required")
			reqLogger.Error(e2, "Unrecoverable error in querying RabbitMQ Management")
			return false, e2
		}
		for _, queueState := range respJSON {
			// Check if it matches one of the data queues from 0.10 onwards
			if ok, e3 := checkRabbitMQQueue(queueState, reqLogger); !ok {
				return false, e3
			}
		}
		return true, nil
	}); err != nil {
		recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
			"Timed out while waiting for queues to drain. Upgrade will be retried, but manual intervention is likely required")
		reqLogger.Error(err, "Failed in waiting for RabbitMQ queues to be drained")
		return err
	}

	reqLogger.Info("RabbitMQ Data Queue(s) drained")
	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(),
		"RabbitMQ Data Queues successfully drained")

	if stopChannel != nil {
		// Close the forwarder
		close(stopChannel)
	}

	return nil
}

func checkRabbitMQQueue(queueState map[string]interface{}, reqLogger logr.Logger) (bool, error) {
	// Check if it matches one of the data queues from 0.10 onwards
	queueName, ok := queueState["name"].(string)
	switch {
	case !ok:
		return false, fmt.Errorf("Malformed JSON reply from RabbitMQ management")
	case queueName == "astarte_data_updater_plant_rpc":
		// Break false positives
		return true, nil
	case queueName == "vmq_all", strings.HasPrefix(queueName, "astarte_data_"):
		// Match, don't do anything
	default:
		// Don't take the queue into account
		return true, nil
	}

	// float64 is how this is decoded by Go
	messagesReady, ok := queueState["messages_ready"].(float64)
	switch {
	case !ok:
		return false, fmt.Errorf("Malformed JSON reply from RabbitMQ management")
	case messagesReady > 0:
		reqLogger.Info("Waiting for RabbitMQ Data Queues to drain.",
			"MessagesLeft", messagesReady, "QueueName", queueName)
		return false, nil
	}

	return true, nil
}

func openRabbitMQPortForward(cr *apiv1alpha1.Astarte) (string, chan struct{}, error) {
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	var fw *portforward.PortForwarder
	var stopChannel chan struct{}

	// Note that we're trying to find out whether the operator is running outside the cluster
	if _, err := GetOperatorNamespace(); err != nil {
		if err == ErrNoNamespace || err == ErrRunLocal {
			reqLogger.Info("Not running in a cluster - trying to forward RabbitMQ port")
			restConfig, e := config.GetConfig()
			if e != nil {
				return "", nil, e
			}

			path := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s-rabbitmq-0/portforward", restConfig.Host, cr.Namespace, cr.Name)
			url, e := url.Parse(path)
			if e != nil {
				return "", nil, e
			}

			transport, upgrader, e := spdy.RoundTripperFor(restConfig)
			if e != nil {
				return "", nil, e
			}
			dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)
			stopChannel = make(chan struct{}, 1)
			readyChannel := make(chan struct{})
			errChannel := make(chan error)

			// Well, Go!
			go func() {
				var ferr error
				if fw, ferr = portforward.New(dialer, []string{"15672:15672"}, stopChannel, readyChannel, nil, nil); ferr != nil {
					errChannel <- ferr
				}
				if ferr = fw.ForwardPorts(); ferr != nil {
					errChannel <- ferr
				}
			}()

			select {
			case <-readyChannel:
				break
			case e := <-errChannel:
				return "", nil, e
			}

			return "localhost", stopChannel, nil
		}
		return "", nil, err
	}

	return "", nil, nil
}

func waitForHousekeepingUpgrade(cr *apiv1alpha1.Astarte, c client.Client, recorder record.EventRecorder) error {
	weirdFailuresCount := 0
	weirdFailuresThreshold := 10

	return wait.Poll(retryInterval, time.Hour, func() (done bool, err error) {
		deployment := &appsv1.Deployment{}
		if err = c.Get(context.TODO(), types.NamespacedName{Name: cr.Name + "-housekeeping-api", Namespace: cr.Namespace}, deployment); err != nil {
			weirdFailuresCount++
			if weirdFailuresCount > weirdFailuresThreshold {
				// Something is off.
				recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
					"Repeated errors in monitoring Database Migration. Manual intervention is likely required")
				return false, fmt.Errorf("Failed in looking up Housekeeping API Deployment. Most likely, manual intervention is required. %v", err)
			}
			// Something is off.
			log.Error(err, "Failed in looking up Housekeeping API Deployment. This might be a temporary problem - will retry")
			return false, nil
		}

		if deployment.Status.ReadyReplicas >= 1 {
			// That's it bros.
			return true, nil
		}

		// Ensure we aren't in the position where Housekeeping itself is crashing.
		housekeepingComponent := apiv1alpha1.Housekeeping
		podList := &v1.PodList{}
		if err = c.List(context.TODO(), podList, client.InNamespace(cr.Namespace),
			client.MatchingLabels{"astarte-component": housekeepingComponent.DashedString()}); err != nil {
			weirdFailuresCount++
			if weirdFailuresCount > weirdFailuresThreshold {
				// Something is off.
				recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
					"Repeated errors in monitoring Database Migration. Manual intervention is likely required")
				return false, fmt.Errorf("Failed in looking up Housekeeping pods. Most likely, manual intervention is required. %v", err)
			}
			// Something is off.
			log.Error(err, "Failed in looking up Housekeeping pods. This might be a temporary problem - will retry")
			return false, nil
		}

		// Inspect the list!
		if len(podList.Items) != 1 {
			weirdFailuresCount++
			if weirdFailuresCount > weirdFailuresThreshold {
				// Something is off.
				recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
					"Repeated errors in monitoring Database Migration. Manual intervention is likely required")
				return false, fmt.Errorf("%v Housekeeping pods found. Most likely, manual intervention is required. %v", len(podList.Items), err)
			}
			// Something is off.
			log.Error(err, fmt.Sprintf("%v Housekeeping pods found. This might be a temporary problem - will retry", len(podList.Items)))
			return false, nil
		}

		if len(podList.Items[0].Status.ContainerStatuses) != 1 {
			weirdFailuresCount++
			if weirdFailuresCount > weirdFailuresThreshold {
				// Something is off.
				recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
					"Repeated errors in monitoring Database Migration. Manual intervention is likely required")
				return false, fmt.Errorf("%v Container Statuses retrieved. Most likely, manual intervention is required. %v", len(podList.Items[0].Status.ContainerStatuses), err)
			}
			// Something is off.
			log.Error(err, fmt.Sprintf("%v Container Statuses retrieved. This might be a temporary problem - will retry", len(podList.Items[0].Status.ContainerStatuses)))
			return false, nil
		}

		if podList.Items[0].Status.ContainerStatuses[0].State.Waiting != nil {
			if podList.Items[0].Status.ContainerStatuses[0].State.Waiting.Reason == "CrashLoopBackoff" {
				recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
					"Database Migration failed. Manual intervention is likely required")
				return true, fmt.Errorf("Housekeeping is crashing repeatedly. There has to be a problem in handling Database migrations. Please take manual action as soon as possible")
			}
		}

		return false, nil
	})
}

func upgradeHousekeeping(version string, drainVerneMQResources bool, cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme,
	recorder record.EventRecorder) (*apiv1alpha1.AstarteGenericClusteredResource, error) {
	housekeepingBackend := cr.Spec.Components.Housekeeping.Backend.DeepCopy()
	housekeepingBackend.Version = version
	housekeepingBackend.Replicas = pointy.Int32(1)
	// Ensure the policy is Replace. We don't want to have old pods hanging around.
	housekeepingBackend.DeploymentStrategy = &appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
	if cr.Spec.VerneMQ.Resources != nil && drainVerneMQResources {
		resourceRequirements := misc.GetResourcesForAstarteComponent(cr, housekeepingBackend.Resources, apiv1alpha1.Housekeeping)
		resourceRequirements.Requests.Cpu().Add(*cr.Spec.VerneMQ.Resources.Requests.Cpu())
		resourceRequirements.Requests.Memory().Add(*cr.Spec.VerneMQ.Resources.Requests.Memory())
		resourceRequirements.Limits.Cpu().Add(*cr.Spec.VerneMQ.Resources.Limits.Cpu())
		resourceRequirements.Limits.Memory().Add(*cr.Spec.VerneMQ.Resources.Limits.Memory())

		// This way, on the next call to GetResourcesForAstarteComponent, these resources will be returned as explicitly stated
		// in the original spec.
		housekeepingBackend.Resources = &resourceRequirements
	}

	// Add a custom, more permissive probe to the Backend
	if err := reconcile.EnsureAstarteGenericBackendWithCustomProbe(cr, *housekeepingBackend, apiv1alpha1.Housekeeping,
		c, scheme, getSpecialHousekeepingMigrationProbe("/health")); err != nil {
		recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventUpgradeError.String(),
			"Could not initiate Database Migration. Upgrade will be retried")
		return nil, err
	}
	housekeepingAPI := cr.Spec.Components.Housekeeping.API.DeepCopy()
	housekeepingAPI.Replicas = pointy.Int32(1)
	housekeepingAPI.Version = version
	// Ensure the policy is Replace. We don't want to have old pods hanging around.
	housekeepingAPI.DeploymentStrategy = &appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
	if err := reconcile.EnsureAstarteGenericAPIWithCustomProbe(cr, *housekeepingAPI, apiv1alpha1.HousekeepingAPI, c,
		scheme, getSpecialHousekeepingMigrationProbe("/health")); err != nil {
		recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventUpgradeError.String(),
			"Could not initiate Database Migration. Upgrade will be retried")
		return nil, err
	}

	return housekeepingBackend, nil
}

func getSpecialHousekeepingMigrationProbe(path string) *v1.Probe {
	// This is a special migration probe that handles longer timeouts due to migrations.
	// Migrations can take an insane amount of time, as such we should take this into account.
	return &v1.Probe{
		ProbeHandler: v1.ProbeHandler{
			HTTPGet: &v1.HTTPGetAction{
				Path: path,
				Port: intstr.FromString("http"),
			},
		},
		// Start checking after 30 seconds.
		InitialDelaySeconds: 30,
		TimeoutSeconds:      5,
		// Check every 30 seconds
		PeriodSeconds: 30,
		// Allow up to an hour before failing. That's 120 failures.
		FailureThreshold: 120,
	}
}

func tearDownCFSSLStatefulSet(cr *apiv1alpha1.Astarte, c client.Client, recorder record.EventRecorder) error {
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	// First, delete CFSSL StatefulSet and wait until it is done
	CFSSLStatefulSetName := cr.Name + "-cfssl"
	CFSSLStatefulSet := &appsv1.StatefulSet{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: CFSSLStatefulSetName, Namespace: cr.Namespace}, CFSSLStatefulSet); err != nil {
		return fmt.Errorf("Cannot retrieve CFSSL StatefulSet: %v", err)
	}
	reqLogger.Info("Tearing down the CFSSL StatefulSet.")
	if err := c.Delete(context.TODO(), CFSSLStatefulSet); err != nil {
		return fmt.Errorf("Could not tear down CFSSL StatefulSet: %v", err)
	}
	recorder.Event(cr, "Normal", apiv1alpha1.AstarteResourceEventUpgrade.String(), "Tearing down the CFSSL StatefulSet.")

	reqLogger.Info("Waiting for the CFSSL StatefulSet to go down...")
	// Now wait
	if err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		statefulSet := &appsv1.StatefulSet{}
		if err := c.Get(context.TODO(), types.NamespacedName{Name: CFSSLStatefulSetName, Namespace: cr.Namespace}, statefulSet); err != nil {
			return true, nil
		}
		return false, fmt.Errorf("Failed in waiting CFSSL StatefulSet to be teared down.")
	}); err != nil {
		recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventUpgradeError.String(),
			"Could not teard down the CFSSL StatefulSet. Upgrade will be retried")
		return err
	}
	return nil
}
