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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Flow is the Schema for the flows API
type Flow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlowSpec   `json:"spec,omitempty"`
	Status FlowStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FlowList contains a list of Flow
type FlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Flow `json:"items"`
}

// FlowState describes the global state of a Flow
type FlowState string

const (
	// FlowStateUnknown represents an Unknown State of the Flow. When in this state, it might
	// have never been reconciled.
	FlowStateUnknown FlowState = ""
	// FlowStateUnstable means the Flow is either reconciling or restarting some of its blocks.
	// It usually transitions to this State before moving to Flowing.
	FlowStateUnstable FlowState = "Unstable"
	// FlowStateUnhealthy means the Flow is currently having some non-transient or unrecoverable errors.
	// Manual intervention might be required.
	FlowStateUnhealthy FlowState = "Unhealthy"
	// FlowStateFlowing means the Flow is currently active and all of its blocks are stable. A healthy flow should stay
	// in this state for most of its lifecycle.
	FlowStateFlowing FlowState = "Flowing"
)

// RabbitMQConfig represents configuration for RabbitMQ
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RabbitMQConfig struct {
	metav1.TypeMeta `json:",inline"`
	Host            string `json:"host"`
	// +optional
	Port int16 `json:"port,omitempty"`
	// +optional
	SSL      *bool  `json:"ssl,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// RabbitMQExchange is a representation of a RabbitMQ Exchange
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RabbitMQExchange struct {
	metav1.TypeMeta `json:",inline"`
	Name            string `json:"name"`
	RoutingKey      string `json:"routingKey"`
}

// RabbitMQDataProvider is a representation of a Data Provider based upon RabbitMQ
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RabbitMQDataProvider struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	Queues []string `json:"queues,omitempty"`
	// +optional
	Exchange *RabbitMQExchange `json:"exchange,omitempty"`
	// RabbitMQConfig is an optional field which allows to specify configuration for an external RabbitMQ
	// broker. If not specified, Astarte's main Broker will be used.
	// +optional
	RabbitMQConfig *RabbitMQConfig `json:"rabbitmq,omitempty"`
}

// Type returns the type of the Data Provider
func (r *RabbitMQDataProvider) Type() string {
	return "rabbitmq"
}

// IsProducer returns whether the Data Provider has a Producer stage
func (r *RabbitMQDataProvider) IsProducer() bool {
	return r.Exchange != nil
}

// IsConsumer returns whether the Data Provider has a Consumer stage
func (r *RabbitMQDataProvider) IsConsumer() bool {
	return len(r.Queues) > 0
}

// DataProvider is a struct which defines which Data Providers (e.g. Brokers) are available for a
// Worker
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DataProvider struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	RabbitMQ *RabbitMQDataProvider `json:"rabbitmq,omitempty"`
}

// BlockWorker defines a Worker for a Container Block
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BlockWorker struct {
	metav1.TypeMeta `json:",inline"`
	WorkerID        string       `json:"id"`
	DataProvider    DataProvider `json:"dataProvider"`
}

// ContainerBlockSpec defines a Container Block in a Flow
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ContainerBlockSpec struct {
	metav1.TypeMeta `json:",inline"`
	BlockID         string `json:"id"`
	Image           string `json:"image"`
	// +optional
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// +optional
	Environment []v1.EnvVar `json:"environment"`
	// +optional
	Resources v1.ResourceRequirements `json:"resources"`
	// Configuration represents the JSON string carrying the user configuration for this block
	Configuration string `json:"config"`
	//+kubebuilder:validation:MinItems:=1
	Workers []BlockWorker `json:"workers"`
}

// FlowSpec defines the desired state of Flow
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FlowSpec struct {
	metav1.TypeMeta `json:",inline"`
	Astarte         v1.LocalObjectReference `json:"astarte"`
	AstarteRealm    string                  `json:"astarteRealm"`
	// Defines the amount of non-container blocks in the Flow
	NativeBlocks int `json:"nativeBlocks"`
	// Defines the overall resources consumed by Native Blocks
	NativeBlocksResources v1.ResourceList `json:"nativeBlocksResources"`
	// EE Only: Defines the Flow Pool in which the Flow will be allocated.
	FlowPool        v1.LocalObjectReference `json:"flowPool,omitempty"`
	ContainerBlocks []ContainerBlockSpec    `json:"blocks"`
}

// FlowStatus defines the observed state of Flow
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FlowStatus struct {
	metav1.TypeMeta `json:",inline"`
	// State defines the overall state of the Flow
	State FlowState `json:"state"`
	// Represents the total number of the Container Blocks in the Flow
	TotalContainerBlocks int `json:"totalContainerBlocks"`
	// Represents the total number of Ready Container Blocks in the Flow. In a healthy Flow,
	// this matches the number of Total Container Blocks.
	ReadyContainerBlocks int `json:"readyContainerBlocks"`
	// The overall resources allocated in the cluster for this Block
	Resources v1.ResourceList `json:"resources"`
	// Represents the total number of Container Blocks with non temporary failures. Present only
	// if any of the Blocks is in such state. When present, manual intervention is most likely required.
	// +optional
	FailingContainerBlocks int `json:"failingContainerBlocks,omitempty"`
	// UnrecoverableFailures lists all the ContainerStates of failing containers, for further inspection.
	// +optional
	UnrecoverableFailures []v1.ContainerState `json:"unrecoverableFailures,omitempty"`
}
