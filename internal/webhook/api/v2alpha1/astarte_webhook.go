/*
Copyright 2025.

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

//nolint:goconst
package v2alpha1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
Next, we'll setup a logger for the webhooks.
*/

// log is for logging in this package.
var (
	astartelog = logf.Log.WithName("astarte-resource")
	c          client.Client
)

// SetupAstarteWebhookWithManager registers the webhook for Memcached in the manager.
func SetupAstarteWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(&apiv2alpha1.Astarte{}).
		WithValidator(&AstarteCustomValidator{}).
		WithDefaulter(&AstarteCustomDefaulter{}).
		Complete()
}

/*
Notice that we use kubebuilder markers to generate webhook manifests.
This marker is responsible for generating a mutating webhook manifest.

The meaning of each marker can be found [here](/reference/markers/webhook.md).
*/

/*
This marker is responsible for generating a mutation webhook manifest.
*/

// +kubebuilder:webhook:path=/mutate-api-astarte-platform-org-v2alpha1-astarte,mutating=true,failurePolicy=fail,sideEffects=None,groups=api.astarte-platform.org,resources=astartes,verbs=create;update,versions=v2alpha1,name=mastarte.kb.io,admissionReviewVersions=v1

// AstarteCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Astarte when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type AstarteCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &AstarteCustomDefaulter{}

/*
We use the `webhook.CustomDefaulter`interface to set defaults to our CRD.
A webhook will automatically be served that calls this defaulting.

The `Default`method is expected to mutate the receiver, setting the defaults.
*/

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Astarte.
func (d *AstarteCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	astarte, ok := obj.(*apiv2alpha1.Astarte)

	if !ok {
		return errors.New("expected a Astarte resource")
	}

	astartelog.Info("Defaulting for Astarte", "name", astarte.GetName())
	return nil
}

/*
We can validate our CRD beyond what's possible with declarative
validation. Generally, declarative validation should be sufficient, but
sometimes more advanced use cases call for complex validation.

If `webhook.CustomValidator` interface is implemented, a webhook will automatically be
served that calls the validation.

The `ValidateCreate`, `ValidateUpdate` and `ValidateDelete` methods are expected
to validate its receiver upon creation, update and deletion respectively.
We separate out ValidateCreate from ValidateUpdate to allow behavior like making
certain fields immutable, so that they can only be set on creation.
ValidateDelete is also separated from ValidateUpdate to allow different
validation behavior on deletion.
Here, however, we just use the same shared validation for `ValidateCreate` and
`ValidateUpdate`. And we do nothing in `ValidateDelete`, since we don't need to
validate anything on deletion.
*/

/*
This marker is responsible for generating a validation webhook manifest.
*/

// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-api-astarte-platform-org-v2alpha1-astarte,mutating=false,failurePolicy=fail,sideEffects=None,groups=api.astarte-platform.org,resources=astartes,verbs=create;update,versions=v2alpha1,name=vastarte.kb.io,admissionReviewVersions=v1

// AstarteCustomValidator struct is responsible for validating the Astarte resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type AstarteCustomValidator struct{}

