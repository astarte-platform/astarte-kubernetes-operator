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
	"fmt"
	"strings"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
)

// This stuff is useful for other components which need to interact with Cassandra
func getCassandraNodes(cr *apiv2alpha1.Astarte) string {
	nodes := []string{}
	for _, node := range cr.Spec.Cassandra.Connection.Nodes {
		nodes = append(nodes, fmt.Sprintf("%s:%d", node.Host, node.Port))
	}

	return strings.Join(nodes, ",")
}
