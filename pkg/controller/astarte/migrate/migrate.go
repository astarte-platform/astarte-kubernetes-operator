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

package migrate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
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
)

// The purpose of this package is solely the migration of the old operator's flaky CR to the shiny new, stable,
// v1alpha1. This is a "if ain't broken, don't fix it" area: if it works, it works, and it doesn't have to work
// in any situation but the migration from the old Operator.

// ToNewCR takes a flaky, existing Astarte instance already on the Kubernetes Cluster and reconciles it with
// the new format and right specifications without losing data.
func ToNewCR(cr *v1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	// Given we know that inconsistencies are around, we want to Get from the client an unstructured Object to make sure
	// we can inspect individual fields in the Spec map.
	oldAstarteObject := &unstructured.Unstructured{}
	oldAstarteObject.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "api.astarte-platform.org",
		Kind:    "Astarte",
		Version: "v1alpha1",
	})
	if err := c.Get(context.TODO(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, oldAstarteObject); err != nil {
		return err
	}

	oldAstarteObjectSpec := oldAstarteObject.Object["spec"].(map[string]interface{})

	// Ok, so: basically *all* resources objects have been interpreted as either a valid string OR integer. As such, we need to run through
	// all them to make sure they're consistent. Let's go.

	// CFSSL
	var err error
	if cr.Spec.CFSSL.Resources, err = normalizeResourcesFor(cr.Spec.CFSSL.Resources, "cfssl.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// RabbitMQ
	if cr.Spec.RabbitMQ.GenericClusteredResource.Resources, err = normalizeResourcesFor(cr.Spec.RabbitMQ.GenericClusteredResource.Resources,
		"rabbitmq.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// Cassandra
	if cr.Spec.Cassandra.GenericClusteredResource.Resources, err = normalizeResourcesFor(cr.Spec.Cassandra.GenericClusteredResource.Resources,
		"cassandra.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// VerneMQ
	if cr.Spec.VerneMQ.GenericClusteredResource.Resources, err = normalizeResourcesFor(cr.Spec.VerneMQ.GenericClusteredResource.Resources,
		"vernemq.resources", oldAstarteObjectSpec); err != nil {
		return err
	}

	// All Components
	if cr.Spec.Components.Resources, err = normalizeResourcesFor(cr.Spec.Components.Resources, "components.resources", oldAstarteObjectSpec); err != nil {
		return err
	}

	// AppEngine API
	if cr.Spec.Components.AppengineAPI.GenericAPISpec.GenericClusteredResource.Resources, err = normalizeResourcesFor(
		cr.Spec.Components.AppengineAPI.GenericAPISpec.GenericClusteredResource.Resources, "components.appengineApi.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// Data Updater Plant
	if cr.Spec.Components.DataUpdaterPlant.GenericClusteredResource.Resources, err = normalizeResourcesFor(
		cr.Spec.Components.DataUpdaterPlant.GenericClusteredResource.Resources, "components.dataUpdaterPlant.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// Housekeeping Backend
	if cr.Spec.Components.Housekeeping.Backend.Resources, err = normalizeResourcesFor(cr.Spec.Components.Housekeeping.Backend.Resources,
		"components.housekeeping.backend.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// Housekeeping API
	if cr.Spec.Components.Housekeeping.API.GenericClusteredResource.Resources, err = normalizeResourcesFor(
		cr.Spec.Components.Housekeeping.API.GenericClusteredResource.Resources, "components.housekeeping.api.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// Pairing Backend
	if cr.Spec.Components.RealmManagement.Backend.Resources, err = normalizeResourcesFor(cr.Spec.Components.RealmManagement.Backend.Resources,
		"components.pairing.backend.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// Pairing API
	if cr.Spec.Components.Pairing.API.GenericClusteredResource.Resources, err = normalizeResourcesFor(
		cr.Spec.Components.Pairing.API.GenericClusteredResource.Resources, "components.pairing.api.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// Realm Management Backend
	if cr.Spec.Components.RealmManagement.Backend.Resources, err = normalizeResourcesFor(cr.Spec.Components.RealmManagement.Backend.Resources,
		"components.realmManagement.backend.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// Realm Management API
	if cr.Spec.Components.RealmManagement.API.GenericClusteredResource.Resources, err = normalizeResourcesFor(
		cr.Spec.Components.RealmManagement.API.GenericClusteredResource.Resources, "components.realmManagement.api.resources", oldAstarteObjectSpec); err != nil {
		return err
	}
	// Trigger Engine
	if cr.Spec.Components.TriggerEngine.Resources, err = normalizeResourcesFor(cr.Spec.Components.TriggerEngine.Resources,
		"components.triggerEngine.resources", oldAstarteObjectSpec); err != nil {
		return err
	}

	// Dashboard
	if cr.Spec.Components.Dashboard.GenericClusteredResource.Resources, err = normalizeResourcesFor(cr.Spec.Components.Dashboard.GenericClusteredResource.Resources,
		"components.dashboard.resources", oldAstarteObjectSpec); err != nil {
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

func normalizeResourcesFor(parsedRequirements v1.ResourceRequirements, resourceField string, objectSpec map[string]interface{}) (v1.ResourceRequirements, error) {
	oldResources := getFromMapRecursively(objectSpec, strings.Split(resourceField, "."))
	if len(oldResources) == 0 {
		return parsedRequirements, nil
	}

	newRequirements := v1.ResourceRequirements{}

	// Inspect all fields from the resources/limits and whatnots.
	if oldRequests, ok := oldResources["requests"]; ok {
		newRequirements.Requests = make(v1.ResourceList)
		if oldRequestsCPU, ok := oldRequests.(map[string]interface{})["cpu"]; ok {
			if q, err := parseFlakyQuantity(oldRequestsCPU); err == nil {
				newRequirements.Requests[v1.ResourceCPU] = q
			} else {
				return v1.ResourceRequirements{}, err
			}
		}
		if oldRequestsMemory, ok := oldRequests.(map[string]interface{})["memory"]; ok {
			if q, err := parseFlakyQuantity(oldRequestsMemory); err == nil {
				newRequirements.Requests[v1.ResourceMemory] = q
			} else {
				return v1.ResourceRequirements{}, err
			}
		}
	}
	if oldLimits, ok := oldResources["limits"]; ok {
		newRequirements.Limits = make(v1.ResourceList)
		if oldLimitsCPU, ok := oldLimits.(map[string]interface{})["cpu"]; ok {
			if q, err := parseFlakyQuantity(oldLimitsCPU); err == nil {
				newRequirements.Limits[v1.ResourceCPU] = q
			} else {
				return v1.ResourceRequirements{}, err
			}
		}
		if oldLimitsMemory, ok := oldLimits.(map[string]interface{})["memory"]; ok {
			if q, err := parseFlakyQuantity(oldLimitsMemory); err == nil {
				newRequirements.Limits[v1.ResourceMemory] = q
			} else {
				return v1.ResourceRequirements{}, err
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
	if len(tokens) == 0 {
		return aMap
	} else if len(tokens) > 1 {
		token, tokens = tokens[0], tokens[1:]
	} else {
		token, tokens = tokens[0], []string{}
	}

	if _, ok := aMap[token]; ok {
		switch aMap[token].(type) {
		case map[string]interface{}:
			return getFromMapRecursively(aMap[token].(map[string]interface{}), tokens)
		default:
			// Pass - we'll return a not found.
		}
	}

	// Not found
	return map[string]interface{}{}
}
