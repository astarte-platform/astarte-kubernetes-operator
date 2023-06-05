/*
  This file is part of Astarte.

  Copyright 2020-23 SECO Mind Srl

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

package voyager

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/openlyinc/pointy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha2"
	voyager "github.com/astarte-platform/astarte-kubernetes-operator/external/voyager/v1beta1"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
)

func EnsureBrokerIngress(cr *apiv1alpha2.AstarteVoyagerIngress, parent *apiv1alpha2.Astarte, c client.Client, scheme *runtime.Scheme, log logr.Logger) error {
	ingressName := getBrokerIngressName(cr)
	if !pointy.BoolValue(cr.Spec.Broker.Deploy, true) {
		// We're not deploying the Ingress, so we're stopping here.
		// However, maybe we have an Ingress to clean up?
		ingress := &voyager.Ingress{}
		if err := c.Get(context.TODO(), types.NamespacedName{Name: ingressName, Namespace: cr.Namespace}, ingress); err == nil {
			// Delete the ingress
			if err := c.Delete(context.TODO(), ingress); err != nil {
				return err
			}
		}
		return nil
	}

	// Start with the Ingress Annotations
	annotations, err := getBrokerIngressAnnotations(cr, parent)
	if err != nil {
		return err
	}

	// Now get the Ingress Spec

	ingressSpec, err := getBrokerIngressSpec(cr, parent, c)
	if err != nil {
		return err
	}

	// Create rule for the Broker
	rules := []voyager.IngressRule{}
	rules = append(rules, voyager.IngressRule{
		Host: parent.Spec.VerneMQ.Host,
		IngressRuleValue: voyager.IngressRuleValue{
			TCP: &voyager.TCPIngressRuleValue{
				Port: intstr.FromInt(int(pointy.Int16Value(parent.Spec.VerneMQ.Port, 8883))),
				Backend: voyager.IngressBackend{
					ServiceName: parent.Name + "-vernemq",
					ServicePort: intstr.FromString("mqtt-reverse"),
					BackendRules: []string{
						"balance source",
					},
				},
			},
		},
	})

	if pointy.BoolValue(cr.Spec.Letsencrypt.Use, true) &&
		(cr.Spec.Letsencrypt.ChallengeProvider.HTTP != nil || pointy.BoolValue(cr.Spec.Letsencrypt.AutoHTTPChallenge, false)) {
		// The Voyager operator will try to add this rule if the HTTP challenge is enabled, so we
		// must add it too on our side, otherwise the two operators will fight over the state of the
		// ingress, resulting in the failure of the HTTP-01 challenge.
		rules = append(rules, voyager.IngressRule{
			IngressRuleValue: voyager.IngressRuleValue{
				HTTP: &voyager.HTTPIngressRuleValue{
					NoTLS: true,
					Paths: []voyager.HTTPIngressPath{
						{
							Path: "/.well-known/acme-challenge/",
							Backend: voyager.HTTPIngressBackend{
								IngressBackend: voyager.IngressBackend{
									ServiceName: "voyager-operator.kube-system",
									ServicePort: intstr.FromInt(56791),
								},
							},
						},
					},
				},
			},
		})
	}

	ingressSpec.Rules = rules

	// Reconcile the Ingress
	ingress := &voyager.Ingress{ObjectMeta: metav1.ObjectMeta{Name: ingressName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, ingress, func() error {
		if e := controllerutil.SetControllerReference(cr, ingress, scheme); e != nil {
			return e
		}

		// Reconcile the Spec
		ingress.SetAnnotations(annotations)
		ingress.Spec = ingressSpec
		return nil
	})
	if err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, ingress)
	}

	return err
}

func getBrokerIngressAnnotations(cr *apiv1alpha2.AstarteVoyagerIngress, parent *apiv1alpha2.Astarte) (map[string]string, error) {
	annotations := map[string]string{
		// Always use this so Astarte can behave correctly
		voyager.KeepSourceIP:        strconv.FormatBool(true),
		voyager.AuthTLSVerifyClient: "required",
		voyager.AuthTLSSecret:       parent.Name + "-cfssl-ca",
		// Soft-cap this to a meaningful value
		voyager.MaxConnections: strconv.Itoa(pointy.IntValue(cr.Spec.Broker.MaxConnections, 10000)),
		// Meaningful options for MQTT
		voyager.DefaultsOption: `{"tcplog": "true", "dontlognull": "true", "clitcpka": "true"}`,
		// Reasonable timeouts for long PINGREQs
		voyager.DefaultsTimeOut: `{"connect": "30s", "server": "1h", "client": "1h", "tunnel": "1h"}`,
	}
	if cr.Spec.Broker.Replicas != nil {
		annotations[voyager.Replicas] = strconv.Itoa(int(pointy.Int32Value(cr.Spec.Broker.Replicas, 1)))
	}
	if cr.Spec.Broker.NodeSelector != "" {
		annotations[voyager.NodeSelector] = cr.Spec.Broker.NodeSelector
	}
	if cr.Spec.Broker.Type != "" {
		annotations[voyager.LBType] = cr.Spec.Broker.Type
	}
	if cr.Spec.Broker.LoadBalancerIP != "" {
		annotations[voyager.LoadBalancerIP] = cr.Spec.Broker.LoadBalancerIP
	}
	if len(cr.Spec.Broker.AnnotationsService) > 0 {
		// Marshal into a JSON, and call it a day.
		aS, err := json.Marshal(cr.Spec.Broker.AnnotationsService)
		if err != nil {
			return nil, err
		}
		annotations[voyager.ServiceAnnotations] = string(aS)
	}

	return annotations, nil
}

func getBrokerIngressSpec(cr *apiv1alpha2.AstarteVoyagerIngress, parent *apiv1alpha2.Astarte, c client.Client) (voyager.IngressSpec, error) {
	// Build the Ingress Spec
	ingressSpec := voyager.IngressSpec{}
	var ingressTLS *voyager.IngressTLS

	// TLS first
	// Priority in options is: Ref - Secret Name - Let's Encrypt
	switch {
	case cr.Spec.Broker.TLSRef != nil:
		ingressTLS = &voyager.IngressTLS{Ref: cr.Spec.Broker.TLSRef, Hosts: []string{parent.Spec.VerneMQ.Host}}
	case cr.Spec.Broker.TLSSecret != "":
		ingressTLS = &voyager.IngressTLS{SecretName: cr.Spec.Broker.TLSSecret, Hosts: []string{parent.Spec.VerneMQ.Host}}
	case pointy.BoolValue(cr.Spec.Letsencrypt.Use, true):
		// Are we bootstrapping?
		if bootstrappingLE, e := isBootstrappingLEChallenge(cr, c); e != nil {
			return ingressSpec, e
		} else if !bootstrappingLE {
			ingressTLS = &voyager.IngressTLS{
				Ref:   &voyager.LocalTypedReference{Kind: "Certificate", Name: getCertificateName(cr)},
				Hosts: []string{parent.Spec.VerneMQ.Host},
			}
		}
	}

	if ingressTLS != nil {
		ingressSpec.TLS = []voyager.IngressTLS{*ingressTLS}
	}

	return ingressSpec, nil
}

func getBrokerIngressName(cr *apiv1alpha2.AstarteVoyagerIngress) string {
	return cr.Name + "-vernemq-ingress"
}

func isBrokerIngressReady(cr *apiv1alpha2.AstarteVoyagerIngress, c client.Client) bool {
	return isIngressReady(getBrokerIngressName(cr), cr, c)
}
