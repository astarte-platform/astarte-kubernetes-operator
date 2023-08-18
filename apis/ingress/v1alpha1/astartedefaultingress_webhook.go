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

package v1alpha1

import (
	"context"
	"errors"
	"fmt"

	"github.com/openlyinc/pointy"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha2"
)

// log is for logging in this package.
var (
	astartedefaultingresslog = logf.Log.WithName("astartedefaultingress-resource")
	c                        client.Client
)

func (r *AstarteDefaultIngress) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-ingress-astarte-platform-org-v1alpha1-astartedefaultingress,mutating=true,failurePolicy=fail,sideEffects=None,groups=ingress.astarte-platform.org,resources=astartedefaultingresses,verbs=create;update,versions=v1alpha1,name=mastartedefaultingress.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &AstarteDefaultIngress{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AstarteDefaultIngress) Default() {
	astartedefaultingresslog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

//+kubebuilder:webhook:path=/validate-ingress-astarte-platform-org-v1alpha1-astartedefaultingress,mutating=false,failurePolicy=fail,sideEffects=None,groups=ingress.astarte-platform.org,resources=astartedefaultingresses,verbs=create;update,versions=v1alpha1,name=vastartedefaultingress.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &AstarteDefaultIngress{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AstarteDefaultIngress) ValidateCreate() error {
	astartedefaultingresslog.Info("validate create", "name", r.Name)

	return r.validateAstarteDefaultIngress()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AstarteDefaultIngress) ValidateUpdate(old runtime.Object) error {
	astartedefaultingresslog.Info("validate update", "name", r.Name)

	return r.validateAstarteDefaultIngress()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AstarteDefaultIngress) ValidateDelete() error {
	astartedefaultingresslog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *AstarteDefaultIngress) validateAstarteDefaultIngress() error {
	allErrors := field.ErrorList{}

	astarte, astarteFoundErr := r.validateReferencedAstarte(c)
	if astarteFoundErr != nil {
		allErrors = append(allErrors, astarteFoundErr)
	}
	// if astarte is not found, do not check the api and broker config as they are not available anyway
	if astarteFoundErr == nil {
		if err := r.validateBrokerTLSConfig(astarte); err != nil {
			allErrors = append(allErrors, err)
		}
		if err := r.validateAPITLSConfig(astarte); err != nil {
			allErrors = append(allErrors, err)
		}
	}
	if err := r.validateBrokerServiceType(); err != nil {
		allErrors = append(allErrors, err)
	}
	if err := r.validateDashboardTLSConfig(); err != nil {
		allErrors = append(allErrors, err)
	}
	if err := r.validateMetricsIngressSetup(); err != nil {
		allErrors = append(allErrors, err)
	}

	allErrors = append(allErrors, r.validateTLSSecretExistence(c)...)

	if len(allErrors) == 0 {
		return nil
	}
	return apierrors.NewInvalid(
		schema.GroupKind{Group: "ingress", Kind: "AstarteDefaultIngress"},
		r.Name,
		allErrors,
	)
}

// TODO use kubebuilder defaults
func (r *AstarteDefaultIngress) validateBrokerServiceType() *field.Error {
	if pointy.BoolValue(r.Spec.Broker.Deploy, true) {
		if r.Spec.Broker.ServiceType == v1.ServiceTypeNodePort || r.Spec.Broker.ServiceType == v1.ServiceTypeLoadBalancer {
			return nil
		}

		fldPath := field.NewPath("spec").Child("broker").Child("serviceType")
		err := errors.New("Wrong broker service type. Allowed values: LoadBalancer, NodePort.")

		astartedefaultingresslog.Error(err, "Allowed service types for the Broker are: LoadBalancer and NodePort")
		return field.Invalid(fldPath, r.Spec.Broker.ServiceType, err.Error())
	}
	return nil
}

