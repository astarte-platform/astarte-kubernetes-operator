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

package utils

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/astarte-platform/astarte-kubernetes-operator/apis/api/commontypes"
	operator "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
)

func EnsureAstarteBecomesGreen(name, namespace string, c client.Client) (commontypes.AstarteClusterHealth, error) {
	astarteLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	createdAstarte := &operator.Astarte{}

	if err := c.Get(context.Background(), astarteLookupKey, createdAstarte); err != nil {
		return commontypes.AstarteClusterHealthRed, err
	}

	return createdAstarte.Status.Health, nil
}

func GetAstarteStatusVersion(name, namespace string, c client.Client) (string, error) {
	astarteLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	installedAstarte := &operator.Astarte{}

	if err := c.Get(context.Background(), astarteLookupKey, installedAstarte); err != nil {
		return "", err
	}

	return installedAstarte.Status.AstarteVersion, nil
}

func GetAstarteReconciliationPhase(name, namespace string, c client.Client) (commontypes.ReconciliationPhase, error) {
	astarteLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
	installedAstarte := &operator.Astarte{}

	if err := c.Get(context.Background(), astarteLookupKey, installedAstarte); err != nil {
		return commontypes.ReconciliationPhaseUnknown, err
	}

	return installedAstarte.Status.ReconciliationPhase, nil
}
