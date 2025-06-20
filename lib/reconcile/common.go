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

package reconcile

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha2"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
)

// EnsureHousekeepingKey makes sure that a valid Housekeeping key is available
func EnsureHousekeepingKey(cr *apiv1alpha2.Astarte, c client.Client, scheme *runtime.Scheme) error {
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

// EnsureGenericErlangConfiguration reconciles the generic Erlang Configuration for Astarte services
func EnsureGenericErlangConfiguration(cr *apiv1alpha2.Astarte, c client.Client, scheme *runtime.Scheme) error {
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

func GetAstarteClusteredServicePolicyRules() []rbacv1.PolicyRule {
	// This is needed for Astarte > 1.2.0, as DUP/AppEngine/VerneMQ are clustered using Erlang.
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods", "endpoints"},
			Verbs:     []string{"list", "get"},
		},
	}
}
