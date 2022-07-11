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
	"strconv"

	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	"github.com/openlyinc/pointy"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
	ingressv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/ingress/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
)

func EnsureAPIIngress(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme, log logr.Logger) error {
	ingressName := getAPIIngressName(cr)
	if !pointy.BoolValue(cr.Spec.API.Deploy, true) {
		// We're not deploying the Ingress, so we're stopping here.
		// However, maybe we have an Ingress to clean up?
		ingress := &networkingv1.Ingress{}
		if err := c.Get(context.TODO(), types.NamespacedName{Name: ingressName, Namespace: cr.Namespace}, ingress); err == nil {
			// Delete the ingress
			if err := c.Delete(context.TODO(), ingress); err != nil {
				return err
			}
		}
		return nil
	}

	// Ensure the configMap is properly configured
	configMapName := getConfigMapName(cr)

	configMap := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: configMapName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, configMap, func() error {
		if e := controllerutil.SetControllerReference(cr, configMap, scheme); e != nil {
			return e
		}
		configMap.Data = map[string]string{
			"use-forwarded-headers": "true",
		}

		if pointy.BoolValue(parent.Spec.API.SSL, true) || pointy.BoolValue(cr.Spec.Dashboard.SSL, true) {
			configMap.Data["hsts"] = strconv.FormatBool(true)
			configMap.Data["hsts-preload"] = strconv.FormatBool(true)
			configMap.Data["hsts-include-subdomains"] = strconv.FormatBool(true)
			configMap.Data["hsts-max-age"] = "180"
		}
		return nil
	})
	if err != nil {
		return err
	}
	misc.LogCreateOrUpdateOperationResult(log, result, cr, configMap)

	// Start with the Ingress Annotations
	annotations := getAPIIngressAnnotations(cr, parent)

	// Then build the Ingress Spec
	ingressSpec := getAPIIngressSpec(cr, parent)

	// Reconcile the Ingress
	ingress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: ingressName, Namespace: cr.Namespace}}
	result, err = controllerutil.CreateOrUpdate(context.TODO(), c, ingress, func() error {
		if e := controllerutil.SetControllerReference(cr, ingress, scheme); e != nil {
			return e
		}

		ingress.SetAnnotations(annotations)
		ingress.Spec = ingressSpec

		return nil
	})
	if err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, ingress)
	}

	return err
}

func getAPIIngressName(cr *ingressv1alpha1.AstarteDefaultIngress) string {
	return cr.Name + "-api-ingress"
}

func getConfigMapName(cr *ingressv1alpha1.AstarteDefaultIngress) string {
	return cr.Name + "-api-ingress-config"
}

func getAPIIngressAnnotations(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv1alpha1.Astarte) map[string]string {
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

	// Should we serve /metrics?
	if !pointy.BoolValue(cr.Spec.API.ServeMetrics, false) {
		allowSubnetAnnotation := ""
		if cr.Spec.API.ServeMetricsToSubnet != "" {
			allowSubnetAnnotation = fmt.Sprintf("allow %s;\n", cr.Spec.API.ServeMetricsToSubnet)
		}

		serverSnippetValue := fmt.Sprintf("location ~* \"/(appengine|flow|housekeeping|pairing|realmmanagement)/metrics\" {\n"+
			"%s"+
			"  deny all;\n"+
			"  return 404;\n"+
			"}", allowSubnetAnnotation)

		annotations["nginx.ingress.kubernetes.io/server-snippet"] = serverSnippetValue
	}

	// Should we enable cors?
	if pointy.BoolValue(cr.Spec.API.Cors, false) {
		annotations["nginx.ingress.kubernetes.io/enable-cors"] = strconv.FormatBool(true)
	}

	return annotations
}

func getAPIIngressSpec(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv1alpha1.Astarte) networkingv1.IngressSpec {
	ingressSpec := networkingv1.IngressSpec{
		// define which ingress controller will implement the ingress
		IngressClassName: getIngressClassName(cr),
		TLS:              getAPIIngressTLS(cr, parent),
		Rules:            getAPIIngressRules(cr, parent),
	}

	return ingressSpec
}

// TODO handle with kubebuilder defaults
func getIngressClassName(cr *ingressv1alpha1.AstarteDefaultIngress) *string {
	if cr.Spec.IngressClass == "" {
		return pointy.String("nginx")
	}
	return pointy.String(cr.Spec.IngressClass)
}

func getAPIIngressTLS(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv1alpha1.Astarte) []networkingv1.IngressTLS {
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
	return ingressTLSs
}

func getAPIIngressRules(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv1alpha1.Astarte) []networkingv1.IngressRule {
	ingressRules := []networkingv1.IngressRule{}
	pathTypePrefix := networkingv1.PathTypePrefix

	// Create rules for all Astarte components
	astarteComponents := []apiv1alpha1.AstarteComponent{apiv1alpha1.AppEngineAPI, apiv1alpha1.FlowComponent, apiv1alpha1.PairingAPI, apiv1alpha1.RealmManagementAPI}

	// are we supposed to expose housekeeping?
	if pointy.BoolValue(cr.Spec.API.ExposeHousekeeping, true) {
		astarteComponents = append(astarteComponents, apiv1alpha1.HousekeepingAPI)
	}

	for _, component := range astarteComponents {
		if misc.IsAstarteComponentDeployed(parent, component) {
			ingressRules = append(ingressRules, networkingv1.IngressRule{
				Host: parent.Spec.API.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     fmt.Sprintf("/%s(/|$)(.*)", component.ServiceRelativePath()),
								PathType: &pathTypePrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: cr.Spec.Astarte + "-" + component.ServiceName(),
										Port: networkingv1.ServiceBackendPort{Name: "http"},
									},
								},
							},
						},
					},
				},
			})
		}
	}

	// and handle the Dashboard, if needed
	if pointy.BoolValue(cr.Spec.Dashboard.Deploy, true) {
		theDashboard := apiv1alpha1.Dashboard
		if misc.IsAstarteComponentDeployed(parent, apiv1alpha1.Dashboard) {
			ingressRules = append(ingressRules, networkingv1.IngressRule{
				Host: getDashboardHost(cr, parent),
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     getDashboardServiceRelativePath(cr),
								PathType: &pathTypePrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: cr.Spec.Astarte + "-" + theDashboard.ServiceName(),
										Port: networkingv1.ServiceBackendPort{Name: "http"},
									},
								},
							},
						},
					},
				},
			})
		}
	}

	return ingressRules
}

func getDashboardHost(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv1alpha1.Astarte) string {
	// Is the Dashboard deployed without a host?
	if cr.Spec.Dashboard.Host == "" {
		return parent.Spec.API.Host
	}
	return cr.Spec.Dashboard.Host
}

func getDashboardServiceRelativePath(cr *ingressv1alpha1.AstarteDefaultIngress) string {
	// Is the Dashboard deployed without a host?
	theDashboard := apiv1alpha1.Dashboard
	if cr.Spec.Dashboard.Host == "" {
		return fmt.Sprintf("/%s(/|$)(.*)", theDashboard.ServiceRelativePath())
	}
	return "/()(.*)"
}