func (r *AstarteDefaultIngress) validateReferencedAstarte(c client.Client) (*apiv1alpha2.Astarte, *field.Error) {
	fldPath := field.NewPath("spec").Child("astarte")

	// ensure that the referenced Astarte instance exists
	theAstarte := &apiv1alpha2.Astarte{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: r.Spec.Astarte, Namespace: r.Namespace}, theAstarte); err != nil {
		astartedefaultingresslog.Error(err, "Could not find the referenced Astarte.")
		return nil, field.NotFound(fldPath, r.Spec.Astarte)
	}
	return theAstarte, nil
}

func (r *AstarteDefaultIngress) validateBrokerTLSConfig(astarte *apiv1alpha2.Astarte) *field.Error {
	if !pointy.BoolValue(astarte.Spec.VerneMQ.SSLListener, false) && pointy.BoolValue(r.Spec.Broker.Deploy, true) {
		fldPath := field.NewPath("spec").Child("broker").Child("deploy")
		return field.Invalid(fldPath, astarte.Spec.VerneMQ.SSLListenerCertSecretName,
			"When deploying Broker Ingress, VerneMQ SSLListener must be enabled in the main Astarte resource.")
	}

	return nil
}

func (r *AstarteDefaultIngress) validateDashboardTLSConfig() *field.Error {
	if r.Spec.TLSSecret == "" && pointy.BoolValue(r.Spec.Dashboard.SSL, true) &&
		pointy.BoolValue(r.Spec.Dashboard.Deploy, true) && r.Spec.Dashboard.TLSSecret == "" {
		fldPath := field.NewPath("spec").Child("dashboard").Child("tlsSecret")
		return field.Required(fldPath, "Requested SSL support for Dashboard, but no TLS Secret provided")
	}
	return nil
}

func (r *AstarteDefaultIngress) validateAPITLSConfig(astarte *apiv1alpha2.Astarte) *field.Error {
	if pointy.BoolValue(astarte.Spec.API.SSL, true) && r.Spec.TLSSecret == "" &&
		r.Spec.API.TLSSecret == "" && pointy.BoolValue(r.Spec.API.Deploy, true) {
		fldPath := field.NewPath("spec").Child("api").Child("tlsSecret")
		return field.Required(fldPath, "Requested SSL support for API, but no TLS Secret provided")
	}
	return nil
}

func (r *AstarteDefaultIngress) validateMetricsIngressSetup() *field.Error {
	if !pointy.BoolValue(r.Spec.API.ServeMetrics, false) && r.Spec.API.ServeMetricsToSubnet != "" {
		fldPath := field.NewPath("spec").Child("api").Child("serveMetricsToSubnet")
		return field.Forbidden(fldPath, "serveMetricsToSubnet must not be set when serveMetrics is set to false.")
	}
	return nil
}

func (r *AstarteDefaultIngress) validateTLSSecretExistence(c client.Client) field.ErrorList {
	allErrs := field.ErrorList{}

	if r.Spec.TLSSecret != "" && (r.Spec.API.TLSSecret == "" || (pointy.BoolValue(r.Spec.Dashboard.SSL, true) &&
		pointy.BoolValue(r.Spec.Dashboard.Deploy, true) && r.Spec.Dashboard.TLSSecret == "")) {

		if err := getSecret(c, r.Spec.TLSSecret, r.Namespace, field.NewPath("spec").Child("tlsSecret")); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if r.Spec.API.TLSSecret != "" {
		if err := getSecret(c, r.Spec.API.TLSSecret, r.Namespace, field.NewPath("spec").Child("api").Child("tlsSecret")); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if pointy.BoolValue(r.Spec.Dashboard.SSL, true) && pointy.BoolValue(r.Spec.Dashboard.Deploy, true) && r.Spec.Dashboard.TLSSecret != "" {
		if err := getSecret(c, r.Spec.Dashboard.TLSSecret, r.Namespace, field.NewPath("spec").Child("dashboard").Child("tlsSecret")); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	return allErrs
}

func getSecret(c client.Client, secretName string, namespace string, fldPath *field.Path) *field.Error {
	theSecret := &v1.Secret{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: namespace}, theSecret); err != nil {
		astartedefaultingresslog.Error(err, fmt.Sprintf("The secret %s does not exist in namespace %s.", secretName, namespace))
		return field.NotFound(fldPath, secretName)
	}
	return nil
}
