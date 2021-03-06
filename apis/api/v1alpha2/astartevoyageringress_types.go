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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/astarte-platform/astarte-kubernetes-operator/apis/api/commontypes"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=avi
// +kubebuilder:subresource:status
// AstarteVoyagerIngress is the Schema for the astartevoyageringresses API
type AstarteVoyagerIngress struct {
	Status commontypes.AstarteVoyagerIngressStatus `json:"status,omitempty"`
	Spec   commontypes.AstarteVoyagerIngressSpec   `json:"spec,omitempty"`

	metav1.ObjectMeta `json:"metadata,omitempty"`
	metav1.TypeMeta   `json:",inline"`
}

// +kubebuilder:object:root=true

// AstarteVoyagerIngressList contains a list of AstarteVoyagerIngress
type AstarteVoyagerIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AstarteVoyagerIngress `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AstarteVoyagerIngress{}, &AstarteVoyagerIngressList{})
}
