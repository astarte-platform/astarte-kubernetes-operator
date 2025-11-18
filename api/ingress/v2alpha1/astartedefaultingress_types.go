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

package v2alpha1

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Support annotations for AstarteDefaultIngress objects.
const (
	// AnnotationIngressControllerSelector is used to specify the Ingress Controller to use with this AstarteDefaultIngress.
	// This is not related to the ingressClass field, which is used to specify which Ingress class the Ingress object should belong to.
	// Depending on the Ingress Controller in use, different annotations and configurations will be applied to the generated Ingress objects.
	// By default, the Astarte Operator assumes the HAProxy Ingress Controller is in use.
	// Value: "haproxy.org" or "nginx.ingress.kubernetes.io" (depending on the Ingress Controller in use)
	AnnotationIngressControllerSelector = "ingress.astarte-platform.org/ingress-controller-selector"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AstarteDefaultIngressSpec defines the desired state of AstarteDefaultIngress
type AstarteDefaultIngressSpec struct {
	metav1.TypeMeta `json:",inline"`
	// The name of the Astarte instance served by the AstarteDefaultIngress.
	Astarte string `json:"astarte"`
	// In clusters with more than one instance of the Ingress-NGINX controller, all
	// instances of the controllers must be aware of which Ingress object they must serve.
	// The ingressClass field of a ingress object is the way to let the controller know about that.
	// Default: "nginx".
	// +optional
	IngressClass string `json:"ingressClass"`
	// Define the desired state of the AstarteDefaultIngressAPISpec resource.
	// +optional
	API AstarteDefaultIngressAPISpec `json:"api,omitempty"`
	// Define the desired state of the AstarteDefaultIngressDashboardSpec resource.
	// +optional
	Dashboard AstarteDefaultIngressDashboardSpec `json:"dashboard,omitempty"`
	// Define the desired state of the AstarteDefaultIngressBrokerSpec resource.
	// +optional
	Broker AstarteDefaultIngressBrokerSpec `json:"broker,omitempty"`
	// The secret containing the TLS certificates and keys used to connect to Astarte. The secret
	// must be present in the namespace in which Astarte resides and it will be used to authenticate
	// requests for API and Dashboard. If specific configurations are required,
	// the TLSSecret can be overridden by setting the secret in any of AstarteDefaultIngressAPISpec
	// and AstarteDefaultIngressDashboardSpec.
	// +optional
	TLSSecret string `json:"tlsSecret"`
}

// AstarteDefaultIngressStatus defines the observed state of AstarteDefaultIngress
type AstarteDefaultIngressStatus struct {
	metav1.TypeMeta `json:",inline"`
	APIStatus       networkingv1.IngressStatus `json:"api,omitempty"`
	BrokerStatus    corev1.ServiceStatus       `json:"broker,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=adi

// AstarteDefaultIngress is the Schema for the astartedefaultingresses API
type AstarteDefaultIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AstarteDefaultIngressSpec   `json:"spec,omitempty"`
	Status AstarteDefaultIngressStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AstarteDefaultIngressList contains a list of AstarteDefaultIngress
type AstarteDefaultIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AstarteDefaultIngress `json:"items"`
}

// AstarteDefaultIngressAPISpec defines how the Astarte APIs are served.
type AstarteDefaultIngressAPISpec struct {
	metav1.TypeMeta `json:",inline"`
	// When true, deploy the API ingress.
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
	// The secret containing the TLS certificates and keys used to access the Astarte API. The secret
	// must be present in the namespace in which Astarte resides. If set, this secret overrides the TLSSecret
	// field contained in AstarteDefaultIngressSpec.
	// +optional
	TLSSecret string `json:"tlsSecret,omitempty"`
	// When true, enable Cross-Origin Resource Sharing (CORS). Default: false.
	// +optional
	Cors *bool `json:"cors,omitempty"`
	// When true, the housekeeping endpoint is publicly exposed. Default: true.
	// +optional
	ExposeHousekeeping *bool `json:"exposeHousekeeping,omitempty"`
}

// AstarteDefaultIngressDashboardSpec defines how the Astarte Dashboard is served.
type AstarteDefaultIngressDashboardSpec struct {
	metav1.TypeMeta `json:",inline"`
	// When true, deploy the Ingress for the Dashboard.
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
	// When true, enable TLS authentication for the Dashboard.
	// +optional
	SSL *bool `json:"ssl,omitempty"`
	// The host handling requests addressed to the dashboard. When deploy is true and host is not set,
	// the dashboard will be exposed at the following URL: https://<astarte-base-url>/dashboard.
	// +optional
	Host string `json:"host,omitempty"`
	// The secret containing the TLS certificates and keys used to access the Astarte Dashboard. The secret
	// must be present in the namespace in which Astarte resides. If set, this secret overrides the TLSSecret
	// field contained in AstarteDefaultIngressSpec.
	// +optional
	TLSSecret string `json:"tlsSecret,omitempty"`
}

// AstarteDefaultIngressBrokerSpec defines how the Astarte Broker is served.
type AstarteDefaultIngressBrokerSpec struct {
	metav1.TypeMeta `json:",inline"`
	// When true, expose the Broker.
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
	// Set the type of service employed to expose the broker. Supported values are "NodePort" and "LoadBalancer".
	// The AstarteDefaultIngress handles TLS termination at VerneMQ level and, as such, no TLSSecret is needed to
	// configure the broker service.
	// Default: "LoadBalancer"
	// kubebuilder:validation:Enum:=LoadBalancer,NodePort
	// kubebuilder:validation:Default:=LoadBalancer
	// +optional
	ServiceType v1.ServiceType `json:"serviceType,omitempty"`
	// Set the LoadBalancerIP if and only if the broker service is of type "LoadBalancer". This feature depends on
	// whether the cloud provider supports specifying the LoadBalancerIP when a load balancer is created.
	// +optional
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`
	// Additional annotations for the service exposing this broker.
	// +optional
	ServiceAnnotations map[string]string `json:"serviceAnnotations,omitempty"`
}

func init() {
	SchemeBuilder.Register(&AstarteDefaultIngress{}, &AstarteDefaultIngressList{})
}
