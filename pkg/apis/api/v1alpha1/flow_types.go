package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
type RabbitMQConfig struct {
	Host string `json:"host"`
	// +optional
	Port int16 `json:"port,omitempty"`
	// +optional
	SSL      *bool  `json:"ssl,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// RabbitMQExchange is a representation of a RabbitMQ Exchange
type RabbitMQExchange struct {
	Name       string `json:"name"`
	RoutingKey string `json:"routingKey"`
}

// DataProviderInterface is the base interface representing Data Providers. It is implemented by any provider
// (e.g. RabbitMQ)
type DataProviderInterface interface {
	// Type returns the type of the Data Provider
	Type() string
	// IsProducer returns whether the Data Provider has a Producer stage
	IsProducer() bool
	// IsConsumer returns whether the Data Provider has a Consumer stage
	IsConsumer() bool
}

// RabbitMQDataProvider is a representation of a Data Provider based upon RabbitMQ
type RabbitMQDataProvider struct {
	// +optional
	Queues []string `json:"queues,omitempty"`
	// +optional
	Exchange *RabbitMQExchange `json:"exchange,omitempty"`
	// RabbitMQConfig is an optional field which allows to specify configuration for an external RabbitMQ
	// broker. If not specified, Astarte's main Broker will be used.
	// +optional
	RabbitMQConfig *RabbitMQConfig `json:"rabbitmq"`
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
type DataProvider struct {
	// +optional
	RabbitMQ *RabbitMQDataProvider `json:"rabbitmq,omitempty"`
}

// BlockWorker defines a Worker for a Container Block
type BlockWorker struct {
	WorkerID     string       `json:"id"`
	DataProvider DataProvider `json:"dataProvider"`
}

// BlockSpec defines a Container Block in a Flow
type BlockSpec struct {
	BlockID string `json:"id"`
	Image   string `json:"image"`
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
type FlowSpec struct {
	Astarte      v1.LocalObjectReference `json:"astarte"`
	AstarteRealm string                  `json:"astarteRealm"`
	//+kubebuilder:validation:MinItems:=1
	Blocks []BlockSpec `json:"blocks"`
}

// FlowStatus defines the observed state of Flow
type FlowStatus struct {
	// State defines the overall state of the Flow
	State FlowState `json:"state"`
	// TotalBlocks represents the total number of the Blocks in the Flow
	TotalBlocks int `json:"totalBlocks"`
	// ReadyBlocks represents the total number of Ready Blocks in the Flow. In a healthy Flow,
	// this matches the number of Total Blocks.
	ReadyBlocks int `json:"readyBlocks"`
	// FailingBlocks represents the total number of Blocks with non temporary failures. Present only
	// if any of the Blocks is in such state. When present, manual intervention is most likely required.
	// +optional
	FailingBlocks int `json:"failingBlocks,omitempty"`
	// UnrecoverableFailures lists all the ContainerStates of failing containers, for further inspection.
	// +optional
	UnrecoverableFailures []v1.ContainerState `json:"unrecoverableFailures,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Flow is the Schema for the flows API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=flows,scope=Namespaced
type Flow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlowSpec   `json:"spec,omitempty"`
	Status FlowStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FlowList contains a list of Flow
type FlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Flow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Flow{}, &FlowList{})
}
