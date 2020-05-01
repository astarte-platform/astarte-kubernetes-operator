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
	operator "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AstarteTestResource is a base common ground for all tests to have a known Astarte resource
var AstarteTestResource *operator.Astarte = &operator.Astarte{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example-astarte",
	},
	Spec: operator.AstarteSpec{
		Version: "0.11.0",
		// Use the "Recreate" strategy. Some test environments are really constrained, and might not have enough
		// resources to support RollingUpdate.
		DeploymentStrategy: appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		},
		API: operator.AstarteAPISpec{
			Host: "api.autotest.astarte-platform.org",
		},
		VerneMQ: operator.AstarteVerneMQSpec{
			Host: "broker.autotest.astarte-platform.org",
			AstarteGenericClusteredResource: operator.AstarteGenericClusteredResource{
				Resources: &v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceCPU:    *resource.NewScaledQuantity(0, resource.Milli),
						v1.ResourceMemory: *resource.NewScaledQuantity(512, resource.Mega),
					},
					Requests: v1.ResourceList{
						v1.ResourceCPU:    *resource.NewScaledQuantity(0, resource.Milli),
						v1.ResourceMemory: *resource.NewScaledQuantity(256, resource.Mega),
					},
				},
			},
		},
		RabbitMQ: operator.AstarteRabbitMQSpec{
			AstarteGenericClusteredResource: operator.AstarteGenericClusteredResource{
				Resources: &v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceCPU:    *resource.NewScaledQuantity(1000, resource.Milli),
						v1.ResourceMemory: *resource.NewScaledQuantity(512, resource.Mega),
					},
					Requests: v1.ResourceList{
						v1.ResourceCPU:    *resource.NewScaledQuantity(200, resource.Milli),
						v1.ResourceMemory: *resource.NewScaledQuantity(256, resource.Mega),
					},
				},
			},
		},
		Cassandra: operator.AstarteCassandraSpec{
			MaxHeapSize: "512M",
			HeapNewSize: "256M",
			Storage: &operator.AstartePersistentStorageSpec{
				Size: resource.NewScaledQuantity(10, resource.Giga),
			},
			AstarteGenericClusteredResource: operator.AstarteGenericClusteredResource{
				Resources: &v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceCPU:    *resource.NewScaledQuantity(1000, resource.Milli),
						v1.ResourceMemory: *resource.NewScaledQuantity(2048, resource.Mega),
					},
					Requests: v1.ResourceList{
						v1.ResourceCPU:    *resource.NewScaledQuantity(500, resource.Milli),
						v1.ResourceMemory: *resource.NewScaledQuantity(1024, resource.Mega),
					},
				},
			},
		},
		CFSSL: operator.AstarteCFSSLSpec{
			Resources: &v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU:    *resource.NewScaledQuantity(0, resource.Milli),
					v1.ResourceMemory: *resource.NewScaledQuantity(128, resource.Mega),
				},
				Requests: v1.ResourceList{
					v1.ResourceCPU:    *resource.NewScaledQuantity(0, resource.Milli),
					v1.ResourceMemory: *resource.NewScaledQuantity(128, resource.Mega),
				},
			},
			Storage: &operator.AstartePersistentStorageSpec{
				Size: resource.NewScaledQuantity(2, resource.Giga),
			},
		},
		Components: operator.AstarteComponentsSpec{
			Resources: &v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU:    *resource.NewScaledQuantity(0, resource.Milli),
					v1.ResourceMemory: *resource.NewScaledQuantity(3, resource.Giga),
				},
				Requests: v1.ResourceList{
					v1.ResourceCPU:    *resource.NewScaledQuantity(0, resource.Milli),
					v1.ResourceMemory: *resource.NewScaledQuantity(2, resource.Giga),
				},
			},
			// TODO: We need to add this here to ensure we don't starve the CI. Remove when it is taken into account
			// in global resource allocation.
			Flow: operator.AstarteGenericAPISpec{
				AstarteGenericClusteredResource: operator.AstarteGenericClusteredResource{
					Resources: &v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    *resource.NewScaledQuantity(0, resource.Milli),
							v1.ResourceMemory: *resource.NewScaledQuantity(256, resource.Mega),
						},
						Requests: v1.ResourceList{
							v1.ResourceCPU:    *resource.NewScaledQuantity(0, resource.Milli),
							v1.ResourceMemory: *resource.NewScaledQuantity(256, resource.Mega),
						},
					},
				},
			},
		},
	},
}
