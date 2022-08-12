/*
  This file is part of Astarte.

  Copyright 2021 Ispirata Srl

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

package defaultingress

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	"github.com/openlyinc/pointy"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
	ingressv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/ingress/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
)

func EnsureBrokerIngress(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme, log logr.Logger) error {
	// we actually don't have an ingress, but a service which exposes the broker
	brokerServiceName := getBrokerServiceName(cr)
	if !pointy.BoolValue(cr.Spec.Broker.Deploy, true) {
		// We're not exposing the broker, so we're stopping here.
		// However, maybe we have a service to clean up?
		brokerService := &v1.Service{}
		if err := c.Get(context.TODO(), types.NamespacedName{Name: brokerServiceName, Namespace: cr.Namespace}, brokerService); err == nil {
			// Delete the broker service
			if err := c.Delete(context.TODO(), brokerService); err != nil {
				return err
			}
		}
		return nil
	}

	// Reconcile the broker service
	brokerService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: brokerServiceName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.Background(), c, brokerService, func() error {
		if e := controllerutil.SetControllerReference(cr, brokerService, scheme); e != nil {
			return e
		}
		brokerService.Spec.Selector = map[string]string{"app": fmt.Sprintf("%s-vernemq", cr.Spec.Astarte)}
		brokerService.Annotations = cr.Spec.Broker.ServiceAnnotations
		brokerService.Spec.Ports = []v1.ServicePort{
			{
				Port:       int32(pointy.Int16Value(parent.Spec.VerneMQ.Port, 8883)),
				TargetPort: intstr.FromInt(8883),
			},
		}
		brokerService.Spec.Type = cr.Spec.Broker.ServiceType
		// required to preserve client IP
		brokerService.Spec.ExternalTrafficPolicy = v1.ServiceExternalTrafficPolicyTypeLocal

		if cr.Spec.Broker.LoadBalancerIP != "" && cr.Spec.Broker.ServiceType == v1.ServiceTypeLoadBalancer {
			brokerService.Spec.LoadBalancerIP = cr.Spec.Broker.LoadBalancerIP
		}

		return nil
	})
	if err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, brokerService)
	}

	return err
}

func getBrokerServiceName(cr *ingressv1alpha1.AstarteDefaultIngress) string {
	return cr.Name + "-broker-service"
}
