/*
  This file is part of Astarte.

  Copyright 2022 SECO Mind Srl

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

	"github.com/go-logr/logr"
	"github.com/openlyinc/pointy"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"

	ingressv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/ingress/v1alpha1"
)

func (r *ReconcileHelper) ComputeADIStatusResource(reqLogger logr.Logger, instance *ingressv1alpha1.AstarteDefaultIngress) ingressv1alpha1.AstarteDefaultIngressStatus {
	newStatus := instance.Status
	newStatus.APIStatus = r.computeAPIStatus(reqLogger, instance)
	newStatus.BrokerStatus = r.computeBrokerStatus(reqLogger, instance)

	return newStatus
}

func (r *ReconcileHelper) computeAPIStatus(reqLogger logr.Logger, instance *ingressv1alpha1.AstarteDefaultIngress) networkingv1.IngressStatus {
	if !pointy.BoolValue(instance.Spec.API.Deploy, true) {
		return networkingv1.IngressStatus{}
	}

	apiIngress := &networkingv1.Ingress{}
	if err := r.Client.Get(context.Background(), types.NamespacedName{Name: instance.Name + "-api-ingress", Namespace: instance.Namespace}, apiIngress); err != nil {
		reqLogger.V(1).Info("Could not Get Astarte API Ingress to compute its status")
		return networkingv1.IngressStatus{}
	}

	return apiIngress.Status
}

func (r *ReconcileHelper) computeBrokerStatus(reqLogger logr.Logger, instance *ingressv1alpha1.AstarteDefaultIngress) corev1.ServiceStatus {
	if !pointy.BoolValue(instance.Spec.Broker.Deploy, true) {
		return corev1.ServiceStatus{}
	}

	brokerService := &corev1.Service{}
	if err := r.Client.Get(context.Background(), types.NamespacedName{Name: instance.Name + "-broker-service", Namespace: instance.Namespace}, brokerService); err != nil {
		reqLogger.V(1).Info("Could not Get Astarte Broker service to compute its status")
		return corev1.ServiceStatus{}
	}

	return brokerService.Status
}
