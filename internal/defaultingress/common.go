/*
This file is part of Astarte.

Copyright 2020-25 SECO Mind Srl.

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
	"strconv"

	"go.openly.dev/pointy"
	networkingv1 "k8s.io/api/networking/v1"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	ingressv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v2alpha1"
)

func getCommonIngressAnnotations(cr *ingressv2alpha1.AstarteDefaultIngress, parent *apiv2alpha1.Astarte) map[string]string {
	apiSslRedirect := pointy.BoolValue(parent.Spec.API.SSL, true) || pointy.BoolValue(cr.Spec.Dashboard.SSL, true)
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/ssl-redirect":   strconv.FormatBool(apiSslRedirect),
		"nginx.ingress.kubernetes.io/use-regex":      "true",
		"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
		"nginx.ingress.kubernetes.io/configuration-snippet": "more_set_headers \"X-Frame-Options: SAMEORIGIN\";\n" +
			"more_set_headers \"X-XSS-Protection: 1; mode=block\";\n" +
			"more_set_headers \"X-Content-Type-Options: nosniff\";\n" +
			"more_set_headers \"Referrer-Policy: no-referrer-when-downgrade\";",
	}

	if pointy.BoolValue(cr.Spec.API.Cors, false) {
		annotations["nginx.ingress.kubernetes.io/enable-cors"] = strconv.FormatBool(true)
	}

	return annotations
}

// TODO handle with kubebuilder defaults
func getIngressClassName(cr *ingressv2alpha1.AstarteDefaultIngress) *string {
	if cr.Spec.IngressClass == "" {
		return pointy.String("nginx")
	}
	return pointy.String(cr.Spec.IngressClass)
}

func getIngressTLS(cr *ingressv2alpha1.AstarteDefaultIngress, parent *apiv2alpha1.Astarte, includeDashboard bool) []networkingv1.IngressTLS {
	ingressTLSs := []networkingv1.IngressTLS{}

	// Check API
	if pointy.BoolValue(parent.Spec.API.SSL, true) || pointy.BoolValue(cr.Spec.Dashboard.SSL, true) {
		secretName := cr.Spec.TLSSecret
		if cr.Spec.API.TLSSecret != "" {
			secretName = cr.Spec.API.TLSSecret
		}
		// Other cases are rejected by validation webhooks

		ingressTLSs = append(ingressTLSs, networkingv1.IngressTLS{
			Hosts:      []string{parent.Spec.API.Host},
			SecretName: secretName,
		})
	}

	// dashboard TLS is not needed when dealing with the metrics ingress
	if includeDashboard {
		// Then check the dashboard, if needed
		if pointy.BoolValue(cr.Spec.Dashboard.Deploy, true) && pointy.BoolValue(cr.Spec.Dashboard.SSL, true) && cr.Spec.Dashboard.Host != "" {
			secretName := cr.Spec.TLSSecret
			if cr.Spec.Dashboard.TLSSecret != "" {
				secretName = cr.Spec.Dashboard.TLSSecret
			}
			// Other cases are rejected by validation webhooks

			ingressTLSs = append(ingressTLSs, networkingv1.IngressTLS{
				Hosts:      []string{cr.Spec.Dashboard.Host},
				SecretName: secretName,
			})
		}
	}
	return ingressTLSs
}
