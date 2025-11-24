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
	"fmt"
	"strconv"

	"go.openly.dev/pointy"
	networkingv1 "k8s.io/api/networking/v1"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	ingressv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v2alpha1"
)

// getCommonIngressAnnotations returns the common annotations for AstarteDefaultIngress Ingresses
// Depending on the Ingress Controller in use, different annotations are applied.
func getCommonIngressAnnotations(cr *ingressv2alpha1.AstarteDefaultIngress, parent *apiv2alpha1.Astarte) map[string]string {
	var annotations map[string]string
	apiSslRedirect := pointy.BoolValue(parent.Spec.API.SSL, true) || pointy.BoolValue(cr.Spec.Dashboard.SSL, true)
	enableCors := pointy.BoolValue(cr.Spec.API.Cors, false)

	if !cr.HAProxyIngressControllerSelected() {
		annotations = map[string]string{
			"nginx.ingress.kubernetes.io/ssl-redirect":   strconv.FormatBool(apiSslRedirect),
			"nginx.ingress.kubernetes.io/use-regex":      "true",
			"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
			"nginx.ingress.kubernetes.io/enable-cors":    strconv.FormatBool(enableCors),
			"nginx.ingress.kubernetes.io/configuration-snippet": "more_set_headers \"X-Frame-Options: SAMEORIGIN\";\n" +
				"more_set_headers \"X-XSS-Protection: 1; mode=block\";\n" +
				"more_set_headers \"X-Content-Type-Options: nosniff\";\n" +
				"more_set_headers \"Referrer-Policy: no-referrer-when-downgrade\";",
		}
		return annotations
	}

	// From here on, we assume HAProxy Ingress Controller is in use

	annotations = map[string]string{
		"haproxy.org/backend-config-snippet": getHAProxyBackendConfig(parent),
		"haproxy.org/ssl-redirect":           strconv.FormatBool(apiSslRedirect),
		"haproxy.org/response-set-header": "\n" +
			"X-Frame-Options SAMEORIGIN\n" +
			"X-XSS-Protection 1; mode=block\n" +
			"X-Content-Type-Options nosniff\n" +
			"Referrer-Policy no-referrer-when-downgrade",
	}
	annotations = appendHAProxyCorsAnnotations(enableCors, annotations)

	return annotations
}

// appendHAProxyCorsAnnotations appends the necessary HAProxy annotations
// depending on whether CORS is enabled or not.
func appendHAProxyCorsAnnotations(enableCors bool, annotations map[string]string) (ret map[string]string) {
	if !enableCors {
		annotations["haproxy.org/cors-enable"] = "false"
		return annotations
	}

	annotations["haproxy.org/cors-enable"] = "true"
	annotations["haproxy.org/cors-allow-origin"] = "*"
	annotations["haproxy.org/cors-allow-methods"] = "GET, POST, OPTIONS, PUT, DELETE"
	annotations["haproxy.org/cors-allow-headers"] = "DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization"
	annotations["haproxy.org/cors-expose-headers"] = "Content-Length,Content-Range"

	return annotations
}

// getHAProxyBackendConfig returns the backend config snippet for HAProxy Ingresses
// Performs path rewriting only for Astarte API paths, leaving others (like dashboard) untouched.
func getHAProxyBackendConfig(cr *apiv2alpha1.Astarte) string {
	return fmt.Sprintf(`http-request replace-path /(appengine|pairing|housekeeping|realmmanagement)/(.*) /\2 if { hdr(host) -i %s }`, cr.Spec.API.Host)
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
