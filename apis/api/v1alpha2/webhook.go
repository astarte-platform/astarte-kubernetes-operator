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

package v1alpha2

import (
	"context"
	"errors"
	"strings"

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

	"github.com/openlyinc/pointy"
)

// log is for logging in this package.
var (
	astartelog = logf.Log.WithName("astarte-resource")
	flowlog    = logf.Log.WithName("flow-resource")
	c          client.Client
)

func (r *Astarte) SetupWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

func (r *Flow) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO: Right now, our tools do *not* support AdmissionReview v1. As such, we're forcing only v1beta1. This needs to change to support v1 or v1;v1beta1 as soon
// as controller-runtime does to be future proof when AdmissionReview v1beta1 will be removed from future Kubernetes versions.

// +kubebuilder:webhook:path=/mutate-api-astarte-platform-org-v1alpha2-astarte,mutating=true,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=astartes,verbs=create;update,versions=v1alpha2,name=mastarte.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Astarte{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Astarte) Default() {
	astartelog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-api-astarte-platform-org-v1alpha2-astarte,mutating=false,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=astartes,versions=v1alpha2,name=vastarte.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Astarte{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Astarte) ValidateCreate() error {
	allErrs := field.ErrorList{}

	astartelog.Info("validate create", "name", r.Name)

	if err := r.validateCreateAstarteInstanceID(); err != nil {
		allErrs = append(allErrs, err)
	}

	if errList := r.validateAstarte(); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "api", Kind: "Astarte"},
		r.Name,
		allErrs,
	)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Astarte) ValidateUpdate(old runtime.Object) error {
	allErrs := field.ErrorList{}

	astartelog.Info("validate update", "name", r.Name)

	oldAstarte, _ := old.(*Astarte)
	if err := r.validateUpdateAstarteInstanceID(oldAstarte); err != nil {
		allErrs = append(allErrs, err)
	}

	if errList := r.validateAstarte(); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "api", Kind: "Astarte"},
		r.Name,
		allErrs,
	)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Astarte) ValidateDelete() error {
	astartelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *Astarte) validateCreateAstarteInstanceID() *field.Error {
	astarteList := &AstarteList{}
	if clientErr := c.List(context.Background(), astarteList); clientErr != nil {
		err := errors.New("Cannot list astarte instances in the cluster. Please retry.")
		astartelog.Info(clientErr.Error(), "details", err.Error())
		return field.InternalError(field.NewPath(""), err)
	}

	for _, otherAstarte := range astarteList.Items {
		if r.Spec.AstarteInstanceID == otherAstarte.Spec.AstarteInstanceID {
			fldPath := field.NewPath("spec").Child("astarteInstanceID")
			err := errors.New("Invalid astarteInstanceID: the chosen ID is already in use")

			astartelog.Info(err.Error(), "astarteInstanceID", r.Spec.AstarteInstanceID)
			return field.Invalid(fldPath, r.Spec.AstarteInstanceID, err.Error())
		}
	}

	return nil
}

func (r *Astarte) validateUpdateAstarteInstanceID(oldAstarte *Astarte) *field.Error {
	if r.Spec.AstarteInstanceID != oldAstarte.Spec.AstarteInstanceID {
		fldPath := field.NewPath("spec").Child("astarteInstanceID")
		err := errors.New("The astarteInstanceId cannot be updated since it is immutable for your Astarte instance")

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

	if errList := validatePodLabelsForClusteredResources(r); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if err := validateAutoscalerForClusteredResources(r); err != nil {
		allErrs = append(allErrs, err)
	}

	return allErrs
}

func validatePodLabelsForClusteredResources(r *Astarte) field.ErrorList {
	allErrs := field.ErrorList{}

	resources := []PodLabelsGetter{r.Spec.VerneMQ.AstarteGenericClusteredResource,
		r.Spec.Cassandra.AstarteGenericClusteredResource, r.Spec.RabbitMQ.AstarteGenericClusteredResource,
		r.Spec.Components.Flow.AstarteGenericClusteredResource, r.Spec.Components.Housekeeping.Backend,
		r.Spec.Components.Housekeeping.API.AstarteGenericClusteredResource, r.Spec.Components.RealmManagement.Backend,
		r.Spec.Components.RealmManagement.API.AstarteGenericClusteredResource, r.Spec.Components.Pairing.Backend,
		r.Spec.Components.Pairing.API.AstarteGenericClusteredResource, r.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource,
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
			err := errors.New("Invalid label key: can't be any of 'app', 'component', 'astarte-*', 'flow-*'")
			astartelog.Info(err.Error(), "label", k)

			allErrs = append(allErrs, field.Invalid(fldPath, k, err.Error()))
		}
	}
	return allErrs
}

func validateAutoscalerForClusteredResources(r *Astarte) *field.Error {
	// We have no constraints on autoscaling except for these components
	excludedResources := []AstarteGenericClusteredResource{
		r.Spec.RabbitMQ.AstarteGenericClusteredResource,
		r.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource,
		r.Spec.Cassandra.AstarteGenericClusteredResource,
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

// +kubebuilder:webhook:path=/mutate-api-astarte-platform-org-v1alpha2-flow,mutating=true,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=flows,verbs=create;update,versions=v1alpha2,name=mflow.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Flow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Flow) Default() {
	flowlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-api-astarte-platform-org-v1alpha2-flow,mutating=false,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=flows,versions=v1alpha2,name=vflow.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Flow{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Flow) ValidateCreate() error {
	flowlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Flow) ValidateUpdate(old runtime.Object) error {
	flowlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Flow) ValidateDelete() error {
	flowlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
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
