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

	"github.com/astarte-platform/astarte-kubernetes-operator/apis/api/commontypes"
	"github.com/astarte-platform/astarte-kubernetes-operator/version"

	"github.com/openlyinc/pointy"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
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

// +kubebuilder:webhook:path=/mutate-api-astarte-platform-org-v1alpha1-astarte,mutating=true,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=astartes,verbs=create;update,versions=v1alpha1,name=mastarte.kb.io,admissionReviewVersions=v1beta1

var _ webhook.Defaulter = &Astarte{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Astarte) Default() {
	astartelog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-api-astarte-platform-org-v1alpha1-astarte,mutating=false,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=astartes,versions=v1alpha1,name=vastarte.kb.io,admissionReviewVersions=v1beta1

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
	var theConfig *rest.Config
	var c client.Client
	var err error

	if theConfig, err = config.GetConfig(); err != nil {
		return err
	}

	if c, err = client.New(theConfig, client.Options{}); err != nil {
		return err
	}

	allErrors := field.ErrorList{}

	// Check if we can manage the requested Astarte version or not
	if !version.CanManageVersion(r.Spec.Version) {
		fldPath := field.NewPath("spec").Child("version")
		err := field.NotSupported(fldPath, r.Spec.Version, []string{version.AstarteVersionConstraintString})
		astartelog.Error(err, "Version %s is not supported by this Operator. Supported Astarte versions adhere to these constraints: %s",
			r.Spec.Version, version.AstarteVersionConstraintString)
		allErrors = append(allErrors, err)
	}

	if r.Spec.VerneMQ.SSLListener {
		if err := validateVerneMQSSLListener(r, c); err != nil {
			allErrors = append(allErrors, err)
		}
	}

	if err := validateCassandraDefinition(r.Spec.Cassandra); err != nil {
		allErrors = append(allErrors, err)
	}

	if err := validateCFSSLDefinition(r.Spec.CFSSL); err != nil {
		allErrors = append(allErrors, err)
	}

	if err := validateRabbitMQDefinition(r, c); err != nil {
		allErrors = append(allErrors, err...)
	}

	if len(allErrors) == 0 {
		return nil
	}
	return apierrors.NewInvalid(
		schema.GroupKind{Group: "api", Kind: "Astarte"},
		r.Name,
		allErrors,
	)
}

func validateVerneMQSSLListener(r *Astarte, c client.Client) *field.Error {
	// check that SSLListenerCertSecretName is set
	if r.Spec.VerneMQ.SSLListenerCertSecretName == "" {
		fldPath := field.NewPath("spec").Child("vernemq").Child("sslListenerCertSecretName")
		err := field.Required(fldPath, "When deploying VerneMQ's SSL Listener, you must provide also SSLListenerCertSecretName.")
		return err
	}

	// ensure the TLS secret is present
	theSecret := &v1.Secret{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: r.Spec.VerneMQ.SSLListenerCertSecretName, Namespace: r.Namespace}, theSecret); err != nil {
		fldPath := field.NewPath("spec").Child("vernemq").Child("sslListenerCertSecretName")
		err := field.NotFound(fldPath, r.Spec.VerneMQ.SSLListenerCertSecretName)
		astartelog.Error(err, "SSLListenerCertSecretName secret %s not found in namespace %s",
			r.Spec.VerneMQ.SSLListenerCertSecretName, r.Namespace)
		return err
	}

	return nil
}

func validateCassandraDefinition(cassandra commontypes.AstarteCassandraSpec) *field.Error {
	if !pointy.BoolValue(cassandra.Deploy, true) && cassandra.Nodes == "" {
		fldPath := field.NewPath("spec").Child("cassandra").Child("nodes")
		err := field.Required(fldPath, "When not deploying Cassandra from the operator, the 'nodes' must be specified.")
		return err
	}

	// All is good.
	return nil
}

func validateCFSSLDefinition(cfssl commontypes.AstarteCFSSLSpec) *field.Error {
	if !cfssl.Deploy && cfssl.URL == "" {
		fldPath := field.NewPath("spec").Child("cfssl").Child("url")
		err := field.Required(fldPath, "When not deploying CFSSL from the operator, 'url' must be specified.")
		return err
	}

	// All is good.
	return nil
}

