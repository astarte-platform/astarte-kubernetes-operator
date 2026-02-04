/*
Copyright 2026 The Kubernetes authors.

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
// +kubebuilder:docs-gen:collapse=Apache License

package v2alpha1

import (
	"context"
	"fmt"

	pointy "go.openly.dev/pointy"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"

	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	adiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v2alpha1"
	v1 "k8s.io/api/core/v1"
	types "k8s.io/apimachinery/pkg/types"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
Next, we'll setup a logger for the webhooks.
*/

// log is for logging in this package.
var (
	adilog = logf.Log.WithName("astartedefaultingress-resource")
	c      client.Client
)

/*
Then, we set up the webhook with the manager.
*/

// SetupAstarteDefaultIngressWebhookWithManager registers the webhook for ADI in the manager.
func SetupAstarteDefaultIngressWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(&adiv2alpha1.AstarteDefaultIngress{}).
		WithValidator(&AstarteDefaultIngressCustomValidator{}).
		WithDefaulter(&AstarteDefaultIngressCustomDefaulter{}).
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

// +kubebuilder:webhook:path=/mutate-ingress-astarte-platform-org-v2alpha1-astartedefaultingress,mutating=true,failurePolicy=fail,sideEffects=None,groups=ingress.astarte-platform.org,resources=astartedefaultingresses,verbs=create;update,versions=v2alpha1,name=mastartedefaultingress.kb.io,admissionReviewVersions=v1

// AstarteDefaultIngressCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind AstarteDefaultIngress when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type AstarteDefaultIngressCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &AstarteDefaultIngressCustomDefaulter{}

/*
We use the `webhook.CustomDefaulter`interface to set defaults to our CRD.
A webhook will automatically be served that calls this defaulting.

The `Default`method is expected to mutate the receiver, setting the defaults.
*/

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind AstarteDefaultIngress.
func (d *AstarteDefaultIngressCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	adi, ok := obj.(*adiv2alpha1.AstarteDefaultIngress)
	if !ok {
		return fmt.Errorf("expected a AstarteDefaultIngress resource")
	}

	adilog.Info("Defaulting for AstarteDefaultIngress", "name", adi.GetName())

	// Set default values
	d.applyDefaults(adi)
	return nil
}

