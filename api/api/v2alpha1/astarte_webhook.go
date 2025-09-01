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

package v2alpha1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.openly.dev/pointy"
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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var (
	astartelog = logf.Log.WithName("astarte-resource")
	c          client.Client
)

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Astarte) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-api-astarte-platform-org-v2alpha1-astarte,mutating=true,failurePolicy=fail,sideEffects=None,groups=api.astarte-platform.org,resources=astartes,verbs=create;update,versions=v2alpha1,name=mastarte.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Astarte{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Astarte) Default() {
	astartelog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-api-astarte-platform-org-v2alpha1-astarte,mutating=false,failurePolicy=fail,sideEffects=None,groups=api.astarte-platform.org,resources=astartes,verbs=create;update,versions=v2alpha1,name=vastarte.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Astarte{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Astarte) ValidateCreate() (admission.Warnings, error) {
	allErrs := field.ErrorList{}

	astartelog.Info("validate create", "name", r.Name)

	if err := r.validateCreateAstarteInstanceID(); err != nil {
		allErrs = append(allErrs, err)
	}

	if errList := r.validateCreateAstarteSystemKeyspace(); errList != nil {
		allErrs = append(allErrs, errList...)
	}

	if errList := r.validateAstarte(); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		schema.GroupKind{Group: "api", Kind: "Astarte"},
		r.Name,
		allErrs,
	)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Astarte) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	allErrs := field.ErrorList{}

	astartelog.Info("validate update", "name", r.Name)

	oldAstarte, _ := old.(*Astarte)
	if err := r.validateUpdateAstarteInstanceID(oldAstarte); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateUpdateAstarteSystemKeyspace(oldAstarte); err != nil {
		allErrs = append(allErrs, err)
	}

	if errList := r.validateAstarte(); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		schema.GroupKind{Group: "api", Kind: "Astarte"},
		r.Name,
		allErrs,
	)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Astarte) ValidateDelete() (admission.Warnings, error) {
	astartelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

func (r *Astarte) validateCreateAstarteInstanceID() *field.Error {
	astarteList := &AstarteList{}
	if clientErr := c.List(context.Background(), astarteList); clientErr != nil {
		err := errors.New("cannot list astarte instances in the cluster. Please retry.")
		astartelog.Info(clientErr.Error(), "details", err.Error())
		return field.InternalError(field.NewPath(""), err)
	}

	for _, otherAstarte := range astarteList.Items {
		if r.Spec.AstarteInstanceID == otherAstarte.Spec.AstarteInstanceID {
			fldPath := field.NewPath("spec").Child("astarteInstanceID")
			err := errors.New("invalid astarteInstanceID: the chosen ID is already in use")

			astartelog.Info(err.Error(), "astarteInstanceID", r.Spec.AstarteInstanceID)
			return field.Invalid(fldPath, r.Spec.AstarteInstanceID, err.Error())
		}
	}

	return nil
}

func (r *Astarte) validateUpdateAstarteInstanceID(oldAstarte *Astarte) *field.Error {
	if r.Spec.AstarteInstanceID != oldAstarte.Spec.AstarteInstanceID {
		fldPath := field.NewPath("spec").Child("astarteInstanceID")
		err := errors.New("the astarteInstanceId cannot be updated since it is immutable for your Astarte instance")

		astartelog.Info(err.Error(), "astarteInstanceID", r.Spec.AstarteInstanceID)
		return field.Invalid(fldPath, r.Spec.AstarteInstanceID, err.Error())
	}

	return nil
}

func (r *Astarte) validateAstarte() field.ErrorList {
	allErrs := field.ErrorList{}

	if pointy.BoolValue(r.Spec.VerneMQ.SSLListener, false) {
		// check that SSLListenerCertSecretName is set
		if r.Spec.VerneMQ.SSLListenerCertSecretName == "" {
			fldPath := field.NewPath("spec").Child("vernemq").Child("sslListenerCertSecretName")
			err := errors.New("sslListenerCertSecretName not set")
			astartelog.Info(err.Error())

			allErrs = append(allErrs, field.Invalid(fldPath, r.Spec.VerneMQ.SSLListenerCertSecretName, err.Error()))
		}

		// ensure the TLS secret is present
		theSecret := &v1.Secret{}
		if err := c.Get(context.Background(), types.NamespacedName{Name: r.Spec.VerneMQ.SSLListenerCertSecretName, Namespace: r.Namespace}, theSecret); err != nil {
			fldPath := field.NewPath("spec").Child("vernemq").Child("sslListenerCertSecretName")
			astartelog.Info(err.Error())
			allErrs = append(allErrs, field.NotFound(fldPath, err.Error()))
		}

		if err := r.validateAstartePriorityClasses(); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if errList := r.validatePodLabelsForClusteredResources(); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if err := validateAutoscalerForClusteredResources(r); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateCFSSLDefinition(); err != nil {
		allErrs = append(allErrs, err)
	}

	return allErrs
}

func (r *Astarte) validatePodLabelsForClusteredResources() field.ErrorList {
	allErrs := field.ErrorList{}

	resources := []PodLabelsGetter{r.Spec.VerneMQ.AstarteGenericClusteredResource,
		r.Spec.Components.Flow.AstarteGenericClusteredResource, r.Spec.Components.Housekeeping, r.Spec.Components.RealmManagement, r.Spec.Components.Pairing,
		r.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource,
		r.Spec.Components.TriggerEngine.AstarteGenericClusteredResource, r.Spec.Components.Dashboard.AstarteGenericClusteredResource, r.Spec.CFSSL}
	for _, v := range resources {
		if errList := validatePodLabelsForClusteredResource(v); len(errList) > 0 {
			allErrs = append(allErrs, errList...)
		}
	}

	return allErrs
}

func validatePodLabelsForClusteredResource(r PodLabelsGetter) field.ErrorList {
	allErrs := field.ErrorList{}
	for k := range r.GetPodLabels() {
		if k == "component" || k == "app" || strings.HasPrefix(k, "astarte-") || strings.HasPrefix(k, "flow-") {
			fldPath := field.NewPath("podLabels")
			err := errors.New("invalid label key: can't be any of 'app', 'component', 'astarte-*', 'flow-*'")
			astartelog.Info(err.Error(), "label", k)

			allErrs = append(allErrs, field.Invalid(fldPath, k, err.Error()))
		}
	}
	return allErrs
}

func validateAutoscalerForClusteredResources(r *Astarte) *field.Error {
	// We have no constraints on autoscaling except for these components
	excludedResources := []AstarteGenericClusteredResource{
		r.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource,
	}

	return validateAutoscalerForClusteredResourcesExcluding(r, excludedResources)
}

func validateAutoscalerForClusteredResourcesExcluding(r *Astarte, excluded []AstarteGenericClusteredResource) *field.Error {
	if r.Spec.Features.Autoscaling {
		for _, v := range excluded {
			if v.Autoscale != nil && v.Autoscale.Horizontal != "" {
				fldPath := field.NewPath("")
				err := errors.New("invalid autoscaler: cannot autoscale horizontally RabbitMQ, DataUpdaterPlant or Cassandra")
				astartelog.Info(err.Error())
				return field.Invalid(fldPath, "", err.Error())
			}
		}
	}
	return nil
}

func (r *Astarte) validateAstartePriorityClasses() *field.Error {
	if r.Spec.Features.AstartePodPriorities.IsEnabled() {
		return r.validatePriorityClassesValues()
	}

	return nil
}

func (r *Astarte) validatePriorityClassesValues() *field.Error {
	// default values guarantee pointers are not nil
	highPriorityValue := *r.Spec.Features.AstartePodPriorities.AstarteHighPriority
	midPriorityValue := *r.Spec.Features.AstartePodPriorities.AstarteMidPriority
	lowPriorityValue := *r.Spec.Features.AstartePodPriorities.AstarteLowPriority
	if midPriorityValue > highPriorityValue || lowPriorityValue > midPriorityValue {
		err := errors.New("Astarte PriorityClass values are incoherent")
		astartelog.Info(err.Error())
		fldPath := field.NewPath("spec").Child("features").Child("astarte{Low|Medium|High}Priority")

		return field.Invalid(fldPath, "", err.Error())
	}
	return nil
}

func (r *Astarte) validateCreateAstarteSystemKeyspace() field.ErrorList {
	allErrs := field.ErrorList{}
	ask := r.Spec.Cassandra.AstarteSystemKeyspace

	if ask.ReplicationStrategy == "SimpleStrategy" {
		// replication factor must be odd
		if ask.ReplicationFactor%2 == 0 {
			err := errors.New("invalid replication factor: it must be odd")
			astartelog.Info(err.Error())
			fldPath := field.NewPath("spec").Child("cassandra").Child("astarteSystemKeyspace").Child("replicationFactor")

			allErrs = append(allErrs, field.Invalid(fldPath, ask.ReplicationFactor, err.Error()))
		}
		return allErrs
	}

	// If we reached this point, NetworkTopologyStrategy has been chosen
	keyValuePairs := make(map[string]int)

	items := strings.Split(r.Spec.Cassandra.AstarteSystemKeyspace.DataCenterReplication, ",")
	for _, dr := range items {
		dataCenterAndReplication := strings.Split(dr, ":")
		if len(dataCenterAndReplication) != 2 {
			err := errors.New("invalid datacenter replication: wrong format")
			astartelog.Info(err.Error())
			fldPath := field.NewPath("spec").Child("cassandra").Child("astarteSystemKeyspace").Child("dataCenterReplication")
			allErrs = append(allErrs, field.Invalid(fldPath, strings.Join(dataCenterAndReplication, ":"), err.Error()))
		} else {
			// ensure the replication factor is an integer...
			dc := dataCenterAndReplication[0]
			dcReplication, err := strconv.Atoi(dataCenterAndReplication[1])

			if err != nil {
				astartelog.Info(fmt.Sprint("invalid datacenter replication: replication must be an integer", err.Error()))
				fldPath := field.NewPath("spec").Child("cassandra").Child("astarteSystemKeyspace").Child("dataCenterReplication")

				allErrs = append(allErrs, field.Invalid(fldPath, strings.Join(dataCenterAndReplication, ":"), err.Error()))

			} else {
				// populate the keyValuePairs map only after we validated the user input
				keyValuePairs[dc] = dcReplication
			}

			// ...and it's odd
			if dcReplication%2 == 0 {
				err := errors.New("invalid datacenter replication: replication must be odd")
				astartelog.Info(err.Error())
				fldPath := field.NewPath("spec").Child("cassandra").Child("astarteSystemKeyspace").Child("dataCenterReplication")

				allErrs = append(allErrs, field.Invalid(fldPath, strings.Join(dataCenterAndReplication, ":"), err.Error()))
			}
		}
	}

	// finally, ensure that it can be marshaled into json
	_, err := json.Marshal(keyValuePairs)
	if err != nil {
		userFacingErr := errors.New("invalid datacenter replication format")
		astartelog.Info(fmt.Sprintf("%s: %s", userFacingErr.Error(), err.Error()))
		fldPath := field.NewPath("spec").Child("cassandra").Child("astarteSystemKeyspace").Child("dataCenterReplication")

		allErrs = append(allErrs, field.Invalid(fldPath, ask.DataCenterReplication, err.Error()))
	}

	return allErrs
}

func (r *Astarte) validateUpdateAstarteSystemKeyspace(oldAstarte *Astarte) *field.Error {
	if r.Spec.Cassandra.AstarteSystemKeyspace != oldAstarte.Spec.Cassandra.AstarteSystemKeyspace {
		err := errors.New("Once Astarte is created, the astarteSystemKeyspace cannot be modified")
		fldPath := field.NewPath("spec").Child("cassandra").Child("astarteSystemKeyspace")
		return field.Invalid(fldPath, r.Spec.Cassandra.AstarteSystemKeyspace, err.Error())
	}

	return nil
}

func (r *Astarte) validateCFSSLDefinition() *field.Error {
	if !pointy.BoolValue(r.Spec.CFSSL.Deploy, true) && r.Spec.CFSSL.URL == "" {
		err := errors.New("When not deploying CFSSL, the 'url' must be specified")
		fldPath := field.NewPath("spec").Child("cfssl").Child("url")
		astartelog.Info(err.Error())
		return field.Invalid(fldPath, r.Spec.CFSSL.URL, err.Error())
	}

	return nil
}
