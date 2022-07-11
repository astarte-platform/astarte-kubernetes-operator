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

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/openlyinc/pointy"
)

// log is for logging in this package.
var astartelog = logf.Log.WithName("astarte-resource")
var avilog = logf.Log.WithName("astartevoyageringress-resource")
var flowlog = logf.Log.WithName("flow-resource")

func (r *Astarte) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

func (r *AstarteVoyagerIngress) SetupWebhookWithManager(mgr ctrl.Manager) error {
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

// +kubebuilder:webhook:path=/mutate-api-astarte-platform-org-v1alpha1-astarte,mutating=true,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=astartes,verbs=create;update,versions=v1alpha1,name=mastarte.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Astarte{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Astarte) Default() {
	astartelog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-api-astarte-platform-org-v1alpha1-astarte,mutating=false,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=astartes,versions=v1alpha1,name=vastarte.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Astarte{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Astarte) ValidateCreate() error {
	astartelog.Info("validate create", "name", r.Name)

	return r.validateAstarte()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Astarte) ValidateUpdate(old runtime.Object) error {
	astartelog.Info("validate update", "name", r.Name)

	return r.validateAstarte()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Astarte) ValidateDelete() error {
	astartelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *Astarte) validateAstarte() error {
	if pointy.BoolValue(r.Spec.VerneMQ.SSLListener, false) {
		// check that SSLListenerCertSecretName is set
		if r.Spec.VerneMQ.SSLListenerCertSecretName == "" {
			err := errors.New("sslListenerCertSecretName not set")
			astartelog.Error(err, "When deploying Astarte, if SSLListener is set to true you must provide also SSLListenerCertSecretName.")
			return err
		}

		var theConfig *rest.Config
		var c client.Client
		var err error

		if theConfig, err = config.GetConfig(); err != nil {
			return err
		}

		if c, err = client.New(theConfig, client.Options{}); err != nil {
			return err
		}

		// ensure the TLS secret is present
		theSecret := &v1.Secret{}
		if err := c.Get(context.Background(), types.NamespacedName{Name: r.Spec.VerneMQ.SSLListenerCertSecretName, Namespace: r.Namespace}, theSecret); err != nil {
			astartelog.Error(err, "SSLListenerCertSecretName references a secret which is not present. Ensure to create the TLS secret in the namespace in which Astarte resides before applying the new configuration.")
			return err
		}
	}

	if err := validatePodLabelsForClusteredResources(r); err != nil {
		return err
	}

	return nil
}

func validatePodLabelsForClusteredResources(r *Astarte) error {
	resources := []PodLabelsGetter{r.Spec.VerneMQ.AstarteGenericClusteredResource,
		r.Spec.Cassandra.AstarteGenericClusteredResource, r.Spec.RabbitMQ.AstarteGenericClusteredResource,
		r.Spec.Components.Flow.AstarteGenericClusteredResource, r.Spec.Components.Housekeeping.Backend,
		r.Spec.Components.Housekeeping.API.AstarteGenericClusteredResource, r.Spec.Components.RealmManagement.Backend,
		r.Spec.Components.RealmManagement.API.AstarteGenericClusteredResource, r.Spec.Components.Pairing.Backend,
		r.Spec.Components.Pairing.API.AstarteGenericClusteredResource, r.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource,
		r.Spec.Components.TriggerEngine.AstarteGenericClusteredResource, r.Spec.Components.Dashboard.AstarteGenericClusteredResource, r.Spec.CFSSL}
	for _, v := range resources {
		if err := validatePodLabelsForClusteredResource(v); err != nil {
			return err
		}
	}

	return nil

}

func validatePodLabelsForClusteredResource(r PodLabelsGetter) error {
	for k := range r.GetPodLabels() {
		if k == "component" || k == "app" || strings.HasPrefix(k, "astarte-") || strings.HasPrefix(k, "flow-") {
			return fmt.Errorf("Invalid label key %s: can't be any of 'app', 'component', 'astarte-*', 'flow-*'", k)
		}
	}
	return nil
}

// +kubebuilder:webhook:path=/mutate-api-astarte-platform-org-v1alpha1-astartevoyageringress,mutating=true,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=astartevoyageringresses,verbs=create;update,versions=v1alpha1,name=mastartevoyageringress.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &AstarteVoyagerIngress{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AstarteVoyagerIngress) Default() {
	avilog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-api-astarte-platform-org-v1alpha1-astartevoyageringress,mutating=false,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=astartevoyageringresses,versions=v1alpha1,name=vastartevoyageringress.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &AstarteVoyagerIngress{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AstarteVoyagerIngress) ValidateCreate() error {
	avilog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AstarteVoyagerIngress) ValidateUpdate(old runtime.Object) error {
	avilog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AstarteVoyagerIngress) ValidateDelete() error {
	avilog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

// +kubebuilder:webhook:path=/mutate-api-astarte-platform-org-v1alpha1-flow,mutating=true,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=flows,verbs=create;update,versions=v1alpha1,name=mflow.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Flow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Flow) Default() {
	flowlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-api-astarte-platform-org-v1alpha1-flow,mutating=false,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=flows,versions=v1alpha1,name=vflow.kb.io,admissionReviewVersions=v1

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