// applyDefaults applies default values to AstarteDefaultIngress fields.
func (d *AstarteDefaultIngressCustomDefaulter) applyDefaults(adi *adiv2alpha1.AstarteDefaultIngress) {
	// Set default Ingress Controller annotation if not set
	if _, ok := adi.GetAnnotations()[adiv2alpha1.AnnotationIngressControllerSelector]; !ok {
		adi.GetAnnotations()[adiv2alpha1.AnnotationIngressControllerSelector] = adiv2alpha1.HAProxySelectorValue
	}

	// Set default IngressClass if not set based on Ingress Controller selection
	if adi.Spec.IngressClass == "" {
		adi.Spec.IngressClass = "haproxy"
		if !adi.HAProxyIngressControllerSelected() {
			adi.Spec.IngressClass = "nginx"
		}
	}
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
// +kubebuilder:webhook:path=/validate-ingress-astarte-platform-org-v2alpha1-astartedefaultingress,mutating=false,failurePolicy=fail,sideEffects=None,groups=ingress.astarte-platform.org,resources=astartedefaultingresses,verbs=create;update,versions=v2alpha1,name=vastartedefaultingress.kb.io,admissionReviewVersions=v1

// AstarteDefaultIngressCustomValidator struct is responsible for validating the AstarteDefaultIngress resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type AstarteDefaultIngressCustomValidator struct{}

var _ webhook.CustomValidator = &AstarteDefaultIngressCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type AstarteDefaultIngress.
func (v *AstarteDefaultIngressCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {

	adi, ok := obj.(*adiv2alpha1.AstarteDefaultIngress)
	if !ok {
		return nil, fmt.Errorf("expected a AstarteDefaultIngress resource")
	}

	adilog.Info("Validation for AstarteDefaultIngress upon creation", "name", adi.GetName())

	return nil, validateAstarteDefaultIngress(adi)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type AstarteDefaultIngress.
func (v *AstarteDefaultIngressCustomValidator) ValidateUpdate(_ context.Context, objOld, objNew runtime.Object) (admission.Warnings, error) {

	newAdi, ok := objNew.(*adiv2alpha1.AstarteDefaultIngress)
	if !ok {
		return nil, fmt.Errorf("expected a AstarteDefaultIngress resource")
	}

	_, ok = objOld.(*adiv2alpha1.AstarteDefaultIngress)
	if !ok {
		return nil, fmt.Errorf("expected a AstarteDefaultIngress resource")
	}

	adilog.Info("Validation for AstarteDefaultIngress upon update", "name", newAdi.GetName())

	return nil, validateAstarteDefaultIngress(newAdi)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type AstarteDefaultIngress.
func (v *AstarteDefaultIngressCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {

	adi, ok := obj.(*adiv2alpha1.AstarteDefaultIngress)
	if !ok {
		return nil, fmt.Errorf("expected a AstarteDefaultIngress resource")
	}

	adilog.Info("Validation for AstarteDefaultIngress upon deletion", "name", adi.GetName())

	return nil, nil
}

// +kubebuilder:docs-gen:collapse=validateAstarteDefaultIngress() Code Implementation
func validateAstarteDefaultIngress(adi *adiv2alpha1.AstarteDefaultIngress) error {
	allErrors := field.ErrorList{}

	astarte, astarteFoundErr := validateReferencedAstarte(adi, c)
	if astarteFoundErr != nil {
		allErrors = append(allErrors, astarteFoundErr)
	}
	// if astarte is not found, do not check the api and broker config as they are not available anyway
	if astarteFoundErr == nil {
		if err := validateBrokerTLSConfig(adi, astarte); err != nil {
			allErrors = append(allErrors, err)
		}
		if err := validateAPITLSConfig(adi, astarte); err != nil {
			allErrors = append(allErrors, err)
		}
	}
	if err := validateDashboardTLSConfig(adi); err != nil {
		allErrors = append(allErrors, err)
	}

	allErrors = append(allErrors, validateTLSSecretExistence(adi, c)...)

	// Validate Ingress Controller selector annotation
	if err := validateIngressControllerSelectorAnnotation(adi); err != nil {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) == 0 {
		return nil
	}
	return apierrors.NewInvalid(
		schema.GroupKind{Group: "ingress", Kind: "AstarteDefaultIngress"},
		adi.Name,
		allErrors,
	)
}

func validateReferencedAstarte(adi *adiv2alpha1.AstarteDefaultIngress, c client.Client) (*apiv2alpha1.Astarte, *field.Error) {
	fldPath := field.NewPath("spec").Child("astarte")

	// ensure that the referenced Astarte instance exists
	theAstarte := &apiv2alpha1.Astarte{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: adi.Spec.Astarte, Namespace: adi.Namespace}, theAstarte); err != nil {
		adilog.Error(err, "Could not find the referenced Astarte.")
		return nil, field.NotFound(fldPath, adi.Spec.Astarte)
	}
	return theAstarte, nil
}

func validateBrokerTLSConfig(adi *adiv2alpha1.AstarteDefaultIngress, astarte *apiv2alpha1.Astarte) *field.Error {
	if !pointy.BoolValue(astarte.Spec.VerneMQ.SSLListener, false) && pointy.BoolValue(adi.Spec.Broker.Deploy, true) {
		fldPath := field.NewPath("spec").Child("broker").Child("deploy")
		return field.Invalid(fldPath, astarte.Spec.VerneMQ.SSLListenerCertSecretName,
			"When deploying Broker Ingress, VerneMQ SSLListener must be enabled in the main Astarte resource.")
	}

	return nil
}

func validateDashboardTLSConfig(adi *adiv2alpha1.AstarteDefaultIngress) *field.Error {
	if adi.Spec.TLSSecret == "" && pointy.BoolValue(adi.Spec.Dashboard.SSL, true) &&
		pointy.BoolValue(adi.Spec.Dashboard.Deploy, true) && adi.Spec.Dashboard.TLSSecret == "" {
		fldPath := field.NewPath("spec").Child("dashboard").Child("tlsSecret")
		return field.Required(fldPath, "Requested SSL support for Dashboard, but no TLS Secret provided")
	}
	return nil
}

func validateAPITLSConfig(adi *adiv2alpha1.AstarteDefaultIngress, astarte *apiv2alpha1.Astarte) *field.Error {
	if pointy.BoolValue(astarte.Spec.API.SSL, true) && adi.Spec.TLSSecret == "" &&
		adi.Spec.API.TLSSecret == "" && pointy.BoolValue(adi.Spec.API.Deploy, true) {
		fldPath := field.NewPath("spec").Child("api").Child("tlsSecret")
		return field.Required(fldPath, "Requested SSL support for API, but no TLS Secret provided")
	}
	return nil
}

func validateTLSSecretExistence(adi *adiv2alpha1.AstarteDefaultIngress, c client.Client) field.ErrorList {
	allErrs := field.ErrorList{}

	if adi.Spec.TLSSecret != "" && (adi.Spec.API.TLSSecret == "" || (pointy.BoolValue(adi.Spec.Dashboard.SSL, true) &&
		pointy.BoolValue(adi.Spec.Dashboard.Deploy, true) && adi.Spec.Dashboard.TLSSecret == "")) {

		if err := getSecret(c, adi.Spec.TLSSecret, adi.Namespace, field.NewPath("spec").Child("tlsSecret")); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if adi.Spec.API.TLSSecret != "" {
		if err := getSecret(c, adi.Spec.API.TLSSecret, adi.Namespace, field.NewPath("spec").Child("api").Child("tlsSecret")); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if pointy.BoolValue(adi.Spec.Dashboard.SSL, true) && pointy.BoolValue(adi.Spec.Dashboard.Deploy, true) && adi.Spec.Dashboard.TLSSecret != "" {
		if err := getSecret(c, adi.Spec.Dashboard.TLSSecret, adi.Namespace, field.NewPath("spec").Child("dashboard").Child("tlsSecret")); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	return allErrs
}

func getSecret(c client.Client, secretName string, namespace string, fldPath *field.Path) *field.Error {
	theSecret := &v1.Secret{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: namespace}, theSecret); err != nil {
		adilog.Error(err, fmt.Sprintf("The secret %s does not exist in namespace %s.", secretName, namespace))
		return field.NotFound(fldPath, secretName)
	}
	return nil
}

// validateIngressControllerSelectorAnnotation checks if the Ingress Controller selector annotation is valid
func validateIngressControllerSelectorAnnotation(adi *adiv2alpha1.AstarteDefaultIngress) *field.Error {
	if adi.GetAnnotations() != nil {
		if ingressSelector, ok := adi.GetAnnotations()[adiv2alpha1.AnnotationIngressControllerSelector]; ok {
			if ingressSelector != adiv2alpha1.NGINXSelectorValue && ingressSelector != adiv2alpha1.HAProxySelectorValue {
				fldPath := field.NewPath("metadata").Child("annotations").Key(adiv2alpha1.AnnotationIngressControllerSelector)
				return field.Invalid(fldPath, ingressSelector, "Unsupported Ingress Controller selector. Supported values are 'nginx.ingress.kubernetes.io' and 'haproxy.org'.")
			}
		}
	}
	return nil
}
