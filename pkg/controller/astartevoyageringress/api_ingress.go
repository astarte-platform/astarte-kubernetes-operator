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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	voyager "github.com/astarte-platform/astarte-kubernetes-operator/external/voyager/v1beta1"
	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/misc"
	"github.com/openlyinc/pointy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func ensureAPIIngress(cr *apiv1alpha1.AstarteVoyagerIngress, parent *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	ingressName := getAPIIngressName(cr)
	if !pointy.BoolValue(cr.Spec.API.Deploy, true) {
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
	annotations, err := getAPIIngressAnnotations(cr, parent)
	if err != nil {
		return err
	}

	// Then build the Ingress Spec
	ingressSpec, err := getAPIIngressSpec(cr, parent, c)
	if err != nil {
		return err
	}

	// Now start with the paths
	apiPaths := []voyager.HTTPIngressPath{}
	// Create rules for all Astarte components
	astarteComponents := []apiv1alpha1.AstarteComponent{apiv1alpha1.AppEngineAPI, apiv1alpha1.FlowComponent, apiv1alpha1.HousekeepingAPI, apiv1alpha1.PairingAPI, apiv1alpha1.RealmManagementAPI}
	// Should we serve /metrics?
	serveMetrics := pointy.BoolValue(cr.Spec.API.ServeMetrics, false)
	// Is the Dashboard deployed without a host?
	if misc.IsAstarteComponentDeployed(parent, apiv1alpha1.Dashboard) && cr.Spec.Dashboard.Host == "" {
		astarteComponents = append(astarteComponents, apiv1alpha1.Dashboard)
	}
	for _, component := range astarteComponents {
		if misc.IsAstarteComponentDeployed(parent, component) {
			apiPaths = append(apiPaths, getHTTPIngressPathForAstarteComponent(parent, component, serveMetrics, cr.Spec.API.ServeMetricsToSubnet))
		}
	}

	rules := []voyager.IngressRule{}
	rules = append(rules, voyager.IngressRule{
		Host:             parent.Spec.API.Host,
		IngressRuleValue: voyager.IngressRuleValue{HTTP: &voyager.HTTPIngressRuleValue{Paths: apiPaths}},
	})

	// Is the Dashboard deployed on a separate host?
	if misc.IsAstarteComponentDeployed(parent, apiv1alpha1.Dashboard) && cr.Spec.Dashboard.Host != "" {
		rules = append(rules, voyager.IngressRule{
			Host: cr.Spec.Dashboard.Host,
			IngressRuleValue: voyager.IngressRuleValue{HTTP: &voyager.HTTPIngressRuleValue{
				// Dashboard has no metrics and the /metrics name might be used in the future, so allow it
				Paths: []voyager.HTTPIngressPath{getHTTPIngressWithPath(parent, "", "dashboard", true, "")},
			}},
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

func getAPIIngressAnnotations(cr *apiv1alpha1.AstarteVoyagerIngress, parent *apiv1alpha1.Astarte) (map[string]string, error) {
	annotations := map[string]string{
		// Always use this so Astarte can behave correctly
		voyager.KeepSourceIP: strconv.FormatBool(true),
		// Meaningful options
		voyager.DefaultsOption: `{"forwardfor": "true", "dontlognull": "true"}`,
		// Tunnel is for websockets - 10m is more then enough
		voyager.DefaultsTimeOut: `{"tunnel": "10m"}`,
	}
	if cr.Spec.API.Replicas != nil {
		annotations[voyager.Replicas] = strconv.Itoa(int(pointy.Int32Value(cr.Spec.API.Replicas, 1)))
	}
	if pointy.BoolValue(cr.Spec.API.Cors, false) {
		annotations[voyager.CORSEnabled] = strconv.FormatBool(true)
	}
	if cr.Spec.API.NodeSelector != "" {
		annotations[voyager.NodeSelector] = cr.Spec.API.NodeSelector
	}
	if cr.Spec.API.Type != "" {
		annotations[voyager.LBType] = cr.Spec.API.Type
	}
	if cr.Spec.API.LoadBalancerIP != "" {
		annotations[voyager.LoadBalancerIP] = cr.Spec.API.LoadBalancerIP
	}
	if pointy.BoolValue(parent.Spec.API.SSL, true) || pointy.BoolValue(cr.Spec.Dashboard.SSL, true) {
		// Add safe-SSL options
		annotations[voyager.EnableHSTS] = strconv.FormatBool(true)
		annotations[voyager.HSTSPreload] = strconv.FormatBool(true)
		annotations[voyager.HSTSIncludeSubDomains] = strconv.FormatBool(true)
		annotations[voyager.HSTSMaxAge] = "180"
	}
	if len(cr.Spec.API.AnnotationsService) > 0 {
		// Marshal into a JSON, and call it a day.
		aS, err := json.Marshal(cr.Spec.API.AnnotationsService)
		if err != nil {
			return nil, err
		}
		annotations[voyager.ServiceAnnotations] = string(aS)
	}

	return annotations, nil
}

func getAPIIngressSpec(cr *apiv1alpha1.AstarteVoyagerIngress, parent *apiv1alpha1.Astarte, c client.Client) (voyager.IngressSpec, error) {
	// Ok - build the Ingress Spec
	ingressSpec := voyager.IngressSpec{}
	// TLS first
	if pointy.BoolValue(parent.Spec.API.SSL, true) || pointy.BoolValue(cr.Spec.Dashboard.SSL, true) {
		// Ok, we should add TLS.
		// Priority in options is: Ref - Secret Name - Let's Encrypt
		ingressTLSs := []voyager.IngressTLS{}
		apiProcessed, dashboardProcessed := false, false
		// Check API first
		if cr.Spec.API.TLSRef != nil {
			ingressTLSs = append(ingressTLSs, voyager.IngressTLS{Ref: cr.Spec.API.TLSRef, Hosts: []string{parent.Spec.API.Host}})
			apiProcessed = true
		} else if cr.Spec.API.TLSSecret != "" {
			ingressTLSs = append(ingressTLSs, voyager.IngressTLS{SecretName: cr.Spec.API.TLSSecret, Hosts: []string{parent.Spec.API.Host}})
			apiProcessed = true
		}
		// Then Dashboard - if needed
		if pointy.BoolValue(cr.Spec.Dashboard.SSL, true) && cr.Spec.Dashboard.Host != "" {
			if cr.Spec.Dashboard.TLSRef != nil {
				ingressTLSs = append(ingressTLSs, voyager.IngressTLS{Ref: cr.Spec.Dashboard.TLSRef, Hosts: []string{cr.Spec.Dashboard.Host}})
				apiProcessed = true
			} else if cr.Spec.Dashboard.TLSSecret != "" {
				ingressTLSs = append(ingressTLSs, voyager.IngressTLS{SecretName: cr.Spec.Dashboard.TLSSecret, Hosts: []string{cr.Spec.Dashboard.Host}})
				apiProcessed = true
			}
		} else {
			// No further actions to take.
			dashboardProcessed = true
		}

		// Finally, let's see if we need to add anything to Let's Encrypt
		if leIngressTLS, err := getLEFixupForAPIIngress(cr, parent, c, apiProcessed, dashboardProcessed); err != nil {
			return ingressSpec, err
		} else if leIngressTLS != nil {
			ingressTLSs = append(ingressTLSs, *leIngressTLS)
		}

		// After we got here, is there anything we need to add?
		if len(ingressTLSs) > 0 {
			ingressSpec.TLS = ingressTLSs
		}
	}

	return ingressSpec, nil
}

func getLEFixupForAPIIngress(cr *apiv1alpha1.AstarteVoyagerIngress, parent *apiv1alpha1.Astarte,
	c client.Client, apiProcessed, dashboardProcessed bool) (*voyager.IngressTLS, error) {
	if (!apiProcessed || !dashboardProcessed) && pointy.BoolValue(cr.Spec.Letsencrypt.Use, true) {
		// Are we bootstrapping?
		bootstrappingLE, err := isBootstrappingLEChallenge(cr, c)
		if err != nil {
			return nil, err
		}
		if !bootstrappingLE {
			// Which hosts do we need to add?
			hosts := []string{}
			if !apiProcessed {
				hosts = append(hosts, parent.Spec.API.Host)
			}
			if !dashboardProcessed && pointy.BoolValue(cr.Spec.Dashboard.SSL, true) && cr.Spec.Dashboard.Host != "" {
				hosts = append(hosts, cr.Spec.Dashboard.Host)
			}

			if len(hosts) > 0 {
				return &voyager.IngressTLS{
					Ref:   &voyager.LocalTypedReference{Kind: "Certificate", Name: getCertificateName(cr)},
					Hosts: hosts,
				}, nil
			}
		}
	}

	// Nothing
	return nil, nil
}

func getHTTPIngressPathForAstarteComponent(parent *apiv1alpha1.Astarte, component apiv1alpha1.AstarteComponent,
	serveMetrics bool, serveMetricsToSubnet string) voyager.HTTPIngressPath {
	return getHTTPIngressWithPath(parent, strings.Replace(component.ServiceName(), "-", "", -1), component.ServiceName(), serveMetrics, serveMetricsToSubnet)
}

func getHTTPIngressWithPath(parent *apiv1alpha1.Astarte, relativePath, serviceName string, serveMetrics bool, serveMetricsToSubnet string) voyager.HTTPIngressPath {
	// Safe HTTP headers to add in responses.
	backendSSLRules := []string{
		"http-response add-header X-Frame-Options SAMEORIGIN",
		"http-response set-header X-XSS-Protection 1;mode=block",
		"http-response set-header X-Content-Type-Options nosniff",
		"http-response set-header Referrer-Policy no-referrer-when-downgrade",
	}
	backendRules := []string{}
	metricsPath := "/metrics"

	if relativePath != "" {
		backendRules = append(backendRules, fmt.Sprintf(`reqrep ^([^\ :]*)\ /%s/(.*$) \1\ /\2`, relativePath))
		metricsPath = fmt.Sprintf("/%s/metrics", relativePath)
	}

	// Should we serve the metrics endpoint?
	switch {
	case !serveMetrics:
		backendRules = append(backendRules, fmt.Sprintf(`http-request deny if { path -i -m beg %s }`, metricsPath))
	case serveMetricsToSubnet != "":
		backendRules = append(backendRules, fmt.Sprintf(`http-request deny if { path -i -m beg %s } !{ src %s }`, metricsPath, serveMetricsToSubnet))
	}

	backendRules = append(backendRules, backendSSLRules...)
	return voyager.HTTPIngressPath{
		Path: fmt.Sprintf("/%s", relativePath),
		Backend: voyager.HTTPIngressBackend{
			IngressBackend: voyager.IngressBackend{
				ServiceName:  fmt.Sprintf("%s-%s", parent.Name, serviceName),
				ServicePort:  intstr.FromString("http"),
				BackendRules: backendRules,
			},
		},
	}
}

func getAPIIngressName(cr *apiv1alpha1.AstarteVoyagerIngress) string {
	return cr.Name + "-api-ingress"
}

func isAPIIngressReady(cr *apiv1alpha1.AstarteVoyagerIngress, c client.Client) bool {
	return isIngressReady(getAPIIngressName(cr), cr, c)
}
