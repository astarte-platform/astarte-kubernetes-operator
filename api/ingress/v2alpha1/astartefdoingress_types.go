/*
This file is part of Astarte.

Copyright 2020-26 SECO Mind Srl.

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

package v2alpha1

import (
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AstarteFDOIngressSpec defines the desired state of AstarteFDOIngress.
type AstarteFDOIngressSpec struct {
	// The name of the Astarte instance for which the FDO Ingress is being created.
	Astarte string `json:"astarte"`
	// In clusters with more than one ingress controllers, all
	// instances of the controllers must be aware of which Ingress object they must serve.
	// The ingressClass field of a ingress object is the way to let the controller know about that.
	// If the annotation is not set, HAProxy Ingress Controller is assumed by default.
	// +kubebuilder:default="haproxy"
	IngressClass string `json:"ingressClass"`
	// The secret containing the TLS certificates and keys used to connect to Astarte FDO Ingress. The secret
	// must be present in the namespace in which Astarte resides and it will be used to authenticate
	// requests to Astarte Pairing using FDO.
	// +optional
	TLSSecret string `json:"tlsSecret"`
}

// AstarteFDOIngressStatus defines the observed state of AstarteFDOIngress.
type AstarteFDOIngressStatus struct {
	metav1.TypeMeta            `json:",inline"`
	networkingv1.IngressStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AstarteFDOIngress is the Schema for the astartefdoingresses API.
type AstarteFDOIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AstarteFDOIngressSpec   `json:"spec,omitempty"`
	Status AstarteFDOIngressStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AstarteFDOIngressList contains a list of AstarteFDOIngress.
type AstarteFDOIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AstarteFDOIngress `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AstarteFDOIngress{}, &AstarteFDOIngressList{})
}
