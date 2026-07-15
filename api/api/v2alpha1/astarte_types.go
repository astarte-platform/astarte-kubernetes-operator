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

// +kubebuilder:object:generate=true
// +groupName=api.astarte-platform.org
package v2alpha1

import (
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Supported custom annotations in Astarte CR.
const (
	// AnnotationHideDashboardSidebar allows to hide the Dashboard sidebar.
	// It is propagated to the Astarte Dashboard configmap
	// Value: "true" or "false"
	AnnotationHideDashboardSidebar = "api.astarte-platform.org/hide-dashboard-sidebar"
)

// AstarteSpec defines the desired state of Astarte
type AstarteSpec struct {
	// The Astarte version for this Resource. This field is required and must be a valid
	// semver string (e.g. "1.3.0" or "1.4.1"). The Operator uses this version
	// to determine which images to pull and what features to enable.
	Version string `json:"version"`
	// Features allows enabling or disabling a set of global, opt-in Astarte features.
	// +kubebuilder:validation:Optional
	Features AstarteFeatures `json:"features,omitempty"`
	// The default image pull policy for all Astarte services. Can be overridden
	// per-component by setting the component's imagePullPolicy field.
	// Default: "IfNotPresent".
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="IfNotPresent"
	ImagePullPolicy *v1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Image pull secrets that will be added to all Astarte pods. Each component can
	// add additional secrets through its own imagePullSecrets field.
	// +kubebuilder:validation:Optional
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// The distribution channel (container registry prefix) for Astarte images.
	// This setting can be overridden by explicitly setting the 'image' value for
	// each service. Defaults to "astarte".
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="astarte"
	DistributionChannel string `json:"distributionChannel,omitempty"`
	// The global default deployment strategy for Astarte components. Can be overridden
	// per-component by setting the component's deploymentStrategy field. Note that
	// DataUpdaterPlant, TriggerEngine, and Flow always use Recreate regardless
	// of this setting. When left unset, defaults to RollingUpdate.
	// +kubebuilder:validation:Optional
	DeploymentStrategy *appsv1.DeploymentStrategy `json:"deploymentStrategy,omitempty"`
	// The default storage class name for persistent storage claims. When a component
	// requires a persistent volume and does not specify its own className, this value
	// is used as a fallback.
	// +kubebuilder:validation:Optional
	StorageClassName string `json:"storageClassName,omitempty"`
	// API defines the external API host and SSL configuration for Astarte services.
	// The Host field is required.
	API AstarteAPISpec `json:"api"`
	// RabbitMQ connection and management configuration for the Astarte message broker.
	// +kubebuilder:validation:Optional
	RabbitMQ AstarteRabbitMQSpec `json:"rabbitmq"`
	// Cassandra/ScyllaDB connection configuration for the Astarte database.
	// The connection and astarteSystemKeyspace fields are required.
	// +kubebuilder:validation:Optional
	Cassandra AstarteCassandraSpec `json:"cassandra"`
	// VerneMQ broker configuration including networking, storage, and device heartbeat settings.
	VerneMQ AstarteVerneMQSpec `json:"vernemq"`
	// Vault is used to connect to a OpenBao or HashiCorp Vault instance.
	// Setting this field is supported and mandatory for Astarte version 1.4 and later.
	// The field is ignored for Astarte 1.3.
	// +kubebuilder:validation:Optional
	Vault *AstarteVaultSpec `json:"vault,omitempty"`
	// FDO (FIDO Device Onboarding) configuration. Available as an opt-in feature
	// starting from Astarte 1.3. From Astarte 1.4.0 onwards, FDO is mandatory
	// and cannot be disabled.
	// +kubebuilder:validation:Optional
	FDO *AstarteFDOSpec `json:"fdo,omitempty"`
	// CFSSL (Cloudflare's PKI/TLS toolkit) configuration. CFSSL is an internal
	// certificate authority used by Astarte for mutual TLS. By default, CFSSL is
	// deployed automatically (deploy=true).
	// +kubebuilder:validation:Optional
	CFSSL AstarteCFSSLSpec `json:"cfssl"`
	// Components configures the individual Astarte services (Flow, Housekeeping,
	// RealmManagement, Pairing, DataUpdaterPlant, AppengineAPI, TriggerEngine, Dashboard).
	// Each component can override its image, resources, replicas, and other settings.
	// +kubebuilder:validation:Optional
	Components AstarteComponentsSpec `json:"components"`
	// AstarteInstanceID is the unique ID that is associated with an Astarte instance. This parameter
	// is used to let different Astarte instances employ a shared database infrastructure.
	// Once set, the AstarteInstanceID cannot be changed. Defaults to "".
	// +kubebuilder:validation:Pattern:=`^[a-z]?[a-z0-9]{0,47}$`
	// +kubebuilder:default:=""
	// +kubebuilder:validation:Optional
	AstarteInstanceID string `json:"astarteInstanceID,omitempty"`
	// ManualMaintenanceMode pauses all reconciliation activities but still computes the resource
	// status. It should be used only when the managed Astarte resources requires manual intervention
	// and the Operator cannot break out of the problem by itself. Do not set this field unless you
	// know exactly what you are doing.
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	ManualMaintenanceMode bool `json:"manualMaintenanceMode,omitempty"`
}

// AstarteStatus defines the observed state of Astarte.
// The Operator updates this subresource as it reconciles the CR, providing
// visibility into the current reconciliation phase, cluster health, and
// connection endpoints.
type AstarteStatus struct {
	// The current reconciliation phase of the Astarte resource.
	ReconciliationPhase ReconciliationPhase `json:"phase"`
	// The Astarte version currently deployed.
	AstarteVersion string `json:"astarteVersion"`
	// The version of the Astarte Operator managing this resource.
	OperatorVersion string `json:"operatorVersion"`
	// The overall health status of the Astarte cluster (red, yellow, or green).
	Health AstarteClusterHealth `json:"health"`
	// The base URL for Astarte API endpoints (derived from spec.api.host and SSL setting).
	BaseAPIURL string `json:"baseAPIURL"`
	// The broker URL for MQTT connections (derived from spec.vernemq.host and port).
	BrokerURL string `json:"brokerURL"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Operator Version",type=string,JSONPath=`.status.operatorVersion`
// +kubebuilder:printcolumn:name="Astarte Version",type=string,JSONPath=`.status.astarteVersion`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.health`
// +kubebuilder:printcolumn:name="Base API URL",type=string,JSONPath=`.status.baseAPIURL`,priority=1
// +kubebuilder:printcolumn:name="Broker URL",type=string,JSONPath=`.status.brokerURL`,priority=1
// Astarte is the Schema for the astartes API
//
// **Custom Astarte annotations**
// Astarte support a set of custom annotations that can be used to
// toggle custom behaviors that are not directly supported by the CRD schema.
// This is often the case for features that are still experimental,
// or that are not expected to be widely used, and that would therefore
// add unnecessary complexity to the CRD schema.
//
// Enable or disable the Astarte Dashboard sidebar
// - Annotation: `api.astarte-platform.org/hide-dashboard-sidebar`
// - Values: `"true"` or `"false"`
type Astarte struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AstarteSpec   `json:"spec,omitempty"`
	Status AstarteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AstarteList contains a list of Astarte
type AstarteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Astarte `json:"items"`
}

// AstarteClusterHealth represents the overall health of the cluster
type AstarteClusterHealth string

const (
	// AstarteClusterHealthRed means the cluster is experiencing serious malfunctions or is down
	AstarteClusterHealthRed AstarteClusterHealth = "red"
	// AstarteClusterHealthYellow means the cluster is experiencing downtimes related to a single service
	AstarteClusterHealthYellow AstarteClusterHealth = "yellow"
	// AstarteClusterHealthGreen means the cluster is healthy, up and running
	AstarteClusterHealthGreen AstarteClusterHealth = "green"
)

// AstarteResourceEvent represents a v1.Event reason for various events. They are all stated
// in this enum to ease understanding and as a reference to users.
type AstarteResourceEvent string

const (
	// AstarteResourceEventInconsistentVersion means the requested Astarte version is inconsistent or unexpected
	AstarteResourceEventInconsistentVersion AstarteResourceEvent = "ErrInconsistentVersion"
	// AstarteResourceEventUnsupportedVersion means the requested Astarte version is not supported by the Operator
	AstarteResourceEventUnsupportedVersion AstarteResourceEvent = "ErrUnsupportedVersion"
	// AstarteResourceEventMigration means the current Astarte Resource will be migrated from a previous one
	AstarteResourceEventMigration AstarteResourceEvent = "Migration"
	// AstarteResourceEventReconciliationFailed means there was a temporary failure in resource Reconciliation
	AstarteResourceEventReconciliationFailed AstarteResourceEvent = "ErrReconcile"
	// AstarteResourceEventCriticalError represents an unrecoverable error which requires manual intervention on the cluster
	AstarteResourceEventCriticalError AstarteResourceEvent = "ErrCritical"
	// AstarteResourceEventStatus represents a generic Status event - in common situations, this is the most common event type
	AstarteResourceEventStatus AstarteResourceEvent = "Status"
	// AstarteResourceEventUpgrade represents an event happening during a Cluster Upgrade
	AstarteResourceEventUpgrade AstarteResourceEvent = "Upgrade"
	// AstarteResourceEventUpgradeError represents an error happening during a Cluster Upgrade
	AstarteResourceEventUpgradeError AstarteResourceEvent = "ErrUpgrade"
)

func (e AstarteResourceEvent) String() string {
	return string(e)
}

// ReconciliationPhase describes the reconciliation phase the Resource is in
type ReconciliationPhase string

const (
	// ReconciliationPhaseUnknown represents an Unknown Phase of the Resource. When in this state, it might
	// have never been reconciled
	ReconciliationPhaseUnknown ReconciliationPhase = ""
	// ReconciliationPhaseReconciling means the Resource is currently in the process of being reconciled
	ReconciliationPhaseReconciling ReconciliationPhase = "Reconciling"
	// ReconciliationPhaseUpgrading means the Resource is currently in the process of being upgraded to a new Astarte version.
	// When successful, the Resource will transition to ReconciliationPhaseReconciling
	ReconciliationPhaseUpgrading ReconciliationPhase = "Upgrading"
	// ReconciliationPhaseReconciled means the Resource is currently reconciled and stable. The resource should stay in this
	// state for most of the time.
	ReconciliationPhaseReconciled ReconciliationPhase = "Reconciled"
	// ReconciliationPhaseManualMaintenanceMode means the Resource is currently not being reconciled as the resource is in
	// Manual Maintenance Mode. This happens only when the user explicitly requires that.
	ReconciliationPhaseManualMaintenanceMode ReconciliationPhase = "Disabled, in Manual Maintenance Mode"
	// ReconciliationPhaseFailed means the Resource failed to reconcile. If this state persists, a manual intervention
	// might be necessary.
	ReconciliationPhaseFailed ReconciliationPhase = "Failed"
)

func (p *ReconciliationPhase) String() string {
	return string(*p)
}

// AstarteComponent describes an internal Astarte Component
type AstarteComponent string

const (
	// AppEngineAPI represents Astarte AppEngine API
	AppEngineAPI AstarteComponent = "appengine_api"
	// DataUpdaterPlant represents Astarte Data Updater Plant
	DataUpdaterPlant AstarteComponent = "data_updater_plant"
	// FlowComponent represents Astarte Flow
	FlowComponent AstarteComponent = "flow"
	// Housekeeping represents Astarte Housekeeping
	Housekeeping AstarteComponent = "housekeeping"
	// Pairing represents Astarte Pairing
	Pairing AstarteComponent = "pairing"
	// RealmManagement represents Astarte Realm Management
	RealmManagement AstarteComponent = "realm_management"
	// TriggerEngine represents Astarte Trigger Engine
	TriggerEngine AstarteComponent = "trigger_engine"
	// Dashboard represents Astarte Dashboard
	Dashboard AstarteComponent = "dashboard"
)

func (a *AstarteComponent) String() string {
	return string(*a)
}

// DashedString returns the Astarte Component in a Kubernetes-friendly format,
// e.g: data-updater-plant instead of data_updater_plant
func (a *AstarteComponent) DashedString() string {
	return strings.ReplaceAll(a.String(), "_", "-")
}

// DockerImageName returns the Docker Image name for this Astarte Component
func (a *AstarteComponent) DockerImageName() string {
	if *a == Dashboard {
		return "astarte-dashboard"
	}
	return "astarte_" + a.String()
}

// ServiceName returns the Kubernetes Service Name associated to this Astarte component.
func (a *AstarteComponent) ServiceName() string {
	return a.DashedString()
}

// ServiceRelativePath returns the relative path where the service will be served by the Ingress
// This will return a meaningful value only for API components or the Dashboard.
func (a *AstarteComponent) ServiceRelativePath() string {
	ret := strings.ReplaceAll(a.DashedString(), "-", "")
	return strings.ReplaceAll(ret, "api", "")
}

// AstarteGenericClusteredResource is a base struct shared by all Astarte components
// that are deployed as either a Deployment or StatefulSet. It provides common
// configuration options such as replicas, affinity, probes, resources, and
// autoscaling that apply uniformly across components.
type AstarteGenericClusteredResource struct {
	// When true, the component is deployed. When false, the component is removed (if
	// already present) or skipped. All components default to true except Flow, which
	// defaults to false and must be explicitly enabled.
	// +kubebuilder:validation:Optional
	Deploy *bool `json:"deploy,omitempty"`
	// The number of replicas for this component.
	// +kubebuilder:validation:Optional
	Replicas *int32 `json:"replicas,omitempty"`
	// When true, pods of this component are spread across nodes using
	// podAntiAffinity.
	// +kubebuilder:validation:Optional
	AntiAffinity *bool `json:"antiAffinity,omitempty"`
	// Custom affinity rules for this component. When set, overrides the default
	// antiAffinity configuration entirely.
	// +kubebuilder:validation:Optional
	CustomAffinity *v1.Affinity `json:"customAffinity,omitempty"`
	// The deployment strategy for this specific component. Overrides the global
	// spec.deploymentStrategy. Note that DataUpdaterPlant, TriggerEngine, and Flow
	// always use Recreate regardless of this setting.
	// +kubebuilder:validation:Optional
	DeploymentStrategy *appsv1.DeploymentStrategy `json:"deploymentStrategy,omitempty"`
	// The Astarte version (image tag) for this specific component. Overrides the
	// global spec.version. Useful for pinning a component to a different version
	// during upgrades or debugging.
	// +kubebuilder:validation:Optional
	Version string `json:"version,omitempty"`
	// The full container image reference (registry/name:tag) for this component.
	// When set, overrides both the distributionChannel and version settings for
	// this component.
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`
	// The image pull policy for this component. Overrides the global
	// spec.imagePullPolicy. Default: inherits from spec.imagePullPolicy.
	// +kubebuilder:validation:Optional
	ImagePullPolicy *v1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Additional image pull secrets for this component's pods. These are appended
	// to the global spec.imagePullSecrets.
	// +kubebuilder:validation:Optional
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Compute Resources for this Component.
	// +kubebuilder:validation:Optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
	// Additional environment variables for this Component. These are appended after
	// the operator-managed environment variables and can override them.
	// +kubebuilder:validation:Optional
	AdditionalEnv []v1.EnvVar `json:"additionalEnv,omitempty"`
	// Additional labels for this Component's pod(s).
	// Label keys can't be of the form "app", "component", "astarte-*", "flow-*"
	// +kubebuilder:validation:Optional
	PodLabels map[string]string `json:"podLabels,omitempty"`
	// Autoscaling resources for this deployment/statefulset.
	// If autoscaling is enabled, this will take precedence over the "Replicas" field.
	// The autoscaling feature must be enabled globally via features.autoscaling.
	// +kubebuilder:validation:Optional
	Autoscale *AstarteGenericClusteredResourceAutoscalerSpec `json:"autoscaler,omitempty"`
	// The PriorityClass for this component.
	// Must be one of "high", "mid", "low" or unspecified.
	// Ignored if astartePodPriorities is not enabled.
	// +kubebuilder:validation:Enum:=high;mid;low;""
	// +kubebuilder:validation:Optional
	PriorityClass string `json:"priorityClass,omitempty"`
	// Override the default Liveness probe for this component.
	// If not set, a default HTTP GET probe is configured to check the /health endpoint on the http port.
	// Default settings: InitialDelaySeconds=10, TimeoutSeconds=5, PeriodSeconds=30, FailureThreshold=5 (15 for Housekeeping).
	// Note: VerneMQ uses different defaults: /metrics endpoint on port 8888, InitialDelaySeconds=60, PeriodSeconds=20, FailureThreshold=3.
	// +kubebuilder:validation:Optional
	LivenessProbe *v1.Probe `json:"livenessProbe,omitempty"`
	// Override the default Readiness probe for this component.
	// If not set, a default HTTP GET probe is configured to check the /health endpoint on the http port.
	// Default settings: InitialDelaySeconds=10, TimeoutSeconds=5, PeriodSeconds=30, FailureThreshold=5 (15 for Housekeeping).
	// Note: VerneMQ uses different defaults: /metrics endpoint on port 8888, InitialDelaySeconds=60, PeriodSeconds=20, FailureThreshold=3.
	// +kubebuilder:validation:Optional
	ReadinessProbe *v1.Probe `json:"readinessProbe,omitempty"`
	// Override the default Startup probe for this component.
	// If not set, no startup probe is configured by default.
	// +kubebuilder:validation:Optional
	StartupProbe *v1.Probe `json:"startupProbe,omitempty"`
}

// AstarteGenericClusteredResourceAutoscalerSpec configures autoscaling for an
// Astarte component. Currently only horizontal autoscaling is supported.
type AstarteGenericClusteredResourceAutoscalerSpec struct {
	// Name of the HorizontalPodAutoscaler for this deployment/statefulset.
	// This will take precedence over the "Replicas" field of the parent Astarte component.
	// The HPA resource must exist in the same namespace as Astarte and the
	// features.autoscaling flag must be enabled.
	// +kubebuilder:validation:Optional
	Horizontal string `json:"horizontal,omitempty"`
	// TODO: Vertical string `json:"vertical,omitempty"`
}

// AstartePersistentStorageSpec configures persistent storage for Astarte components
// that require it (VerneMQ, CFSSL).
type AstartePersistentStorageSpec struct {
	// The size of the persistent volume. When not set, the Operator uses a
	// sensible default size.
	// +kubebuilder:validation:Optional
	Size *resource.Quantity `json:"size"`
	// The storage class name for the persistent volume. When not set, the
	// global spec.storageClassName is used as fallback.
	// +kubebuilder:validation:Optional
	ClassName string `json:"className,omitempty"`
	// A complete volume definition that replaces the Operator-managed
	// persistent volume claim. Use this to reference an externally-managed
	// volume. When set, size and className are ignored.
	// +kubebuilder:validation:Optional
	VolumeDefinition *v1.Volume `json:"volumeDefinition,omitempty"`
}

type AstarteAPISpec struct {
	// Enable or disable SSL for the Astarte API. Default: true.
	// +kubebuilder:validation:Optional
	SSL  *bool  `json:"ssl,omitempty"`
	Host string `json:"host"`
}

// HostAndPort represents a network endpoint with a hostname and port.
type HostAndPort struct {
	// The hostname or IP address of the service.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Host string `json:"host"`
	// The port number the service listens on.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:Required
	Port *int32 `json:"port"`
}

// LoginCredentialsSecret references a Kubernetes Secret containing login credentials
// (username and password) for connecting to an external service.
type LoginCredentialsSecret struct {
	// The name of the Kubernetes Secret.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// The key within the Secret that holds the username.
	// +kubebuilder:validation:MinLength=1
	UsernameKey string `json:"usernameKey"`
	// The key within the Secret that holds the password.
	// +kubebuilder:validation:MinLength=1
	PasswordKey string `json:"passwordKey"`
}

// ConnectionStringSecret references a Kubernetes Secret containing a connection string
// (e.g. a full URL or DSN) for an external service.
type ConnectionStringSecret struct {
	// The name of the Kubernetes Secret.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// The key within the Secret that holds the connection string.
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
}

// GenericConnectionSpec holds common connection configuration for external services
// (Cassandra, RabbitMQ, Vault). It supports either username/password credentials
// or a raw connection string.
type GenericConnectionSpec struct {
	// SSL configuration for the connection.
	// +kubebuilder:validation:Optional
	SSLConfiguration GenericSSLConfigurationSpec `json:"sslConfiguration,omitempty"`
	// The secret containing Username and Password to login.
	// Either this field or `connectionStringSecret` must be set.
	// +kubebuilder:validation:Optional
	CredentialsSecret *LoginCredentialsSecret `json:"credentialsSecret,omitempty"`
	// The secret containing a connection string to the service.
	// Either this field or `credentialsSecret` must be set.
	// TODO: currently, Astarte services do not allow the connection string to be
	// put as-is in the env. Therefore, setting this field is a no-op.
	// Not using `credentialsSecret` WILL break your Astarte instance.
	// +kubebuilder:validation:Optional
	ConnectionStringSecret *ConnectionStringSecret `json:"connectionStringSecret,omitempty"`
}

// GenericSSLConfigurationSpec configures SSL/TLS for connections to external services.
type GenericSSLConfigurationSpec struct {
	// When true, enable SSL for the connection. Default: false.
	// +kubebuilder:validation:Optional
	Enable bool `json:"enable,omitempty"`
	// A reference to a Kubernetes Secret containing a custom CA certificate
	// to validate the server's TLS certificate. The Secret must be in the
	// same namespace as the Astarte resource.
	// +kubebuilder:validation:Optional
	CustomCASecret v1.LocalObjectReference `json:"customCASecret,omitempty"`
	// When false, disable SNI (Server Name Indication) for the connection.
	// Default: true (SNI is enabled).
	// +kubebuilder:validation:Optional
	SNI *bool `json:"sni,omitempty"`
	// A custom SNI hostname to use for the connection. When set, overrides
	// the default hostname-based SNI.
	// +kubebuilder:validation:Optional
	CustomSNI string `json:"customSNI,omitempty"`
}

// AstarteRabbitMQBaseConnectionSpec defines the host, port, credentials, and SSL
// settings for connecting to a RabbitMQ endpoint (AMQP or Management API).
type AstarteRabbitMQBaseConnectionSpec struct {
	HostAndPort `json:",inline"`
	// Credentials and SSL configuration for the RabbitMQ connection.
	// +kubebuilder:validation:Optional
	GenericConnectionSpec `json:",inline"`
}

// AstarteRabbitMQConnectionSpec extends the base RabbitMQ connection with an
// optional virtual host for the AMQP connection.
type AstarteRabbitMQConnectionSpec struct {
	AstarteRabbitMQBaseConnectionSpec `json:",inline"`
	// The virtual host for the RabbitMQ AMQP connection. Default: "/".
	// +kubebuilder:validation:Optional
	VirtualHost string `json:"virtualHost,omitempty"`
}

// AstarteRabbitMQSpec defines the RabbitMQ configuration for Astarte.
// Both the AMQP connection and the Management API connection are required.
type AstarteRabbitMQSpec struct {
	// RabbitMQ AMQP connection details. Required.
	// +kubebuilder:validation:Required
	Connection *AstarteRabbitMQConnectionSpec `json:"connection,omitempty"`
	// RabbitMQ management APIs connection details. Required.
	// +kubebuilder:validation:Required
	ManagementConnection *AstarteRabbitMQBaseConnectionSpec `json:"managementConnection,omitempty"`
	// Configures the data queues prefix on RabbitMQ. You should change this setting only
	// in custom RabbitMQ installations.
	// +kubebuilder:validation:Optional
	DataQueuesPrefix string `json:"dataQueuesPrefix,omitempty"`
	// Configures the events exchange name on RabbitMQ. You should change this setting only
	// in custom RabbitMQ installations.
	// +kubebuilder:validation:Optional
	EventsExchangeName string `json:"eventsExchangeName,omitempty"`
}

// AstarteCassandraConnectionSpec defines the connection to an external Cassandra/ScyllaDB cluster.
type AstarteCassandraConnectionSpec struct {
	GenericConnectionSpec `json:",inline"`
	// The list of Cassandra/ScyllaDB seed nodes. At least one node must be provided.
	Nodes []HostAndPort `json:"nodes,omitempty"`
	// The size of the connection pool to each Cassandra/ScyllaDB node.
	// Adjust this value if you need to increase or limit the number of concurrent
	// queries per node.
	// +kubebuilder:validation:Optional
	PoolSize *int `json:"poolSize,omitempty"`
	// Enable or disable the keepalive option for the xandra connection.
	// Default: true.
	// +kubebuilder:validation:Optional
	EnableKeepalive *bool `json:"enableKeepalive,omitempty"`
}

// AstarteCassandraSpec configures the Cassandra/ScyllaDB backend for Astarte.
type AstarteCassandraSpec struct {
	// The Cassandra/ScyllaDB connection configuration. Required.
	// +kubebuilder:validation:Required
	Connection *AstarteCassandraConnectionSpec `json:"connection,omitempty"`
	// The keyspace configuration for Astarte's system keyspace. Required.
	// +kubebuilder:validation:Required
	AstarteSystemKeyspace AstarteSystemKeyspaceSpec `json:"astarteSystemKeyspace"`
}

// AstarteVerneMQSpec configures the VerneMQ MQTT broker component.
type AstarteVerneMQSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// The host and port for VerneMQ brokers. Required.
	HostAndPort `json:",inline"`
	// The name of a Kubernetes Secret containing the CA certificate for VerneMQ
	// internal TLS communication (for Astarte >= 1.2). The Secret must be in
	// the same namespace as the Astarte resource.
	// +kubebuilder:validation:Optional
	CaSecret string `json:"caSecret,omitempty"`
	// Persistent storage configuration for VerneMQ. If not set, the default
	// storage size and class (from spec.storageClassName) are used.
	// +kubebuilder:validation:Optional
	Storage *AstartePersistentStorageSpec `json:"storage,omitempty"`
	// Controls the device heartbeat from the broker to Astarte. The heartbeat is sent periodically
	// to prevent Astarte from keeping up stale connections from Devices in case the broker misbehaves
	// and does not send disconnection events. You should usually not tweak this value. Moreover, keep
	// in mind that when a lot of devices are connected simultaneously, having a short heartbeat time
	// might cause performance issues. When not set, no heartbeat env var is passed and the VerneMQ
	// container default (1 hour) is used.
	// +kubebuilder:validation:Optional
	DeviceHeartbeatSeconds int `json:"deviceHeartbeatSeconds,omitempty"`
	// The maximum number of QoS 1 or 2 messages to hold in the offline queue.
	// Defaults to 1000000. Set to -1 for no maximum (not recommended). Set to 0
	// if no messages should be stored offline.
	// +kubebuilder:validation:Optional
	MaxOfflineMessages *int `json:"maxOfflineMessages,omitempty"`
	// This option allows persistent clients ( = clean session set to
	// false) to be removed if they do not reconnect within 'persistent_client_expiration'.
	// This is a non-standard option. As far as the MQTT specification is concerned,
	// persistent clients persist forever.
	// The expiration period should be an integer followed by one of 'd', 'w', 'm', 'y' for
	// day, week, month, and year.
	// Default: 1 year
	// +kubebuilder:validation:Optional
	PersistentClientExpiration string `json:"persistentClientExpiration,omitempty"`
	// Configures the mirror queue for VerneMQ. When set, all MQTT messages are
	// forwarded to the specified queue for audit/logging purposes. Leave empty
	// unless you have a specific mirror queue setup.
	// +kubebuilder:validation:Optional
	MirrorQueue string `json:"mirrorQueue,omitempty"`
	// This option allows, when true, to handle SSL termination at VerneMQ level.
	// Default: false
	// +kubebuilder:validation:Optional
	SSLListener *bool `json:"sslListener,omitempty"`
	// Reference the name of the secret containing the TLS certificate for VerneMQ.
	// The secret must be present in the same namespace in which Astarte resides.
	// The field will be used only if SSLListener is set to true.
	// +kubebuilder:validation:Optional
	SSLListenerCertSecretName string `json:"sslListenerCertSecretName,omitempty"`
}

// AstarteDataUpdaterPlantSpec configures the Data Updater Plant (DUP) component,
// which handles data ingestion from the MQTT broker into the database.
type AstarteDataUpdaterPlantSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// Controls the number of data queues used by the Data Updater Plant.
	// This corresponds to the AMQP queues from which DUP consumers pull data.
	// Defaults to 128. You should change this only when fine-tuning a
	// custom RabbitMQ setup.
	// +kubebuilder:validation:Optional
	DataQueueCount *int `json:"dataQueueCount,omitempty"`
	// Controls the prefetch count for Data Updater Plant. When fine-tuning Astarte, this parameter
	// can make a difference for what concerns Data Updater Plant ingestion performance. However,
	// it can also degrade performance significantly and/or increase risk of data loss when misconfigured.
	// Configure this value only if you know what you're doing and you have experience with RabbitMQ.
	// Defaults to 300
	// +kubebuilder:validation:Optional
	PrefetchCount *int `json:"prefetchCount,omitempty"`
}

// AstarteTriggerEngineSpec configures the Trigger Engine component, which processes
// Astarte triggers (user-defined rules) and dispatches events.
type AstarteTriggerEngineSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// Configures the name of the Events queue. Should be configured only in installations with a highly
	// customized RabbitMQ. It is advised to leave empty unless you know exactly what you're doing.
	// +kubebuilder:validation:Optional
	EventsQueueName string `json:"eventsQueueName,omitempty"`
	// Configures the routing key for Trigger Events. Should be configured only in installations
	// with a highly customized RabbitMQ and a custom Trigger Engine setup. It is advised to leave
	// empty unless you know exactly what you're doing, misconfiguring this value can cause heavy
	// breakage within Trigger Engine.
	// +kubebuilder:validation:Optional
	EventsRoutingKey string `json:"eventsRoutingKey,omitempty"`
}

// AstarteAppengineAPISpec configures the AppEngine API component.
type AstarteAppengineAPISpec struct {
	AstarteGenericAPIComponentSpec `json:",inline"`
	// The maximum number of results returned by a single AppEngine API query.
	// Must be at least 100. Defaults to 10000.
	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Optional
	MaxResultsLimit *int `json:"maxResultsLimit,omitempty"`
	// Configures the name of the Room Events queue. Should be configured only in installations with a highly
	// customized RabbitMQ. It is advised to leave empty unless you know exactly what you're doing.
	// +kubebuilder:validation:Optional
	RoomEventsQueueName string `json:"roomEventsQueueName,omitempty"`
	// Configures the name of the Room Events exchange. Should be configured only in installations with a highly
	// customized RabbitMQ. It is advised to leave empty unless you know exactly what you're doing.
	// +kubebuilder:validation:Optional
	RoomEventsExchangeName string `json:"roomEventsExchangeName,omitempty"`
}

// AstarteDashboardConfigAuthSpec defines an authentication provider configuration
// for the Astarte Dashboard.
type AstarteDashboardConfigAuthSpec struct {
	// The authentication type (e.g. "token", "oauth").
	Type string `json:"type"`
	// The OAuth API URL (only used when type is "oauth").
	// +kubebuilder:validation:Optional
	OAuthAPIURL string `json:"oauth_api_url,omitempty"`
}

// AstarteDashboardConfigSpec configures the Astarte Dashboard UI settings.
type AstarteDashboardConfigSpec struct {
	// The URL of the Realm Management API. When set, overrides the default
	// (derived from the Astarte API host).
	// +kubebuilder:validation:Optional
	RealmManagementAPIURL string `json:"realmManagementApiUrl,omitempty"`
	// The URL of the AppEngine API. When set, overrides the default.
	// +kubebuilder:validation:Optional
	AppEngineAPIURL string `json:"appEngineApiUrl,omitempty"`
	// The URL of the Pairing API. When set, overrides the default.
	// +kubebuilder:validation:Optional
	PairingAPIURL string `json:"pairingApiUrl,omitempty"`
	// The URL of the Flow API. When set, overrides the default.
	// +kubebuilder:validation:Optional
	FlowAPIURL string `json:"flowApiUrl,omitempty"`
	// The default realm for the Dashboard. On first access, the Dashboard
	// will pre-select this realm.
	// +kubebuilder:validation:Optional
	DefaultRealm string `json:"defaultRealm,omitempty"`
	// The default authentication method for the Dashboard. Default: "token".
	// +kubebuilder:validation:Optional
	DefaultAuth string `json:"defaultAuth,omitempty"`
	// Authentication provider configurations available in the Dashboard.
	// If not set, defaults to a single "token" auth provider.
	// +kubebuilder:validation:Optional
	Auth []AstarteDashboardConfigAuthSpec `json:"auth,omitempty"`
}

// AstarteDashboardSpec configures the Astarte Dashboard component.
type AstarteDashboardSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// Dashboard-specific UI and API configuration.
	// +kubebuilder:validation:Optional
	AstarteDashboardConfigSpec `json:",inline"`
}

// AstarteGenericAPIComponentSpec extends the base clustered resource with an
// option to disable authentication on the component. Used by all Astarte API
// components (Flow, Housekeeping, RealmManagement, Pairing, AppengineAPI).
type AstarteGenericAPIComponentSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// When true, disables authentication for this API component. This is useful
	// in development or when an external auth proxy is used. Do not disable
	// authentication in production without an external auth mechanism.
	// +kubebuilder:validation:Optional
	DisableAuthentication *bool `json:"disableAuthentication,omitempty"`
}

// AstarteComponentsSpec configures all Astarte service components.
type AstarteComponentsSpec struct {
	// Compute Resources shared across all components. Can be overridden
	// per-component by setting the component's own resources field.
	// +kubebuilder:validation:Optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
	// Flow component configuration. Defaults to deploy: false.
	// +kubebuilder:validation:Optional
	Flow AstarteGenericAPIComponentSpec `json:"flow,omitempty"`
	// Housekeeping component configuration.
	// +kubebuilder:validation:Optional
	Housekeeping AstarteGenericAPIComponentSpec `json:"housekeeping,omitempty"`
	// RealmManagement component configuration.
	// +kubebuilder:validation:Optional
	RealmManagement AstarteGenericAPIComponentSpec `json:"realmManagement,omitempty"`
	// Pairing component configuration.
	// +kubebuilder:validation:Optional
	Pairing AstarteGenericAPIComponentSpec `json:"pairing,omitempty"`
	// DataUpdaterPlant component configuration.
	// +kubebuilder:validation:Optional
	DataUpdaterPlant AstarteDataUpdaterPlantSpec `json:"dataUpdaterPlant,omitempty"`
	// AppengineAPI component configuration.
	// +kubebuilder:validation:Optional
	AppengineAPI AstarteAppengineAPISpec `json:"appengineApi,omitempty"`
	// TriggerEngine component configuration.
	// +kubebuilder:validation:Optional
	TriggerEngine AstarteTriggerEngineSpec `json:"triggerEngine,omitempty"`
	// Dashboard component configuration.
	// +kubebuilder:validation:Optional
	Dashboard AstarteDashboardSpec `json:"dashboard,omitempty"`
}

// AstarteCFSSLDBConfigSpec configures the database backend used by CFSSL.
type AstarteCFSSLDBConfigSpec struct {
	// The database driver (e.g. "sqlite3", "postgres").
	Driver string `json:"driver,omitempty"`
	// The database data source name (connection string).
	DataSource string `json:"dataSource,omitempty"`
}

// AstarteCFSSLCSRRootCAKeySpec defines the key algorithm and size for the CFSSL root CA.
type AstarteCFSSLCSRRootCAKeySpec struct {
	// The key algorithm (e.g. "rsa", "ecdsa").
	Algo string `json:"algo"`
	// The key size in bits (e.g. 2048, 4096).
	Size int `json:"size"`
}

// AstarteCFSSLCSRRootCANamesSpec defines the distinguished name components
// for the CFSSL root CA certificate.
type AstarteCFSSLCSRRootCANamesSpec struct {
	C  string `json:"C"`
	L  string `json:"L"`
	O  string `json:"O"`
	OU string `json:"OU"`
	ST string `json:"ST"`
}

// AstarteCFSSLCSRRootCASpec defines the certificate signing request for the CFSSL root CA.
type AstarteCFSSLCSRRootCASpec struct {
	CN     string                           `json:"CN"`
	Key    *AstarteCFSSLCSRRootCAKeySpec    `json:"key"`
	Names  []AstarteCFSSLCSRRootCANamesSpec `json:"names"`
	Expiry string                           `json:"expiry"`
}

// AstarteCFSSLCARootConfigSigningCAConstraintSpec defines constraints on the CA certificate.
type AstarteCFSSLCARootConfigSigningCAConstraintSpec struct {
	MaxPathLen     int  `json:"max_path_len"`
	IsCA           bool `json:"is_ca"`
	MaxPathLenZero bool `json:"max_path_len_zero"`
}

// AstarteCFSSLCARootConfigSigningDefaultSpec defines the default signing parameters
// for the CFSSL CA.
type AstarteCFSSLCARootConfigSigningDefaultSpec struct {
	Usages       []string                                         `json:"usages"`
	Expiry       string                                           `json:"expiry"`
	CAConstraint *AstarteCFSSLCARootConfigSigningCAConstraintSpec `json:"ca_constraint"`
}

// AstarteCFSSLCARootConfigSpec defines the root CA configuration for CFSSL.
type AstarteCFSSLCARootConfigSpec struct {
	SigningDefault *AstarteCFSSLCARootConfigSigningDefaultSpec `json:"signingDefault"`
}

// AstarteCFSSLSpec configures CFSSL (Cloudflare's PKI/TLS toolkit), the internal
// certificate authority used by Astarte for mutual TLS between components.
// By default, CFSSL is deployed and managed by the Operator.
type AstarteCFSSLSpec struct {
	// When true, deploy CFSSL. When false, an external CFSSL instance must be
	// provided via the url field. Default: true.
	// +kubebuilder:validation:Optional
	Deploy *bool `json:"deploy,omitempty"`
	// The URL of an external CFSSL instance. Used only when deploy is false.
	// +kubebuilder:validation:Optional
	URL string `json:"url,omitempty"`
	// The expiry duration for the CA certificate (e.g. "87600h"). Only applies
	// when CFSSL is deployed by the Operator.
	// +kubebuilder:validation:Optional
	CaExpiry string `json:"caExpiry,omitempty"`
	// A reference to a Kubernetes Secret containing an externally-managed CA
	// certificate. The Secret must be in the same namespace as Astarte.
	// When set, CFSSL uses this CA instead of generating a new one.
	// +kubebuilder:validation:Optional
	CASecret v1.LocalObjectReference `json:"caSecret,omitempty"`
	// The expiry duration for certificates issued by CFSSL (e.g. "8760h").
	// +kubebuilder:validation:Optional
	CertificateExpiry string `json:"certificateExpiry,omitempty"`
	// Database configuration for CFSSL's certificate storage.
	// +kubebuilder:validation:Optional
	DBConfig *AstarteCFSSLDBConfigSpec `json:"dbConfig,omitempty"`
	// Compute Resources for this Component.
	// +kubebuilder:validation:Optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
	// The CFSSL container version (image tag) to use. Overrides the global
	// spec.version for CFSSL specifically.
	// +kubebuilder:validation:Optional
	Version string `json:"version,omitempty"`
	// The full CFSSL container image reference. Overrides the distributionChannel
	// and version for CFSSL.
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`
	// Persistent storage configuration for CFSSL's certificate database.
	// +kubebuilder:validation:Optional
	Storage *AstartePersistentStorageSpec `json:"storage,omitempty"`
	// Certificate signing request configuration for the root CA.
	// +kubebuilder:validation:Optional
	CSRRootCa *AstarteCFSSLCSRRootCASpec `json:"csrRootCa,omitempty"`
	// Root CA configuration for signing certificates.
	// +kubebuilder:validation:Optional
	CARootConfig *AstarteCFSSLCARootConfigSpec `json:"caRootConfig,omitempty"`
	// Additional labels for this Component's pod(s).
	// Label keys can't be of the form "app", "component", "astarte-*", "flow-*"
	// +kubebuilder:validation:Optional
	PodLabels map[string]string `json:"podLabels,omitempty"`
	// The PriorityClass for this component.
	// Must be one of "high", "mid", "low" or unspecified.
	// Ignored if astartePodPriorities is not enabled.
	// +kubebuilder:validation:Enum:=high;mid;low;""
	// +kubebuilder:validation:Optional
	PriorityClass string `json:"priorityClass,omitempty"`
	// Override the default Liveness probe for CFSSL.
	// If not set, a default HTTP GET probe is configured to check the /api/v1/cfssl/health endpoint on the http port.
	// Default settings: InitialDelaySeconds=10, TimeoutSeconds=5, PeriodSeconds=20, FailureThreshold=3.
	// +kubebuilder:validation:Optional
	LivenessProbe *v1.Probe `json:"livenessProbe,omitempty"`
	// Override the default Readiness probe for CFSSL.
	// If not set, a default HTTP GET probe is configured to check the /api/v1/cfssl/health endpoint on the http port.
	// Default settings: InitialDelaySeconds=10, TimeoutSeconds=5, PeriodSeconds=20, FailureThreshold=3.
	// +kubebuilder:validation:Optional
	ReadinessProbe *v1.Probe `json:"readinessProbe,omitempty"`
	// Override the default Startup probe for CFSSL.
	// If not set, no startup probe is configured by default.
	// +kubebuilder:validation:Optional
	StartupProbe *v1.Probe `json:"startupProbe,omitempty"`
}

// This interface is implemented by all Astarte components which have a podLabels field.
// +k8s:deepcopy-gen=false
type PodLabelsGetter interface {
	GetPodLabels() map[string]string
}

func (r AstarteGenericClusteredResource) GetPodLabels() map[string]string {
	return r.PodLabels
}

func (r AstarteCFSSLSpec) GetPodLabels() map[string]string {
	return r.PodLabels
}

// AstarteSystemKeyspaceSpec configures the ScyllaDB/Cassandra keyspace for Astarte.
//
// By configuring these fields, you control the replication strategy, replication factor, and (for multi-datacenter
// deployments) the replica distribution per datacenter. These settings take effect only upon keyspace creation.
//
// Fields:
//   - ReplicationStrategy chooses the replication strategy for the keyspace.
//   - ReplicationFactor (for SimpleStrategy or for default replication factor with NetworkTopologyStrategy).
//   - DataCenterReplication (for flexible NetworkTopologyStrategy configurations).
//
// These fields must be set at the first apply of the CR and cannot be changed later on,
// for this reason no default is provided: the user shall make a conscious choice.
type AstarteSystemKeyspaceSpec struct {
	// ReplicationStrategy specifies the Cassandra/ScyllaDB replication strategy for the keyspace.
	// Must be either "SimpleStrategy" or "NetworkTopologyStrategy" (for production deployments and/or
	// multi-datacenter deployments).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=SimpleStrategy;NetworkTopologyStrategy
	ReplicationStrategy string `json:"replicationStrategy"`
	// ReplicationFactor sets the total number of replicas for the keyspace when using SimpleStrategy.
	// Must be at least 1. Must be odd. Defaults to 1.
	// Shall be set if and only if replicationStrategy is SimpleStrategy (checked with Admission Webhooks).
	// This field is ignored if ReplicationStrategy is set to NetworkTopologyStrategy.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	ReplicationFactor *int `json:"replicationFactor,omitempty"`
	// DataCenterReplication specifies custom replication factors per datacenter when using NetworkTopologyStrategy.
	// If set, this string must be a comma-separated list of <DataCenter>:<ReplicationFactor> entries
	// (e.g., "dc1:3,dc2:5"). <ReplicationFactor> must be odd.
	// Shall be set if and only if replicationStrategy is NetworkTopologyStrategy (checked with Admission Webhooks)
	// This field is ignored if ReplicationStrategy is set to SimpleStrategy.
	// +kubebuilder:validation:Optional
	DataCenterReplication string `json:"dataCenterReplication,omitempty"`
}

// AstartePodPriorities allows to set different priorityClasses for Astarte pods.
// Note that enabling this feature might generate some counter-intuitive
// scheduling beahaviour if not done properly.
type AstartePodPrioritiesSpec struct {
	// +kubebuilder:validation:Optional
	Enable bool `json:"enable,omitempty"`
	// The value of the highest PriorityClass for Astarte pods.
	// Once the value is set, updating it will not have effect.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=1000
	// +kubebuilder:validation:Minimum:=0
	AstarteHighPriority *int `json:"astarteHighPriority,omitempty"`
	// The value of the medium PriorityClass for Astarte pods.
	// Once the value is set, updating it will not have effect.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=100
	// +kubebuilder:validation:Minimum:=0
	AstarteMidPriority *int `json:"astarteMidPriority,omitempty"`
	// The value of the least PriorityClass for Astarte pods.
	// Once the value is set, updating it will not have effect.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=10
	// +kubebuilder:validation:Minimum:=0
	AstarteLowPriority *int `json:"astarteLowPriority,omitempty"`
}

// AstarteRendezvousServerConnectionSpec defines the connection to an FDO Rendezvous Server.
type AstarteRendezvousServerConnectionSpec struct {
	HostAndPort `json:",inline"`
	// SSL configuration for the Rendezvous Server connection.
	// +kubebuilder:validation:Optional
	SSLConfiguration GenericSSLConfigurationSpec `json:"sslConfiguration,omitempty"`
}

// AstarteRendezvousServerSpec configures the FDO Rendezvous Server connection.
type AstarteRendezvousServerSpec struct {
	// The Rendezvous Server connection details.
	// +kubebuilder:validation:Optional
	Connection AstarteRendezvousServerConnectionSpec `json:"connection,omitempty"`
}

// AstarteVaultSpec configures the OpenBao/HashiCorp Vault integration.
// Required for Astarte >= 1.4.0. Ignored for Astarte 1.3.
type AstarteVaultSpec struct {
	// The Vault connection details.
	// +kubebuilder:validation:Optional
	Connection AstarteVaultConnectionSpec `json:"connection,omitempty"`
	// Base vault namespace prefix under which Astarte will create further sub-namespaces
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	BaseNamespace string `json:"baseNamespace,omitempty"`
}

// AstarteVaultConnectionSpec defines the connection to an OpenBao or HashiCorp Vault instance.
type AstarteVaultConnectionSpec struct {
	// The Vault server host and port. If port is not specified, defaults to 8200.
	// +kubebuilder:validation:Optional
	HostAndPort `json:",inline"`
	// SSL configuration for the Vault connection.
	// +kubebuilder:validation:Optional
	SSLConfiguration GenericSSLConfigurationSpec `json:"sslConfiguration,omitempty"`
	// The secret containing a token to login. The Secret must be in the same
	// namespace as the Astarte resource.
	// +kubebuilder:validation:Optional
	ConnectionStringSecret *ConnectionStringSecret `json:"connectionStringSecret,omitempty"`
}

// AstarteFDOSpec configures FDO (FIDO Device Onboarding) support in Astarte.
// Available as an opt-in feature starting from Astarte 1.3. From Astarte 1.4.0
// onwards, FDO is mandatory and cannot be disabled.
type AstarteFDOSpec struct {
	// When true, enable FDO support. For Astarte < 1.4.0, this is optional.
	// For Astarte >= 1.4.0, FDO is always enabled and this field cannot be
	// set to false (validating webhook enforces this).
	// +kubebuilder:validation:Optional
	Enable bool `json:"enable,omitempty"`
	// The FDO Rendezvous Server configuration. Required when FDO is enabled.
	// For Astarte >= 1.4.0, this is always required since FDO cannot be disabled.
	// +kubebuilder:validation:Optional
	RendezvousServer *AstarteRendezvousServerSpec `json:"rendezvousServer,omitempty"`
}

func (a *AstartePodPrioritiesSpec) IsEnabled() bool {
	return a != nil && a.Enable
}

// AstarteFeatures enables/disables selectively a set of global, opt-in features in Astarte.
// All features default to false (disabled) unless explicitly set.
type AstarteFeatures struct {
	// Enable realm deletion support. When enabled, realms can be deleted
	// through the Realm Management API.
	// +kubebuilder:validation:Optional
	RealmDeletion bool `json:"realmDeletion,omitempty"`
	// Enable horizontal pod autoscaling for Astarte components. When enabled,
	// each component's autoscaler configuration (if set) will be used to
	// dynamically scale replicas based on resource utilization. Requires
	// a metrics server to be installed in the cluster.
	// +kubebuilder:validation:Optional
	Autoscaling bool `json:"autoscaling,omitempty"`
	// Configure pod priority classes for Astarte components. When enabled,
	// the Operator creates three PriorityClasses (high, mid, low) and assigns
	// them to components based on their priorityClass field. Note that enabling
	// this feature might generate some counter-intuitive scheduling behaviour
	// if not done properly.
	// +kubebuilder:validation:Optional
	AstartePodPriorities *AstartePodPrioritiesSpec `json:"astartePodPriorities,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Astarte{}, &AstarteList{})
}
