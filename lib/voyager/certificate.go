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

package voyager

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/openlyinc/pointy"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
	voyager "github.com/astarte-platform/astarte-kubernetes-operator/external/voyager/v1beta1"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
)

func EnsureCertificate(cr *apiv1alpha1.AstarteVoyagerIngress, parent *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme, log logr.Logger) error {
	acmeSecretName := cr.Name + "-voyager-acme-account"
	certificateName := getCertificateName(cr)
	if !pointy.BoolValue(cr.Spec.Letsencrypt.Use, true) {
		// We're not using Let's Encrypt, so we're stopping here.
		// However, maybe we have a certificate to clean up?
		certificate := &voyager.Certificate{}
		if err := c.Get(context.TODO(), types.NamespacedName{Name: certificateName, Namespace: cr.Namespace}, certificate); err == nil {
			// Delete the certificate
			return c.Delete(context.TODO(), certificate)
		}

		// If nothing was found or there was an error, we don't really care.
		return nil
	}

	// Ensure the ACME Secret, first of all
	data := map[string]string{"ACME_EMAIL": cr.Spec.Letsencrypt.AcmeEmail}
	if pointy.BoolValue(cr.Spec.Letsencrypt.Staging, false) {
		data["ACME_SERVER_URL"] = "https://acme-staging-v02.api.letsencrypt.org/directory"
	}
	if _, err := misc.ReconcileSecretString(acmeSecretName, data, cr, c, scheme, log); err != nil {
		return err
	}

	// It might not be time for creating the Certificate just yet. In case we're on a HTTP-01 challenge,
	// we need to wait until all of our ingresses are ready before even trying to create our certificate.
	// Let's check, and just return nil in that case. Reconciliation will happen due to the status change
	// on the Load Balancer.
	if (cr.Spec.Letsencrypt.ChallengeProvider.HTTP != nil || pointy.BoolValue(cr.Spec.Letsencrypt.AutoHTTPChallenge, false)) &&
		(!isAPIIngressReady(cr, c) || !isBrokerIngressReady(cr, c)) {
		log.Info("Skipping Certificate for now, as Ingresses are not ready yet. Will check again in next Reconciliation")
		return nil
	}

	// All clear. Let's build the Certificate
	domains := getCertificateDomains(cr, parent)

	certificate := &voyager.Certificate{ObjectMeta: metav1.ObjectMeta{Name: certificateName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, certificate, func() error {
		if err := controllerutil.SetControllerReference(cr, certificate, scheme); err != nil {
			return err
		}

		// Set all fields to our needed state
		certificate.Spec.Domains = domains
		certificate.Spec.ACMEUserSecretName = acmeSecretName
		if pointy.BoolValue(cr.Spec.Letsencrypt.AutoHTTPChallenge, false) {
			// We build the right HTTP Challenge for the user, which will work with our basic setup.
			certificate.Spec.ChallengeProvider = voyager.ChallengeProvider{
				HTTP: &voyager.HTTPChallengeProvider{
					Ingress: voyager.LocalTypedReference{
						Kind:       "Ingress",
						APIVersion: "voyager.appscode.com/v1beta1",
						Name:       getAPIIngressName(cr),
					},
				},
			}
		} else {
			certificate.Spec.ChallengeProvider = cr.Spec.Letsencrypt.ChallengeProvider
		}
		return nil
	})
	if err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, certificate)
	}

	return err
}

func getCertificateDomains(cr *apiv1alpha1.AstarteVoyagerIngress, parent *apiv1alpha1.Astarte) []string {
	domains := cr.Spec.Letsencrypt.Domains
	if len(domains) == 0 {
		// Compute the domains list based on the parent Astarte resource
		if pointy.BoolValue(parent.Spec.Components.Dashboard.Deploy, true) && cr.Spec.Dashboard.Host != "" &&
			pointy.BoolValue(cr.Spec.Dashboard.SSL, true) {
			domains = append(domains, cr.Spec.Dashboard.Host)
		}
		if pointy.BoolValue(parent.Spec.API.SSL, true) {
			domains = append(domains, parent.Spec.API.Host)
		}
		domains = append(domains, parent.Spec.VerneMQ.Host)
	}

	return domains
}

func getCertificateName(cr *apiv1alpha1.AstarteVoyagerIngress) string {
	return cr.Name + "-ingress-certificate"
}

func isBootstrappingLEChallenge(cr *apiv1alpha1.AstarteVoyagerIngress, c client.Client) (bool, error) {
	// If we're not using Let's Encrypt, that's pretty easy
	if !pointy.BoolValue(cr.Spec.Letsencrypt.Use, true) {
		return false, nil
	}
	// If we're not on a HTTP-01 Challenge, same deal.
	if cr.Spec.Letsencrypt.ChallengeProvider.HTTP == nil && !pointy.BoolValue(cr.Spec.Letsencrypt.AutoHTTPChallenge, false) {
		return false, nil
	}

	// We might be in a bootstrapping situation. Inspect the certificate to find out if it has been issued.
	certificate := &voyager.Certificate{}
	err := c.Get(context.TODO(), types.NamespacedName{Name: getCertificateName(cr), Namespace: cr.Namespace}, certificate)
	if err == nil {
		// Check the certificate status
		for _, cond := range certificate.Status.Conditions {
			if cond.Type == voyager.CertificateIssued {
				// We're good to go, and not bootstrapping any longer.
				return false, nil
			}
		}
		// If we got here, we're definitely bootstrapping.
		return true, nil
	} else if errors.IsNotFound(err) {
		// Definitely bootstrapping still
		return true, nil
	}

	// If there was an error in obtaining the certificate, we must fail to prevent disasters
	return false, err
}