var _ webhook.CustomValidator = &AstarteCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Astarte.
func (v *AstarteCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {

	r, ok := obj.(*apiv2alpha1.Astarte)
	if !ok {
		return nil, errors.New("expected a Astarte resource")
	}

	astartelog.Info("Validation for Astarte upon creation", "name", r.GetName())

	allErrs := field.ErrorList{}

	if err := validateCreateAstarteInstanceID(r); err != nil {
		allErrs = append(allErrs, err)
	}

	if errList := validateCreateAstarteSystemKeyspace(r); errList != nil {
		allErrs = append(allErrs, errList...)
	}

	if errList := validateAstarte(r); len(errList) > 0 {
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

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Astarte.
func (v *AstarteCustomValidator) ValidateUpdate(_ context.Context, objOld, objNew runtime.Object) (admission.Warnings, error) {
	allErrs := field.ErrorList{}

	newAstarte, ok := objNew.(*apiv2alpha1.Astarte)
	if !ok {
		return nil, errors.New("expected a Astarte resource")
	}

	oldAstarte, ok := objOld.(*apiv2alpha1.Astarte)
	if !ok {
		return nil, errors.New("expected a Astarte resource")
	}

	astartelog.Info("validate update", "name", newAstarte.Name)

	if err := validateUpdateAstarteInstanceID(newAstarte, oldAstarte); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateUpdateAstarteSystemKeyspace(newAstarte, oldAstarte); err != nil {
		allErrs = append(allErrs, err)
	}

	if errList := validateAstarte(newAstarte); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		schema.GroupKind{Group: "api", Kind: "Astarte"},
		newAstarte.Name,
		allErrs,
	)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Astarte.
func (v *AstarteCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*apiv2alpha1.Astarte)
	if !ok {
		return nil, errors.New("expected a Astarte resource")
	}

	astartelog.Info("Validation for Astarte upon deletion", "name", r.GetName())

	return nil, nil
}

func validateAstarte(r *apiv2alpha1.Astarte) field.ErrorList {
	allErrs := field.ErrorList{}

	if errList := validateSSLListener(r); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if err := validateAstartePriorityClasses(r); err != nil {
		allErrs = append(allErrs, err)
	}

	if errList := validatePodLabelsForClusteredResources(r); len(errList) > 0 {
		allErrs = append(allErrs, errList...)
	}

	if err := validateAutoscalerForClusteredResources(r); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateCFSSLDefinition(r); err != nil {
		allErrs = append(allErrs, err)
	}

	return allErrs
}

func validateUpdateAstarteInstanceID(r *apiv2alpha1.Astarte, oldAstarte *apiv2alpha1.Astarte) *field.Error {
	if r.Spec.AstarteInstanceID != oldAstarte.Spec.AstarteInstanceID {
		fldPath := field.NewPath("spec").Child("astarteInstanceID")
		err := errors.New("the astarteInstanceId cannot be updated since it is immutable for your Astarte instance")

		astartelog.Info(err.Error(), "astarteInstanceID", r.Spec.AstarteInstanceID)
		return field.Invalid(fldPath, r.Spec.AstarteInstanceID, err.Error())
	}

	return nil
}

func validateCreateAstarteInstanceID(r *apiv2alpha1.Astarte) *field.Error {
	astarteList := &apiv2alpha1.AstarteList{}
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

func validateSSLListener(r *apiv2alpha1.Astarte) field.ErrorList {
	allErrs := field.ErrorList{}

	// Only validate if the SSL Listener is explicitly enabled.
	if pointy.BoolValue(r.Spec.VerneMQ.SSLListener, false) {
		fldPath := field.NewPath("spec").Child("vernemq").Child("sslListenerCertSecretName")
		secretName := r.Spec.VerneMQ.SSLListenerCertSecretName

		// First, check that SSLListenerCertSecretName is set.
		if secretName == "" {
			err := errors.New("must be set when sslListener is true")
			astartelog.Info(err.Error())
			allErrs = append(allErrs, field.Invalid(fldPath, secretName, err.Error()))
		} else {
			// If the name is set, then ensure the Secret resource exists.
			secret := &v1.Secret{}
			if err := c.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: r.Namespace}, secret); err != nil {
				astartelog.Info(err.Error())
				allErrs = append(allErrs, field.NotFound(fldPath, secretName))
			}
		}
	}
	return allErrs
}

func validatePodLabelsForClusteredResources(r *apiv2alpha1.Astarte) field.ErrorList {
	allErrs := field.ErrorList{}

	resources := []apiv2alpha1.PodLabelsGetter{r.Spec.VerneMQ.AstarteGenericClusteredResource,
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

func validatePodLabelsForClusteredResource(r apiv2alpha1.PodLabelsGetter) field.ErrorList {
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

func validateAutoscalerForClusteredResources(r *apiv2alpha1.Astarte) *field.Error {
	// We have no constraints on autoscaling except for these components
	excludedResources := []apiv2alpha1.AstarteGenericClusteredResource{
		r.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource,
	}

	return validateAutoscalerForClusteredResourcesExcluding(r, excludedResources)
}

func validateAutoscalerForClusteredResourcesExcluding(r *apiv2alpha1.Astarte, excluded []apiv2alpha1.AstarteGenericClusteredResource) *field.Error {
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

func validateAstartePriorityClasses(r *apiv2alpha1.Astarte) *field.Error {
	if r.Spec.Features.AstartePodPriorities.IsEnabled() {
		return validatePriorityClassesValues(r)
	}

	return nil
}

func validatePriorityClassesValues(r *apiv2alpha1.Astarte) *field.Error {
	// default values guarantee pointers are not nil
	highPriorityValue := *r.Spec.Features.AstartePodPriorities.AstarteHighPriority
	midPriorityValue := *r.Spec.Features.AstartePodPriorities.AstarteMidPriority
	lowPriorityValue := *r.Spec.Features.AstartePodPriorities.AstarteLowPriority

	if midPriorityValue >= highPriorityValue || lowPriorityValue >= midPriorityValue {
		err := errors.New("Astarte PriorityClass values are incoherent")
		astartelog.Info(err.Error())
		fldPath := field.NewPath("spec").Child("features").Child("astarte{Low|Medium|High}Priority")

		return field.Invalid(fldPath, "", err.Error())
	}
	return nil
}

func validateCreateAstarteSystemKeyspace(r *apiv2alpha1.Astarte) field.ErrorList {
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

func validateUpdateAstarteSystemKeyspace(r *apiv2alpha1.Astarte, oldAstarte *apiv2alpha1.Astarte) *field.Error {
	if r.Spec.Cassandra.AstarteSystemKeyspace != oldAstarte.Spec.Cassandra.AstarteSystemKeyspace {
		err := errors.New("Once Astarte is created, the astarteSystemKeyspace cannot be modified")
		fldPath := field.NewPath("spec").Child("cassandra").Child("astarteSystemKeyspace")
		return field.Invalid(fldPath, r.Spec.Cassandra.AstarteSystemKeyspace, err.Error())
	}

	return nil
}

func validateCFSSLDefinition(r *apiv2alpha1.Astarte) *field.Error {
	if pointy.BoolValue(r.Spec.CFSSL.Deploy, true) {
		return nil
	}

	// If we are here, CFSSL is not being deployed. Ensure URL is set.
	if r.Spec.CFSSL.URL == "" {
		err := errors.New("When not deploying CFSSL, the 'url' must be specified")
		fldPath := field.NewPath("spec").Child("cfssl").Child("url")
		astartelog.Info(err.Error())
		return field.Invalid(fldPath, r.Spec.CFSSL.URL, err.Error())
	}

	// If URL is set, ensure it is compliant with RFC 3986
	_, err := url.Parse(r.Spec.CFSSL.URL)
	if err != nil {
		err := errors.New("The provided URL is not valid")
		fldPath := field.NewPath("spec").Child("cfssl").Child("url")
		astartelog.Info(err.Error())
		return field.Invalid(fldPath, r.Spec.CFSSL.URL, err.Error())
	}

	return nil
}
