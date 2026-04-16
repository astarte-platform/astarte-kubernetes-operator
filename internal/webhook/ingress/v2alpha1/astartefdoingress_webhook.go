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

package v2alpha1

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	ingressv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v2alpha1"
)

var (
	fdolog = logf.Log.WithName("astartefdoingress-resource")
)

// SetupAstarteFDOIngressWebhookWithManager registers the webhook for AstarteFDOIngress in the manager.
func SetupAstarteFDOIngressWebhookWithManager(mgr ctrl.Manager) error {
	c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(&ingressv2alpha1.AstarteFDOIngress{}).
		WithValidator(&AstarteFDOIngressCustomValidator{}).
		WithDefaulter(&AstarteFDOIngressCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-ingress-astarte-platform-org-v2alpha1-astartefdoingress,mutating=true,failurePolicy=fail,sideEffects=None,groups=ingress.astarte-platform.org,resources=astartefdoingresses,verbs=create;update,versions=v2alpha1,name=mastartefdoingress.kb.io,admissionReviewVersions=v1

// AstarteFDOIngressCustomDefaulter sets default values on AstarteFDOIngress resources.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type AstarteFDOIngressCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &AstarteFDOIngressCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind AstarteFDOIngress.
func (d *AstarteFDOIngressCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	return nil
}

// +kubebuilder:webhook:path=/validate-ingress-astarte-platform-org-v2alpha1-astartefdoingress,mutating=false,failurePolicy=fail,sideEffects=None,groups=ingress.astarte-platform.org,resources=astartefdoingresses,verbs=create;update,versions=v2alpha1,name=vastartefdoingress.kb.io,admissionReviewVersions=v1

// AstarteFDOIngressCustomValidator validates AstarteFDOIngress resources.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type AstarteFDOIngressCustomValidator struct{}

var _ webhook.CustomValidator = &AstarteFDOIngressCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type AstarteFDOIngress.
func (v *AstarteFDOIngressCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	fdoIngress, ok := obj.(*ingressv2alpha1.AstarteFDOIngress)
	if !ok {
		return nil, fmt.Errorf("expected an AstarteFDOIngress resource")
	}

	fdolog.Info("Validation for AstarteFDOIngress upon creation", "name", fdoIngress.GetName())

	return nil, validateAstarteFDOIngress(fdoIngress)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type AstarteFDOIngress.
func (v *AstarteFDOIngressCustomValidator) ValidateUpdate(_ context.Context, objOld, objNew runtime.Object) (admission.Warnings, error) {
	fdoIngress, ok := objNew.(*ingressv2alpha1.AstarteFDOIngress)
	if !ok {
		return nil, fmt.Errorf("expected an AstarteFDOIngress resource")
	}

	if _, ok := objOld.(*ingressv2alpha1.AstarteFDOIngress); !ok {
		return nil, fmt.Errorf("expected an AstarteFDOIngress resource")
	}

	fdolog.Info("Validation for AstarteFDOIngress upon update", "name", fdoIngress.GetName())

	return nil, validateAstarteFDOIngress(fdoIngress)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type AstarteFDOIngress.
func (v *AstarteFDOIngressCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	fdoIngress, ok := obj.(*ingressv2alpha1.AstarteFDOIngress)
	if !ok {
		return nil, fmt.Errorf("expected an AstarteFDOIngress resource")
	}

	fdolog.Info("Validation for AstarteFDOIngress upon deletion", "name", fdoIngress.GetName())

	return nil, nil
}

func validateAstarteFDOIngress(fdoIngress *ingressv2alpha1.AstarteFDOIngress) error {
	allErrors := field.ErrorList{}

	if fdoIngress.Spec.Astarte == "" {
		allErrors = append(allErrors, field.Required(field.NewPath("spec").Child("astarte"), "must reference an Astarte instance"))
	}

	_, astarteFoundErr := validateFDOReferencedAstarte(fdoIngress)
	if astarteFoundErr != nil {
		allErrors = append(allErrors, astarteFoundErr)
	}

	allErrors = append(allErrors, validateFDOIngressTLSSecretExistence(fdoIngress)...)

	if len(allErrors) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "ingress", Kind: "AstarteFDOIngress"},
		fdoIngress.Name,
		allErrors,
	)
}

func validateFDOReferencedAstarte(fdoIngress *ingressv2alpha1.AstarteFDOIngress) (*apiv2alpha1.Astarte, *field.Error) {
	fldPath := field.NewPath("spec").Child("astarte")

	theAstarte := &apiv2alpha1.Astarte{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: fdoIngress.Spec.Astarte, Namespace: fdoIngress.Namespace}, theAstarte); err != nil {
		fdolog.Error(err, "Could not find the referenced Astarte.")
		return nil, field.NotFound(fldPath, fdoIngress.Spec.Astarte)
	}

	return theAstarte, nil
}

func validateFDOIngressTLSSecretExistence(fdoIngress *ingressv2alpha1.AstarteFDOIngress) field.ErrorList {
	if fdoIngress.Spec.TLSSecret == "" {
		return nil
	}

	allErrs := field.ErrorList{}
	if err := getSecret(c, fdoIngress.Spec.TLSSecret, fdoIngress.Namespace, field.NewPath("spec").Child("tlsSecret")); err != nil {
		allErrs = append(allErrs, err)
	}

	return allErrs
}
