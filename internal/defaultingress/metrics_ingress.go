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
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"go.openly.dev/pointy"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	ingressv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/misc"
)

func EnsureMetricsIngress(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv2alpha1.Astarte, c client.Client, scheme *runtime.Scheme, log logr.Logger) error {
	ingressName := getMetricsIngressName(cr)
	if !pointy.BoolValue(cr.Spec.API.Deploy, true) || !pointy.BoolValue(cr.Spec.API.ServeMetrics, false) {
		// We're not deploying the Ingress, so we're stopping here.
		// However, maybe we have an Ingress to clean up?
		ingress := &networkingv1.Ingress{}
		if err := c.Get(context.TODO(), types.NamespacedName{Name: ingressName, Namespace: cr.Namespace}, ingress); err == nil {
			// Delete the ingress
			if err := c.Delete(context.Background(), ingress); err != nil {
				return err
			}
		}
		return nil
	}

	// Start with the Metrics Ingress Annotations
	annotations := getCommonIngressAnnotations(cr, parent)

	// And in case metrics can be exposed to a given subnet, the following
	// additional annotation is needed
	if cr.Spec.API.ServeMetricsToSubnet != "" {
		annotations["nginx.ingress.kubernetes.io/whitelist-source-range"] = cr.Spec.API.ServeMetricsToSubnet
	}

	// Then build the Ingress Spec
	spec := getMetricsIngressSpec(cr, parent)

	// Reconcile the Ingress
	ingress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: ingressName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, ingress, func() error {
		if e := controllerutil.SetControllerReference(cr, ingress, scheme); e != nil {
			return e
		}

		ingress.SetAnnotations(annotations)
		ingress.Spec = spec

		return nil
	})
	if err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, ingress)
	}

	return err
}

func getMetricsIngressName(cr *ingressv1alpha1.AstarteDefaultIngress) string {
	return cr.Name + "-metrics-ingress"
}

func getMetricsIngressSpec(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv2alpha1.Astarte) networkingv1.IngressSpec {
	ingressSpec := networkingv1.IngressSpec{
		// define which ingress controller will implement the ingress
		IngressClassName: getIngressClassName(cr),
		TLS:              getIngressTLS(cr, parent, false),
		Rules:            getMetricsIngressRules(cr, parent),
	}

	return ingressSpec
}

func getMetricsIngressRules(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv2alpha1.Astarte) []networkingv1.IngressRule {
	ingressRules := []networkingv1.IngressRule{}
	pathTypePrefix := networkingv1.PathTypePrefix

	// Create rules for all Astarte components
	astarteComponents := []apiv2alpha1.AstarteComponent{
		apiv2alpha1.AppEngineAPI,
		apiv2alpha1.DataUpdaterPlant,
		apiv2alpha1.Housekeeping,
		apiv2alpha1.FlowComponent,
		apiv2alpha1.Pairing,
		apiv2alpha1.RealmManagement,
		apiv2alpha1.TriggerEngine,
	}

	for _, component := range astarteComponents {
		if misc.IsAstarteComponentDeployed(parent, component) {
			servicePath := component.ServiceRelativePath()
			// TE and DUP are not publicly exposed. Thus, ServiceRelativePath() returns an empty string
			// for those components. In order to properly serve metrics, the service paths are handled
			// as special cases only in the metrics ingress.
			switch component {
			case apiv2alpha1.DataUpdaterPlant:
				servicePath = "dataupdaterplant"
			case apiv2alpha1.TriggerEngine:
				servicePath = "triggerengine"
			}

			ingressRules = append(ingressRules, networkingv1.IngressRule{
				Host: parent.Spec.API.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     fmt.Sprintf("/()(metrics)/%s($|/$)", servicePath),
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
	return ingressRules
}
