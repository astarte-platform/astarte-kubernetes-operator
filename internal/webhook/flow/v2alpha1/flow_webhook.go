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

	"k8s.io/apimachinery/pkg/runtime"

	flowv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/flow/v2alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
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
	adilog = logf.Log.WithName("flow-resource")
	// c      client.Client
)

/*
Then, we set up the webhook with the manager.
*/

// SetupAstarteFlowWebhookWithManager registers the webhook for ADI in the manager.
func SetupAstarteFlowWebhookWithManager(mgr ctrl.Manager) error {
	// c = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(&flowv2alpha1.Flow{}).
		WithValidator(&AstarteFlowCustomValidator{}).
		WithDefaulter(&AstarteFlowCustomDefaulter{}).
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

// +kubebuilder:webhook:path=/mutate-flow-astarte-platform-org-v2alpha1-flow,mutating=true,failurePolicy=fail,sideEffects=None,groups=flow.astarte-platform.org,resources=flows,verbs=create;update,versions=v2alpha1,name=mflow.kb.io,admissionReviewVersions=v1

// AstarteFlowCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind AstarteFlow when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type AstarteFlowCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &AstarteFlowCustomDefaulter{}

/*
We use the `webhook.CustomDefaulter`interface to set defaults to our CRD.
A webhook will automatically be served that calls this defaulting.

The `Default`method is expected to mutate the receiver, setting the defaults.
*/

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind AstarteFlow.
func (d *AstarteFlowCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	f, ok := obj.(*flowv2alpha1.Flow)
	if !ok {
		return fmt.Errorf("expected a Flow resource")
	}

	adilog.Info("Defaulting for Flow", "name", f.GetName())

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
// +kubebuilder:webhook:path=/validate-flow-astarte-platform-org-v2alpha1-flow,mutating=false,failurePolicy=fail,sideEffects=None,groups=flow.astarte-platform.org,resources=flows,verbs=create;update,versions=v2alpha1,name=vflow.kb.io,admissionReviewVersions=v1

// AstarteFlowCustomValidator struct is responsible for validating the AstarteFlow resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type AstarteFlowCustomValidator struct{}

var _ webhook.CustomValidator = &AstarteFlowCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type AstarteFlow.
func (v *AstarteFlowCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {

	flow, ok := obj.(*flowv2alpha1.Flow)
	if !ok {
		return nil, fmt.Errorf("expected a AstarteFlow resource")
	}

	adilog.Info("Validation for AstarteFlow upon creation", "name", flow.GetName())

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type AstarteFlow.
func (v *AstarteFlowCustomValidator) ValidateUpdate(_ context.Context, objOld, objNew runtime.Object) (admission.Warnings, error) {

	newFlow, ok := objNew.(*flowv2alpha1.Flow)
	if !ok {
		return nil, fmt.Errorf("expected a AstarteFlow resource")
	}

	_, ok = objOld.(*flowv2alpha1.Flow)
	if !ok {
		return nil, fmt.Errorf("expected a AstarteFlow resource")
	}

	adilog.Info("Validation for AstarteFlow upon update", "name", newFlow.GetName())

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type AstarteFlow.
func (v *AstarteFlowCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {

	flow, ok := obj.(*flowv2alpha1.Flow)
	if !ok {
		return nil, fmt.Errorf("expected a AstarteFlow resource")
	}

	adilog.Info("Validation for AstarteFlow upon deletion", "name", flow.GetName())

	return nil, nil
}
