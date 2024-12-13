/*
Copyright 2024.

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

package migrate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
)

// The purpose of this package is solely the migration of the old operator's flaky CR to the shiny new, stable,
// v1alpha1. This is a "if ain't broken, don't fix it" area: if it works, it works, and it doesn't have to work
// in any situation but the migration from the old Operator.

// ToNewCR takes a flaky, existing Astarte instance already on the Kubernetes Cluster and reconciles it with
// the new format and right specifications without losing data.
func ToNewCR(cr *v1alpha2.Astarte, c client.Client, scheme *runtime.Scheme) error {
	oldAstarteObjectSpec, err := getOldSpec(cr, c)
	if err != nil {
		return err
	}

	// Ok, so: basically *all* resources objects have been interpreted as either a valid string OR integer. As such, we need to run through
	// all them to make sure they're consistent. Let's go.

	// CFSSL
	if cr.Spec.CFSSL.Resources, err = normalizeResourcesFor(cr.Spec.CFSSL.Resources, "cfssl.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// RabbitMQ
	if cr.Spec.RabbitMQ.Resources, err = normalizeResourcesFor(cr.Spec.RabbitMQ.Resources,
		"rabbitmq.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// Cassandra
	if cr.Spec.Cassandra.Resources, err = normalizeResourcesFor(cr.Spec.Cassandra.Resources,
		"cassandra.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// VerneMQ
	if cr.Spec.VerneMQ.Resources, err = normalizeResourcesFor(cr.Spec.VerneMQ.Resources,
		"vernemq.resources", oldAstarteObjectSpec); err != nil {
		return err
	}

	// All Components
	if cr, err = normalizeResourcesForAstarteComponents(cr, oldAstarteObjectSpec); err != nil {
		return err
	}

	// If we got here, we most likely survived the trip. It's time to update the resource and hope for the best.
	if err := c.Update(context.TODO(), cr); err != nil {
		return err
	}

	// Now, we need to deal with the StatefulSets. They will most likely refuse to update due to some changes in this operator.
	// So, delete the old statefulsets (without deleting the Pods) and wait for reconciliation.
	statefulsets := []string{
		cr.Name + "-rabbitmq",
		cr.Name + "-cfssl",
		cr.Name + "-cassandra",
		cr.Name + "-vernemq",
	}
	for _, s := range statefulsets {
		if err := deleteStatefulsetWithoutCascading(s, cr.Namespace, c); err != nil {
			return err
		}
	}

	// All good.
	return nil
}

func normalizeResourcesForAstarteComponents(cr *v1alpha2.Astarte, oldAstarteObjectSpec map[string]interface{}) (*v1alpha2.Astarte, error) {
	var err error

	// All Components
	if cr.Spec.Components.Resources, err = normalizeResourcesFor(cr.Spec.Components.Resources, "components.resources", oldAstarteObjectSpec); err != nil {
		return nil, err
	}

	// AppEngine API
	if cr.Spec.Components.AppengineAPI.Resources, err = normalizeResourcesFor(
		cr.Spec.Components.AppengineAPI.Resources, "components.appengineApi.resources", oldAstarteObjectSpec); err != nil {
		return nil, err
	}
	// Data Updater Plant
	if cr.Spec.Components.DataUpdaterPlant.Resources, err = normalizeResourcesFor(
		cr.Spec.Components.DataUpdaterPlant.Resources, "components.dataUpdaterPlant.resources", oldAstarteObjectSpec); err != nil {
		return nil, err
	}
	// Housekeeping Backend
	if cr.Spec.Components.Housekeeping.Backend.Resources, err = normalizeResourcesFor(cr.Spec.Components.Housekeeping.Backend.Resources,
		"components.housekeeping.backend.resources", oldAstarteObjectSpec); err != nil {
		return nil, err
	}
	// Housekeeping API
	if cr.Spec.Components.Housekeeping.API.Resources, err = normalizeResourcesFor(
		cr.Spec.Components.Housekeeping.API.Resources, "components.housekeeping.api.resources", oldAstarteObjectSpec); err != nil {
		return nil, err
	}
	// Pairing Backend
	if cr.Spec.Components.RealmManagement.Backend.Resources, err = normalizeResourcesFor(cr.Spec.Components.RealmManagement.Backend.Resources,
		"components.pairing.backend.resources", oldAstarteObjectSpec); err != nil {
		return nil, err
	}
	// Pairing API
	if cr.Spec.Components.Pairing.API.Resources, err = normalizeResourcesFor(
		cr.Spec.Components.Pairing.API.Resources, "components.pairing.api.resources", oldAstarteObjectSpec); err != nil {
		return nil, err
	}
	// Realm Management Backend
	if cr.Spec.Components.RealmManagement.Backend.Resources, err = normalizeResourcesFor(cr.Spec.Components.RealmManagement.Backend.Resources,
		"components.realmManagement.backend.resources", oldAstarteObjectSpec); err != nil {
		return nil, err
	}
	// Realm Management API
	if cr.Spec.Components.RealmManagement.API.Resources, err = normalizeResourcesFor(
		cr.Spec.Components.RealmManagement.API.Resources, "components.realmManagement.api.resources", oldAstarteObjectSpec); err != nil {
		return nil, err
	}
	// Trigger Engine
	if cr.Spec.Components.TriggerEngine.Resources, err = normalizeResourcesFor(cr.Spec.Components.TriggerEngine.Resources,
		"components.triggerEngine.resources", oldAstarteObjectSpec); err != nil {
		return nil, err
	}

	// Dashboard
	if cr.Spec.Components.Dashboard.Resources, err = normalizeResourcesFor(cr.Spec.Components.Dashboard.Resources,
		"components.dashboard.resources", oldAstarteObjectSpec); err != nil {
		return nil, err
	}

	return cr, nil
}

func getOldSpec(cr *v1alpha2.Astarte, c client.Client) (map[string]interface{}, error) {
	// Given we know that inconsistencies are around, we want to Get from the client an unstructured Object to make sure
	// we can inspect individual fields in the Spec map.
	oldAstarteObject := &unstructured.Unstructured{}
	oldAstarteObject.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "api.astarte-platform.org",
		Kind:    "Astarte",
		Version: "v1alpha2",
	})
	if err := c.Get(context.TODO(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, oldAstarteObject); err != nil {
		return nil, err
	}

	return oldAstarteObject.Object["spec"].(map[string]interface{}), nil
}

func deleteStatefulsetWithoutCascading(name, namespace string, c client.Client) error {
	oldStatefulSet := &appsv1.StatefulSet{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, oldStatefulSet); err != nil {
		if errors.IsNotFound(err) {
			// It's fine - if it wasn't deployed, there's no need to do anything else.
			return nil
		}
		return err
	}
	return c.Delete(context.TODO(), oldStatefulSet, client.PropagationPolicy(metav1.DeletePropagationOrphan))
}

func normalizeResourcesFor(parsedRequirements *v1.ResourceRequirements, resourceField string, objectSpec map[string]interface{}) (*v1.ResourceRequirements, error) {
	oldResources := getFromMapRecursively(objectSpec, strings.Split(resourceField, "."))
	if len(oldResources) == 0 {
		return parsedRequirements, nil
	}

	newRequirements := &v1.ResourceRequirements{}

	// Inspect all fields from the resources/limits and whatnots.
	if oldRequests, ok := oldResources["requests"]; ok {
		newRequirements.Requests = make(v1.ResourceList)
		if oldRequestsCPU, ok := oldRequests.(map[string]interface{})["cpu"]; ok {
			if q, err := parseFlakyQuantity(oldRequestsCPU); err == nil {
				newRequirements.Requests[v1.ResourceCPU] = q
			} else {
				return nil, err
			}
		}
		if oldRequestsMemory, ok := oldRequests.(map[string]interface{})["memory"]; ok {
			if q, err := parseFlakyQuantity(oldRequestsMemory); err == nil {
				newRequirements.Requests[v1.ResourceMemory] = q
			} else {
				return nil, err
			}
		}
	}
	if oldLimits, ok := oldResources["limits"]; ok {
		newRequirements.Limits = make(v1.ResourceList)
		if oldLimitsCPU, ok := oldLimits.(map[string]interface{})["cpu"]; ok {
			if q, err := parseFlakyQuantity(oldLimitsCPU); err == nil {
				newRequirements.Limits[v1.ResourceCPU] = q
			} else {
				return nil, err
			}
		}
		if oldLimitsMemory, ok := oldLimits.(map[string]interface{})["memory"]; ok {
			if q, err := parseFlakyQuantity(oldLimitsMemory); err == nil {
				newRequirements.Limits[v1.ResourceMemory] = q
			} else {
				return nil, err
			}
		}
	}

	return newRequirements, nil
}

func parseFlakyQuantity(flakyQuantity interface{}) (resource.Quantity, error) {
	switch v := flakyQuantity.(type) {
	// In the int/float cases, make sure we convert them to a string
	case int:
		return resource.ParseQuantity(strconv.Itoa(v))
	case int64:
		return resource.ParseQuantity(strconv.Itoa(int(v)))
	case float64:
		return resource.ParseQuantity(strconv.Itoa(int(v)))
	case string:
		return resource.ParseQuantity(v)
	}

	return resource.Quantity{}, fmt.Errorf("Could not parse %v as a quantity. Type is %T", flakyQuantity, flakyQuantity)
}

func getFromMapRecursively(aMap map[string]interface{}, tokens []string) map[string]interface{} {
	// Pop first element
	var token string
	switch {
	case len(tokens) == 0:
		return aMap
	case len(tokens) > 1:
		token, tokens = tokens[0], tokens[1:]
	default:
		token, tokens = tokens[0], []string{}
	}

	if _, ok := aMap[token]; ok {
		switch v := aMap[token].(type) {
		case map[string]interface{}:
			return getFromMapRecursively(v, tokens)
		default:
			// Pass - we'll return a not found.
		}
	}

	// Not found
	return map[string]interface{}{}
}
