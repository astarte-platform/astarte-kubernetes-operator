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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AstarteSpec defines the desired state of Astarte
type AstarteSpec struct {
	// The Astarte Version for this Resource
	Version string `json:"version"`
	// +kubebuilder:validation:Optional
	Features AstarteFeatures `json:"features,omitempty"`
	// +kubebuilder:validation:Optional
	ImagePullPolicy *v1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// +kubebuilder:validation:Optional
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// +kubebuilder:validation:Optional
	DeploymentStrategy *appsv1.DeploymentStrategy `json:"deploymentStrategy,omitempty"`
	// +kubebuilder:validation:Optional
	StorageClassName string         `json:"storageClassName,omitempty"`
	API              AstarteAPISpec `json:"api"`
	// +kubebuilder:validation:Optional
	RabbitMQ AstarteRabbitMQSpec `json:"rabbitmq"`
	// +kubebuilder:validation:Optional
	Cassandra AstarteCassandraSpec `json:"cassandra"`
	VerneMQ   AstarteVerneMQSpec   `json:"vernemq"`
	// +kubebuilder:validation:Optional
	CFSSL AstarteCFSSLSpec `json:"cfssl"`
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

// AstarteStatus defines the observed state of Astarte
type AstarteStatus struct {
	ReconciliationPhase ReconciliationPhase  `json:"phase"`
	AstarteVersion      string               `json:"astarteVersion"`
	OperatorVersion     string               `json:"operatorVersion"`
	Health              AstarteClusterHealth `json:"health"`
	BaseAPIURL          string               `json:"baseAPIURL"`
	BrokerURL           string               `json:"brokerURL"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Astarte is the Schema for the astartes API
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

// ServiceRelativePath returns the relative path where the service will be served by the Astarte Voyager Ingress.
// This will return a meaningful value only for API components or the Dashboard.
func (a *AstarteComponent) ServiceRelativePath() string {
	if !strings.Contains(a.String(), "api") && a.String() != "dashboard" && a.String() != "flow" {
		return ""
	}
	ret := strings.ReplaceAll(a.DashedString(), "-", "")
	return strings.ReplaceAll(ret, "api", "")
}

type AstarteGenericClusteredResource struct {
	// +kubebuilder:validation:Optional
	Deploy *bool `json:"deploy,omitempty"`
	// +kubebuilder:validation:Optional
	Replicas *int32 `json:"replicas,omitempty"`
	// +kubebuilder:validation:Optional
	AntiAffinity *bool `json:"antiAffinity,omitempty"`
	// +kubebuilder:validation:Optional
	CustomAffinity *v1.Affinity `json:"customAffinity,omitempty"`
	// +kubebuilder:validation:Optional
	DeploymentStrategy *appsv1.DeploymentStrategy `json:"deploymentStrategy,omitempty"`
	// +kubebuilder:validation:Optional
	Version string `json:"version,omitempty"`
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`
	// +kubebuilder:validation:Optional
	ImagePullPolicy *v1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// +kubebuilder:validation:Optional
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Compute Resources for this Component.
	// +kubebuilder:validation:Optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
	// Additional environment variables for this Component
	// +kubebuilder:validation:Optional
	AdditionalEnv []v1.EnvVar `json:"additionalEnv,omitempty"`
	// Additional labels for this Component's pod(s).
	// Label keys can't be of the form "app", "component", "astarte-*", "flow-*"
	// +kubebuilder:validation:Optional
	PodLabels map[string]string `json:"podLabels,omitempty"`
	// Autoscaling resources for this deployment/statefulset.
	// If autoscaling is enabled, this will take precedence over the "Replicas" field.
	// +kubebuilder:validation:Optional
	Autoscale *AstarteGenericClusteredResourceAutoscalerSpec `json:"autoscaler,omitempty"`
	// The PriorityClass for this component.
	// Must be one of "high", "mid", "low" or unspecified.
	// Ignored if astartePodPriorities is not enabled.
	// +kubebuilder:validation:Enum:=high;mid;low;""
	// +kubebuilder:validation:Optional
	PriorityClass string `json:"priorityClass,omitempty"`
}

type AstarteGenericClusteredResourceAutoscalerSpec struct {
	// Name of the HorizontalPodAutoscaler for this deployment/statefulset.
	// This will take precedence over the "Replicas" field of the parent Astarte component.
	// +kubebuilder:validation:Optional
	Horizontal string `json:"horizontal,omitempty"`
	// TODO: Vertical string `json:"vertical,omitempty"`
}

type AstartePersistentStorageSpec struct {
	// +kubebuilder:validation:Optional
	Size *resource.Quantity `json:"size"`
	// +kubebuilder:validation:Optional
	ClassName string `json:"className,omitempty"`
	// +kubebuilder:validation:Optional
	VolumeDefinition *v1.Volume `json:"volumeDefinition,omitempty"`
}

type AstarteAPISpec struct {
	// +kubebuilder:validation:Optional
	SSL  *bool  `json:"ssl,omitempty"`
	Host string `json:"host"`
}

type HostAndPort struct {
	// +kubebuilder:validation:MinLength=1
	Host string `json:"host"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port *int32 `json:"port"`
}

type LoginCredentialsSecret struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
	UsernameKey string `json:"usernameKey"`
	// +kubebuilder:validation:MinLength=1
	PasswordKey string `json:"passwordKey"`
}

type ConnectionStringSecret struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
}

type GenericConnectionSpec struct {
	// +kubebuilder:validation:Optional
	SSLConfiguration GenericSSLConfigurationSpec `json:"sslConfiguration,omitempty"`
	// The secret containing Username and Password to login.
	// Either this field or `connectionStringSecret` must be set.
	// +kubebuilder:validation:Optional
	CredentialsSecret *LoginCredentialsSecret `json:"credentialsSecret,omitempty"`
	// The secret containing a connection string to the service.
	// Either this field or `credentialsSecret` must be set.
	// +kubebuilder:validation:Optional
	ConnectionStringSecret *ConnectionStringSecret `json:"connectionStringSecret,omitempty"`
}

type GenericSSLConfigurationSpec struct {
	// +kubebuilder:validation:Optional
	Enable bool `json:"enable,omitempty"`
	// +kubebuilder:validation:Optional
	CustomCASecret v1.LocalObjectReference `json:"customCASecret,omitempty"`
	// +kubebuilder:validation:Optional
	SNI *bool `json:"sni,omitempty"`
	// +kubebuilder:validation:Optional
	CustomSNI string `json:"customSNI,omitempty"`
}

type AstarteRabbitMQConnectionSpec struct {
	HostAndPort `json:",inline"`
	// +kubebuilder:validation:Optional
	GenericConnectionSpec `json:",inline"`
	// +kubebuilder:validation:Optional
	VirtualHost string `json:"virtualHost,omitempty"`
}

type AstarteRabbitMQSpec struct {
	// +kubebuilder:validation:Optional
	Connection *AstarteRabbitMQConnectionSpec `json:"connection,omitempty"`
	// Configures the data queues prefix on RabbitMQ. You should change this setting only
	// in custom RabbitMQ installations.
	// +kubebuilder:validation:Optional
	DataQueuesPrefix string `json:"dataQueuesPrefix,omitempty"`
	// Configures the events exchange name on RabbitMQ. You should change this setting only
	// in custom RabbitMQ installations.
	// +kubebuilder:validation:Optional
	EventsExchangeName string `json:"eventsExchangeName,omitempty"`
}

type AstarteCassandraConnectionSpec struct {
	GenericConnectionSpec `json:",inline"`
	Nodes                 []HostAndPort `json:"nodes,omitempty"`
	// +kubebuilder:validation:Optional
	PoolSize *int `json:"poolSize,omitempty"`
}

type AstarteCassandraSpec struct {
	// +kubebuilder:validation:Optional
	Connection *AstarteCassandraConnectionSpec `json:"connection,omitempty"`
	// +kubebuilder:validation:Optional
	AstarteSystemKeyspace AstarteSystemKeyspaceSpec `json:"astarteSystemKeyspace,omitempty"`
}

type AstarteVerneMQSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	Host                            string `json:"host"`
	// +kubebuilder:validation:Optional
	Port *int32 `json:"port,omitempty"`
	// +kubebuilder:validation:Optional
	CaSecret string `json:"caSecret,omitempty"`
	// +kubebuilder:validation:Optional
	Storage *AstartePersistentStorageSpec `json:"storage,omitempty"`
	// Controls the device heartbeat from the broker to Astarte. The heartbeat is sent periodically
	// to prevent Astarte from keeping up stale connections from Devices in case the broker misbehaves
	// and does not send disconnection events. You should usually not tweak this value. Moreover, keep
	// in mind that when a lot of devices are connected simultaneously, having a short heartbeat time
	// might cause performance issues. Defaults to an hour.
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

type AstarteDataUpdaterPlantSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
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

type AstarteAppengineAPISpec struct {
	AstarteGenericAPIComponentSpec `json:",inline"`
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

type AstarteDashboardConfigAuthSpec struct {
	Type string `json:"type"`
	// +kubebuilder:validation:Optional
	OAuthAPIURL string `json:"oauth_api_url,omitempty"`
}

type AstarteDashboardConfigSpec struct {
	// +kubebuilder:validation:Optional
	RealmManagementAPIURL string `json:"realmManagementApiUrl,omitempty"`
	// +kubebuilder:validation:Optional
	AppEngineAPIURL string `json:"appEngineApiUrl,omitempty"`
	// +kubebuilder:validation:Optional
	PairingAPIURL string `json:"pairingApiUrl,omitempty"`
	// +kubebuilder:validation:Optional
	FlowAPIURL string `json:"flowApiUrl,omitempty"`
	// +kubebuilder:validation:Optional
	DefaultRealm string `json:"defaultRealm,omitempty"`
	// +kubebuilder:validation:Optional
	DefaultAuth string `json:"defaultAuth,omitempty"`
	// +kubebuilder:validation:Optional
	Auth []AstarteDashboardConfigAuthSpec `json:"auth,omitempty"`
}

type AstarteDashboardSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// +kubebuilder:validation:Optional
	AstarteDashboardConfigSpec `json:",inline"`
}

type AstarteGenericAPIComponentSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// +kubebuilder:validation:Optional
	DisableAuthentication *bool `json:"disableAuthentication,omitempty"`
}

type AstarteComponentsSpec struct {
	// Compute Resources for this Component.
	// +kubebuilder:validation:Optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
	// +kubebuilder:validation:Optional
	Flow AstarteGenericAPIComponentSpec `json:"flow,omitempty"`
	// +kubebuilder:validation:Optional
	Housekeeping AstarteGenericAPIComponentSpec `json:"housekeeping,omitempty"`
	// +kubebuilder:validation:Optional
	RealmManagement AstarteGenericAPIComponentSpec `json:"realmManagement,omitempty"`
	// +kubebuilder:validation:Optional
	Pairing AstarteGenericAPIComponentSpec `json:"pairing,omitempty"`
	// +kubebuilder:validation:Optional
	DataUpdaterPlant AstarteDataUpdaterPlantSpec `json:"dataUpdaterPlant,omitempty"`
	// +kubebuilder:validation:Optional
	AppengineAPI AstarteAppengineAPISpec `json:"appengineApi,omitempty"`
	// +kubebuilder:validation:Optional
	TriggerEngine AstarteTriggerEngineSpec `json:"triggerEngine,omitempty"`
	// +kubebuilder:validation:Optional
	Dashboard AstarteDashboardSpec `json:"dashboard,omitempty"`
}

type AstarteCFSSLDBConfigSpec struct {
	Driver     string `json:"driver,omitempty"`
	DataSource string `json:"dataSource,omitempty"`
}

type AstarteCFSSLCSRRootCAKeySpec struct {
	Algo string `json:"algo"`
	Size int    `json:"size"`
}

type AstarteCFSSLCSRRootCANamesSpec struct {
	C  string `json:"C"`
	L  string `json:"L"`
	O  string `json:"O"`
	OU string `json:"OU"`
	ST string `json:"ST"`
}

type AstarteCFSSLCSRRootCASpec struct {
	CN     string                           `json:"CN"`
	Key    *AstarteCFSSLCSRRootCAKeySpec    `json:"key"`
	Names  []AstarteCFSSLCSRRootCANamesSpec `json:"names"`
	Expiry string                           `json:"expiry"`
}

type AstarteCFSSLCARootConfigSigningCAConstraintSpec struct {
	MaxPathLen     int  `json:"max_path_len"`
	IsCA           bool `json:"is_ca"`
	MaxPathLenZero bool `json:"max_path_len_zero"`
}

type AstarteCFSSLCARootConfigSigningDefaultSpec struct {
	Usages       []string                                         `json:"usages"`
	Expiry       string                                           `json:"expiry"`
	CAConstraint *AstarteCFSSLCARootConfigSigningCAConstraintSpec `json:"ca_constraint"`
}

type AstarteCFSSLCARootConfigSpec struct {
	SigningDefault *AstarteCFSSLCARootConfigSigningDefaultSpec `json:"signingDefault"`
}

type AstarteCFSSLSpec struct {
	// +kubebuilder:validation:Optional
	Deploy *bool `json:"deploy,omitempty"`
	// +kubebuilder:validation:Optional
	URL string `json:"url,omitempty"`
	// +kubebuilder:validation:Optional
	CaExpiry string `json:"caExpiry,omitempty"`
	// +kubebuilder:validation:Optional
	CASecret v1.LocalObjectReference `json:"caSecret,omitempty"`
	// +kubebuilder:validation:Optional
	CertificateExpiry string `json:"certificateExpiry,omitempty"`
	// +kubebuilder:validation:Optional
	DBConfig *AstarteCFSSLDBConfigSpec `json:"dbConfig,omitempty"`
	// Compute Resources for this Component.
	// +kubebuilder:validation:Optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
	// +kubebuilder:validation:Optional
	Version string `json:"version,omitempty"`
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`
	// +kubebuilder:validation:Optional
	Storage *AstartePersistentStorageSpec `json:"storage,omitempty"`
	// +kubebuilder:validation:Optional
	CSRRootCa *AstarteCFSSLCSRRootCASpec `json:"csrRootCa,omitempty"`
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

// astarteSystemKeyspace configures the main system keyspace for Astarte. As of now, these settings
// have effect only upon cluster initialization, and will be ignored otherwise.
type AstarteSystemKeyspaceSpec struct {
	// The Replication Factor for the keyspace. Currently,
	// using NetworkTopologyStrategy is not supported.
	// +kubebuilder:validation:Optional
	ReplicationFactor int `json:"replicationFactor,omitempty"`
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

func (a *AstartePodPrioritiesSpec) IsEnabled() bool {
	return a != nil && a.Enable
}

// AstarteFeatures enables/disables selectively a set of global features in Astarte
type AstarteFeatures struct {
	// +kubebuilder:validation:Optional
	RealmDeletion bool `json:"realmDeletion,omitempty"`
	// +kubebuilder:validation:Optional
	Autoscaling bool `json:"autoscaling,omitempty"`
	// +kubebuilder:validation:Optional
	AstartePodPriorities *AstartePodPrioritiesSpec `json:"astartePodPriorities,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Astarte{}, &AstarteList{})
}
