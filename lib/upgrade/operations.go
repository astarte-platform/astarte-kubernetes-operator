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
	"time"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/misc"
	"github.com/openlyinc/pointy"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

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

	// Get the 0.10 queue state
	httpClient := &http.Client{}
	req, _ := http.NewRequest("GET", "http://"+rmqHost+":15672/api/queues/%2F/vmq_all", nil)
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
		respJSON := map[string]interface{}{}
		if e2 := json.Unmarshal(respBody, &respJSON); e2 != nil {
			recorder.Event(cr, "Warning", apiv1alpha1.AstarteResourceEventCriticalError.String(),
				"Unrecoverable error in querying RabbitMQ Management. Upgrade will be retried, but manual intervention is likely required")
			reqLogger.Error(e2, "Unrecoverable error in querying RabbitMQ Management")
			return false, e2
		}
		// float64 is how this is decoded by Go
		messagesReady := respJSON["messages_ready"].(float64)
		if messagesReady > 0 {
			reqLogger.Info("Waiting for RabbitMQ Data Queue to drain.", "MessagesLeft", messagesReady)
			return false, nil
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

func openRabbitMQPortForward(cr *apiv1alpha1.Astarte) (string, chan struct{}, error) {
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	var fw *portforward.PortForwarder
	var stopChannel chan struct{} = nil

	// Note that we're trying to find out whether the operator is running outside the cluster
	if _, err := k8sutil.GetOperatorNamespace(); err != nil {
		if err == k8sutil.ErrNoNamespace || err == k8sutil.ErrRunLocal {
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

func getSpecialHousekeepingMigrationProbe(path string) *v1.Probe {
	// This is a special migration probe that handles longer timeouts due to migrations.
	// Migrations can take an insane amount of time, as such we should take this into account.
	return &v1.Probe{
		Handler: v1.Handler{
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
