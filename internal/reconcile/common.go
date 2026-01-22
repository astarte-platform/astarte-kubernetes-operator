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

package reconcile

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/misc"
)

// EnsureHousekeepingKey makes sure that a valid Housekeeping key is available
func EnsureHousekeepingKey(cr *apiv2alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	publicSecretName := fmt.Sprintf("%s-housekeeping-public-key", cr.Name)
	theSecret := &v1.Secret{}
	err := c.Get(context.TODO(), types.NamespacedName{Name: publicSecretName, Namespace: cr.Namespace}, theSecret)
	if err != nil && !errors.IsNotFound(err) {
		return err
	} else if errors.IsNotFound(err) {
		// Let's create one.
		privateSecretName := fmt.Sprintf("%s-housekeeping-private-key", cr.Name)
		reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
		// Check if a private key already exists - in that case, we want to erase it.
		err := c.Get(context.TODO(), types.NamespacedName{Name: privateSecretName, Namespace: cr.Namespace}, theSecret)
		if err == nil {
			// If the call had no errors, it means the private key exists.
			reqLogger.Info("Existing Housekeeping Private Key found with no matching public key: deleting the existing private key")
			if err = c.Delete(context.TODO(), theSecret); err != nil {
				reqLogger.Error(err, "Could not delete the previous Housekeeping Private key!")
				return err
			}
		} else if !errors.IsNotFound(err) {
			return err
		}

		reqLogger.Info("Housekeeping Key not found: creating one")

		key, err := generateKeyPair()
		if err != nil {
			return err
		}

		reqLogger.Info("Creating Housekeeping private Key Secret")
		if err = storePrivateKeyInSecret(privateSecretName, key, cr, c, scheme); err != nil {
			return err
		}

		reqLogger.Info("Creating Housekeeping public Key Secret")
		if err = storePublicKeyInSecret(publicSecretName, &key.PublicKey, cr, c, scheme); err != nil {
			return err
		}
	}

	// All good.
	return nil
}

// EnsureSecretKeyBase makes sure that a valid Secret Key Base is available
// for FDO Device Onboarding and other services that may need it.
// If there is none, it creates a new one and stores it in a Secret.
func EnsureSecretKeyBase(cr *apiv2alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	secretName := fmt.Sprintf("%s-secret-key-base", cr.Name)

	theSecret := &v1.Secret{}
	err := c.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: cr.Namespace}, theSecret)

	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if errors.IsNotFound(err) {
		// Let's create one.
		reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
		reqLogger.Info("Secret Key Base not found: creating one")

		// Secret will be encoded in base64, with 48 bytes the output string will be 64 characters long
		b := make([]byte, 48)
		_, err := rand.Read(b)
		if err != nil {
			return err
		}

		// Encode bytes in base64
		k := base64.StdEncoding.EncodeToString(b)

		// Store it in a Secret
		s := map[string]string{
			"key": k,
		}

		_, err = misc.ReconcileSecretString(secretName, s, cr, c, scheme, reqLogger)
		return err
	}

	// If the secret exists but is empty or malformed, Astarte will crash and log an error.
	// For this reason, we do not check for that here, leaving Astarte to handle it.

	return err
}

// EnsureGenericErlangConfiguration reconciles the generic Erlang Configuration for Astarte services
func EnsureGenericErlangConfiguration(cr *apiv2alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	genericErlangConfigurationMapName := fmt.Sprintf("%s-generic-erlang-configuration", cr.Name)

	genericErlangConfigurationMapData := map[string]string{
		"vm.args": `## Name of the node
-name ${RELEASE_NAME}@${MY_POD_IP}

## Cookie for distributed erlang
-setcookie ${ERLANG_COOKIE}

## Heartbeat management; auto-restarts VM if it dies or becomes unresponsive
## (Disabled by default..use with caution!)
##-heart

## Enable kernel poll and a few async threads
##+K true
##+A 5

## Increase number of concurrent ports/sockets
##-env ERL_MAX_PORTS 4096

## Tweak GC to run more often
##-env ERL_FULLSWEEP_AFTER 10

# Enable SMP automatically based on availability
-smp auto
`,
	}

	_, err := misc.ReconcileConfigMap(genericErlangConfigurationMapName, genericErlangConfigurationMapData, cr, c, scheme, log)
	return err
}

// EnsureErlangClusteringCookie reconciles the Erlang Cookie Secret needed for Astarte services RPCs
func EnsureErlangClusteringCookie(cr *apiv2alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	secretName := cr.Name + "-erlang-clustering-cookie"

	return ensureErlangCookieSecret(secretName, cr, c, scheme)
}

func GetAstarteClusteredServicePolicyRules() []rbacv1.PolicyRule {
	// This is needed for Astarte > 1.2.0, as DUP/AppEngine/VerneMQ/RM/Pairing are clustered using Erlang.
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods", "endpoints"},
			Verbs:     []string{"list", "get"},
		},
	}
}
