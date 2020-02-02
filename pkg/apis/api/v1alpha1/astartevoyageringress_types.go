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

package v1alpha1

import (
	voyager "github.com/astarte-platform/astarte-kubernetes-operator/external/voyager/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AstarteGenericIngressSpec is a common struct for all Ingresses defined by AstarteVoyagerIngress
type AstarteGenericIngressSpec struct {
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// +optional
	Type string `json:"type,omitempty"`
	// +optional
	LoadBalancerIP string `json:"loadBalancerIp,omitempty"`
	// +optional
	NodeSelector string `json:"nodeSelector,omitempty"`
	// +optional
	TLSSecret string `json:"tlsSecret,omitempty"`
	// +optional
	TLSRef *voyager.LocalTypedReference `json:"tlsRef,omitempty"`
	// +optional
	AnnotationsService map[string]string `json:"annotationsService,omitempty"`
}

// AstarteVoyagerIngressAPISpec defines the specification of the APIs
type AstarteVoyagerIngressAPISpec struct {
	AstarteGenericIngressSpec `json:",inline"`
	// +optional
	Cors *bool `json:"cors,omitempty"`
	// +optional
	ExposeHousekeeping *bool `json:"exposeHousekeeping,omitempty"`
}

// AstarteVoyagerIngressDashboardSpec defines the specification of the Dashboard
type AstarteVoyagerIngressDashboardSpec struct {
	// +optional
	SSL *bool `json:"ssl,omitempty"`
	// +optional
	Host string `json:"host,omitempty"`
	// +optional
	TLSSecret string `json:"tlsSecret,omitempty"`
	// +optional
	TLSRef *voyager.LocalTypedReference `json:"tlsRef,omitempty"`
}

// AstarteVoyagerIngressBrokerSpec defines the specification of the Broker
type AstarteVoyagerIngressBrokerSpec struct {
	AstarteGenericIngressSpec `json:",inline"`
	// +optional
	MaxConnections *int `json:"maxConnections,omitempty"`
}

// AstarteVoyagerIngressLetsEncryptSpec defines the specification of the Let's Encrypt Integration
type AstarteVoyagerIngressLetsEncryptSpec struct {
	// +optional
	Use *bool `json:"use,omitempty"`
	// +optional
	Staging *bool `json:"staging,omitempty"`
	// +optional
	AcmeEmail string `json:"acmeEmail,omitempty"`
	// +optional
	Domains []string `json:"domains,omitempty"`
	// +optional
	AutoHTTPChallenge *bool `json:"autoHTTPChallenge,omitempty"`
	// +optional
	ChallengeProvider voyager.ChallengeProvider `json:"challengeProvider,omitempty"`
}

// AstarteVoyagerIngressSpec defines the desired state of AstarteVoyagerIngress
type AstarteVoyagerIngressSpec struct {
	// +optional
	ImagePullPolicy *v1.PullPolicy `json:"imagePullPolicy,omitempty"`
	Astarte         string         `json:"astarte"`
	// +optional
	API AstarteVoyagerIngressAPISpec `json:"api,omitempty"`
	// +optional
	Dashboard AstarteVoyagerIngressDashboardSpec `json:"dashboard,omitempty"`
	// +optional
	Broker AstarteVoyagerIngressBrokerSpec `json:"broker,omitempty"`
	// +optional
	Letsencrypt AstarteVoyagerIngressLetsEncryptSpec `json:"letsencrypt,omitempty"`
}

// AstarteVoyagerIngressStatus defines the observed state of AstarteVoyagerIngress
type AstarteVoyagerIngressStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AstarteVoyagerIngress is the Schema for the astartevoyageringresses API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=astartevoyageringresses,scope=Namespaced,shortName=avi
type AstarteVoyagerIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AstarteVoyagerIngressSpec   `json:"spec,omitempty"`
	Status AstarteVoyagerIngressStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AstarteVoyagerIngressList contains a list of AstarteVoyagerIngress
type AstarteVoyagerIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AstarteVoyagerIngress `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AstarteVoyagerIngress{}, &AstarteVoyagerIngressList{})
}