func validateRabbitMQDefinition(r *Astarte, c client.Client) field.ErrorList {
	allErrors := field.ErrorList{}
	if !pointy.BoolValue(r.Spec.RabbitMQ.Deploy, true) {
		// We need to make sure that we have all needed components
		if r.Spec.RabbitMQ.Connection == nil {
			fldPath := field.NewPath("spec").Child("rabbitmq").Child("connection")
			err := field.Required(fldPath, "When not deploying RabbitMQ from the operator, the 'connection' section must be specified.")
			allErrors = append(allErrors, err)
		}
		if r.Spec.RabbitMQ.Connection.Host == "" {
			fldPath := field.NewPath("spec").Child("rabbitmq").Child("connection").Child("host")
			err := field.Required(fldPath, "When not deploying RabbitMQ from the operator, the host must be specified.")
			allErrors = append(allErrors, err)
		}
		if (r.Spec.RabbitMQ.Connection.Username == "" || r.Spec.RabbitMQ.Connection.Password == "") && r.Spec.RabbitMQ.Connection.Secret == nil {
			fldPath := field.NewPath("spec").Child("rabbitmq").Child("connection").Child("host")
			err := field.Required(fldPath,
				"When not deploying RabbitMQ from the operator, either username and password or a secret containing credentials must be specified.")
			allErrors = append(allErrors, err)
		}
		if r.Spec.RabbitMQ.Connection.Secret != nil {
			// Check if the secret exists and has the keys we expect
			theSecret := &v1.Secret{}
			if err := c.Get(context.Background(), types.NamespacedName{Name: r.Spec.RabbitMQ.Connection.Secret.Name, Namespace: r.Namespace}, theSecret); err != nil {
				fldPath := field.NewPath("spec").Child("rabbitmq").Child("connection").Child("secret").Child("name")
				err := field.NotFound(fldPath, r.Spec.RabbitMQ.Connection.Secret.Name)
				allErrors = append(allErrors, err)
			} else {
				if _, ok := theSecret.Data[r.Spec.RabbitMQ.Connection.Username]; !ok {
					fldPath := field.NewPath("spec").Child("rabbitmq").Child("connection").Child("secret").Child("username")
					err := field.Required(fldPath, "Specified username key not found in secret")
					allErrors = append(allErrors, err)
				}
				if _, ok := theSecret.Data[r.Spec.RabbitMQ.Connection.Password]; !ok {
					fldPath := field.NewPath("spec").Child("rabbitmq").Child("connection").Child("secret").Child("password")
					err := field.Required(fldPath, "Specified password key not found in secret")
					allErrors = append(allErrors, err)
				}
			}
		}
	}

	return allErrors
}

// +kubebuilder:webhook:path=/mutate-api-astarte-platform-org-v1alpha1-astartevoyageringress,mutating=true,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=astartevoyageringresses,verbs=create;update,versions=v1alpha1,name=mastartevoyageringress.kb.io,admissionReviewVersions=v1beta1

var _ webhook.Defaulter = &AstarteVoyagerIngress{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AstarteVoyagerIngress) Default() {
	avilog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-api-astarte-platform-org-v1alpha1-astartevoyageringress,mutating=false,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=astartevoyageringresses,versions=v1alpha1,name=vastartevoyageringress.kb.io,admissionReviewVersions=v1beta1

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

// +kubebuilder:webhook:path=/mutate-api-astarte-platform-org-v1alpha1-flow,mutating=true,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=flows,verbs=create;update,versions=v1alpha1,name=mflow.kb.io,admissionReviewVersions=v1beta1

var _ webhook.Defaulter = &Flow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Flow) Default() {
	flowlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-api-astarte-platform-org-v1alpha1-flow,mutating=false,sideEffects=None,failurePolicy=fail,groups=api.astarte-platform.org,resources=flows,versions=v1alpha1,name=vflow.kb.io,admissionReviewVersions=v1beta1

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
