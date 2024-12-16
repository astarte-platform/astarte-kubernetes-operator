/*
Copyright 2024.

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

	"github.com/go-logr/logr"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
	ingressv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/misc"
)

func EnsureAPIIngress(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv1alpha2.Astarte, c client.Client, scheme *runtime.Scheme, log logr.Logger) error {
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
	annotations := getCommonIngressAnnotations(cr, parent)

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

func getAPIIngressSpec(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv1alpha2.Astarte) networkingv1.IngressSpec {
	ingressSpec := networkingv1.IngressSpec{
		// define which ingress controller will implement the ingress
		IngressClassName: getIngressClassName(cr),
		TLS:              getIngressTLS(cr, parent, true),
		Rules:            getAPIIngressRules(cr, parent),
	}

	return ingressSpec
}

func getAPIIngressRules(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv1alpha2.Astarte) []networkingv1.IngressRule {
	ingressRules := []networkingv1.IngressRule{}
	pathTypePrefix := networkingv1.PathTypePrefix

	// Create rules for all Astarte components
	astarteComponents := []apiv1alpha2.AstarteComponent{apiv1alpha2.AppEngineAPI, apiv1alpha2.FlowComponent, apiv1alpha2.PairingAPI, apiv1alpha2.RealmManagementAPI}

	// are we supposed to expose housekeeping?
	if pointy.BoolValue(cr.Spec.API.ExposeHousekeeping, true) {
		astarteComponents = append(astarteComponents, apiv1alpha2.HousekeepingAPI)
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
		theDashboard := apiv1alpha2.Dashboard
		if misc.IsAstarteComponentDeployed(parent, apiv1alpha2.Dashboard) {
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

func getDashboardHost(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv1alpha2.Astarte) string {
	// Is the Dashboard deployed without a host?
	if cr.Spec.Dashboard.Host == "" {
		return parent.Spec.API.Host
	}
	return cr.Spec.Dashboard.Host
}

func getDashboardServiceRelativePath(cr *ingressv1alpha1.AstarteDefaultIngress) string {
	// Is the Dashboard deployed without a host?
	theDashboard := apiv1alpha2.Dashboard
	if cr.Spec.Dashboard.Host == "" {
		return fmt.Sprintf("/%s(/|$)(.*)", theDashboard.ServiceRelativePath())
	}
	return "/()(.*)"
}
