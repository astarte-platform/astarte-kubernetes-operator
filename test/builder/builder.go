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

package builder

import (
	"github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	"go.openly.dev/pointy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TestAstarteBuilder struct {
	astarte *v2alpha1.Astarte
}

func (b *TestAstarteBuilder) Build() *v2alpha1.Astarte {
	return b.astarte
}

// NewTestAstarteBuilder creates a new builder with default values
func NewTestAstarteBuilder(name, namespace string) *TestAstarteBuilder {
	return &TestAstarteBuilder{
		astarte: &v2alpha1.Astarte{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v2alpha1.AstarteSpec{
				Version: "1.3.0",
				RabbitMQ: v2alpha1.AstarteRabbitMQSpec{
					Connection: &v2alpha1.AstarteRabbitMQConnectionSpec{
						HostAndPort: v2alpha1.HostAndPort{
							Host: "rabbitmq.example.com",
							Port: pointy.Int32(5672),
						},
					},
				},
				VerneMQ: v2alpha1.AstarteVerneMQSpec{
					HostAndPort: v2alpha1.HostAndPort{
						Host: "vernemq.example.com",
						Port: pointy.Int32(8883),
					},
				},
				Cassandra: v2alpha1.AstarteCassandraSpec{
					Connection: &v2alpha1.AstarteCassandraConnectionSpec{
						Nodes: []v2alpha1.HostAndPort{
							{
								Host: "cassandra.example.com",
								Port: pointy.Int32(9042),
							},
						},
					},
				},
			},
		},
	}
}

// WithVersion sets the Astarte version
func (b *TestAstarteBuilder) WithVersion(version string) *TestAstarteBuilder {
	b.astarte.Spec.Version = version
	return b
}

// WithManualMaintenanceMode enables or disables manual maintenance mode
func (b *TestAstarteBuilder) WithManualMaintenanceMode(enabled bool) *TestAstarteBuilder {
	b.astarte.Spec.ManualMaintenanceMode = enabled
	return b
}

// WithRabbitMQ configures RabbitMQ settings
func (b *TestAstarteBuilder) WithRabbitMQ(host string, port int32) *TestAstarteBuilder {
	b.astarte.Spec.RabbitMQ.Connection.HostAndPort.Host = host
	b.astarte.Spec.RabbitMQ.Connection.HostAndPort.Port = pointy.Int32(port)
	return b
}

// WithVerneMQ configures VerneMQ settings
func (b *TestAstarteBuilder) WithVerneMQ(host string, port int32) *TestAstarteBuilder {
	b.astarte.Spec.VerneMQ.HostAndPort.Host = host
	b.astarte.Spec.VerneMQ.HostAndPort.Port = pointy.Int32(port)
	return b
}

// WithCassandraNodes sets the Cassandra nodes
func (b *TestAstarteBuilder) WithCassandraNodes(nodes []v2alpha1.HostAndPort) *TestAstarteBuilder {
	b.astarte.Spec.Cassandra.Connection.Nodes = nodes
	return b
}
