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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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
	// HousekeepingAPI represents Astarte Housekeeping API
	HousekeepingAPI AstarteComponent = "housekeeping_api"
	// Pairing represents Astarte Pairing
	Pairing AstarteComponent = "pairing"
	// PairingAPI represents Astarte Pairing API
	PairingAPI AstarteComponent = "pairing_api"
	// RealmManagement represents Astarte Realm Management
	RealmManagement AstarteComponent = "realm_management"
	// RealmManagementAPI represents Astarte Realm Management API
	RealmManagementAPI AstarteComponent = "realm_management_api"
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
	return strings.Replace(a.String(), "_", "-", -1)
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
	ret := strings.Replace(a.DashedString(), "-", "", -1)
	return strings.Replace(ret, "api", "", -1)
}

type AstarteGenericClusteredResource struct {
	// +optional
	/// +kubebuilder:default=true
	Deploy *bool `json:"deploy,omitempty"`
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// +optional
	/// +kubebuilder:default=true
	AntiAffinity *bool `json:"antiAffinity,omitempty"`
	// +optional
	CustomAffinity *v1.Affinity `json:"customAffinity,omitempty"`
	// +optional
	DeploymentStrategy *appsv1.DeploymentStrategy `json:"deploymentStrategy,omitempty"`
	// +optional
	Version string `json:"version,omitempty"`
	// +optional
	Image string `json:"image,omitempty"`
	// Compute Resources for this Component.
	// +optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
}

// AstarteGenericAPISpec represents a generic Astarte API Component in the Deployment spec
type AstarteGenericAPISpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// +optional
	DisableAuthentication *bool `json:"disableAuthentication,omitempty"`
}

type AstartePersistentStorageSpec struct {
	// +optional
	Size *resource.Quantity `json:"size"`
	// +optional
	ClassName string `json:"className,omitempty"`
	// +optional
	VolumeDefinition *v1.Volume `json:"volumeDefinition,omitempty"`
}

type AstarteAPISpec struct {
	// +optional
	SSL  *bool  `json:"ssl,omitempty"`
	Host string `json:"host"`
}

type AstarteRabbitMQSSLConfigurationSpec struct {
	Enabled bool `json:"enabled"`
	// +optional
	CustomCASecret v1.LocalObjectReference `json:"customCASecret,omitempty"`
	// +optional
	SNI *bool `json:"sni,omitempty"`
	// +optional
	CustomSNI string `json:"customSNI,omitempty"`
}

type AstarteRabbitMQConnectionSpec struct {
	Host string `json:"host"`
	Port *int16 `json:"port"`
	// +optional
	Username string `json:"username,omitempty"`
	// +optional
	Password string `json:"password,omitempty"`
	// +optional
	VirtualHost string `json:"virtualHost,omitempty"`
	// +optional
	SSLConfiguration AstarteRabbitMQSSLConfigurationSpec `json:"sslConfiguration,omitempty"`
	// +optional
	Secret *LoginCredentialsSecret `json:"secret,omitempty"`
}

type AstarteRabbitMQSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// +optional
	Connection *AstarteRabbitMQConnectionSpec `json:"connection,omitempty"`
	// +optional
	/// +kubebuilder:default={"size": "4Gi"}
	Storage *AstartePersistentStorageSpec `json:"storage,omitempty"`
	// +optional
	AdditionalPlugins []string `json:"additionalPlugins,omitempty"`
	// Configures the data queues prefix on RabbitMQ. You should change this setting only
	// in custom RabbitMQ installations.
	// +optional
	DataQueuesPrefix string `json:"dataQueuesPrefix,omitempty"`
	// Configures the events exchange name on RabbitMQ. You should change this setting only
	// in custom RabbitMQ installations.
	// +optional
	EventsExchangeName string `json:"eventsExchangeName,omitempty"`
}

type AstarteCassandraSSLConfigurationSpec struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// +optional
	CustomCASecret v1.LocalObjectReference `json:"customCASecret,omitempty"`
	// +optional
	SNI *bool `json:"sni,omitempty"`
	// +optional
	CustomSNI string `json:"customSNI,omitempty"`
}

type LoginCredentialsSecret struct {
	Name        string `json:"name"`
	UsernameKey string `json:"usernameKey"`
	PasswordKey string `json:"passwordKey"`
}

type AstarteCassandraConnectionSpec struct {
	// +optional
	PoolSize *int `json:"poolSize,omitempty"`
	// +optional
	Autodiscovery *bool `json:"autodiscovery,omitempty"`
	// +optional
	SSLConfiguration AstarteCassandraSSLConfigurationSpec `json:"sslConfiguration,omitempty"`
	// +optional
	Secret *LoginCredentialsSecret `json:"secret,omitempty"`
	// +optional
	Username string `json:"username,omitempty"`
	// +optional
	Password string `json:"password,omitempty"`
}

type AstarteCassandraSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// +optional
	Nodes string `json:"nodes,omitempty"`
	// +optional
	MaxHeapSize string `json:"maxHeapSize,omitempty"`
	// +optional
	HeapNewSize string `json:"heapNewSize,omitempty"`
	// +optional
	Storage *AstartePersistentStorageSpec `json:"storage,omitempty"`
	// +optional
	Connection *AstarteCassandraConnectionSpec `json:"connection,omitempty"`
}

type AstarteVerneMQSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	Host                            string `json:"host"`
	// +optional
	Port *int16 `json:"port,omitempty"`
	// +optional
	// +optional
	CaSecret string `json:"caSecret,omitempty"`
	// +optional
	Storage *AstartePersistentStorageSpec `json:"storage,omitempty"`
	// Controls the device heartbeat from the broker to Astarte. The heartbeat is sent periodically
	// to prevent Astarte from keeping up stale connections from Devices in case the broker misbehaves
	// and does not send disconnection events. You should usually not tweak this value. Moreover, keep
	// in mind that when a lot of devices are connected simultaneously, having a short heartbeat time
	// might cause performance issues. Defaults to an hour.
	// +optional
	DeviceHeartbeatSeconds int `json:"deviceHeartbeatSeconds,omitempty"`
	// The maximum number of QoS 1 or 2 messages to hold in the offline queue.
	// Defaults to 1000000. Set to -1 for no maximum (not recommended). Set to 0
	// if no messages should be stored offline.
	// +optional
	MaxOfflineMessages *int `json:"maxOfflineMessages,omitempty"`
	// This option allows persistent clients ( = clean session set to
	// false) to be removed if they do not reconnect within 'persistent_client_expiration'.
	// This is a non-standard option. As far as the MQTT specification is concerned,
	// persistent clients persist forever.
	// The expiration period should be an integer followed by one of 'd', 'w', 'm', 'y' for
	// day, week, month, and year.
	// Default: 1 year
	// +optional
	PersistentClientExpiration string `json:"persistentClientExpiration,omitempty"`
}

type AstarteGenericComponentSpec struct {
	// +optional
	API AstarteGenericAPISpec `json:"api,omitempty"`
	// +optional
	Backend AstarteGenericClusteredResource `json:"backend,omitempty"`
}

type AstarteDataUpdaterPlantSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// +optional
	DataQueueCount *int `json:"dataQueueCount,omitempty"`
	// Controls the prefetch count for Data Updater Plant. When fine-tuning Astarte, this parameter
	// can make a difference for what concerns Data Updater Plant ingestion performance. However,
	// it can also degrade performance significantly and/or increase risk of data loss when misconfigured.
	// Configure this value only if you know what you're doing and you have experience with RabbitMQ.
	// Defaults to 300
	// +optional
	PrefetchCount *int `json:"prefetchCount,omitempty"`
}

type AstarteTriggerEngineSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// Configures the name of the Events queue. Should be configured only in installations with a highly
	// customized RabbitMQ. It is advised to leave empty unless you know exactly what you're doing.
	// +optional
	EventsQueueName string `json:"eventsQueueName,omitempty"`
	// Configures the routing key for Trigger Events. Should be configured only in installations
	// with a highly customized RabbitMQ and a custom Trigger Engine setup. It is advised to leave
	// empty unless you know exactly what you're doing, misconfiguring this value can cause heavy
	// breakage within Trigger Engine.
	// +optional
	EventsRoutingKey string `json:"eventsRoutingKey,omitempty"`
}

type AstarteAppengineAPISpec struct {
	AstarteGenericAPISpec `json:",inline"`
	// +kubebuilder:validation:Minimum=100
	// +optional
	MaxResultsLimit *int `json:"maxResultsLimit,omitempty"`
	// Configures the name of the Room Events queue. Should be configured only in installations with a highly
	// customized RabbitMQ. It is advised to leave empty unless you know exactly what you're doing.
	// +optional
	RoomEventsQueueName string `json:"roomEventsQueueName,omitempty"`
	// Configures the name of the Room Events exchange. Should be configured only in installations with a highly
	// customized RabbitMQ. It is advised to leave empty unless you know exactly what you're doing.
	// +optional
	RoomEventsExchangeName string `json:"roomEventsExchangeName,omitempty"`
}

type AstarteDashboardConfigAuthSpec struct {
	Type string `json:"type"`
	// +optional
	OAuthAPIURL string `json:"oauth_api_url,omitempty"`
}

type AstarteDashboardConfigSpec struct {
	// +optional
	RealmManagementAPIURL string `json:"realmManagementApiUrl,omitempty"`
	// +optional
	AppEngineAPIURL string `json:"appEngineApiUrl,omitempty"`
	// +optional
	PairingAPIURL string `json:"pairingApiUrl,omitempty"`
	// +optional
	FlowAPIURL string `json:"flowApiUrl,omitempty"`
	// +optional
	DefaultRealm string `json:"defaultRealm,omitempty"`
	// +optional
	DefaultAuth string `json:"defaultAuth,omitempty"`
	// +optional
	Auth []AstarteDashboardConfigAuthSpec `json:"auth,omitempty"`
}

type AstarteDashboardSpec struct {
	AstarteGenericClusteredResource `json:",inline"`
	// +optional
	Config AstarteDashboardConfigSpec `json:",inline"`
}

type AstarteComponentsSpec struct {
	// Compute Resources for this Component.
	// +optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
	// +optional
	Flow AstarteGenericAPISpec `json:"flow,omitempty"`
	// +optional
	Housekeeping AstarteGenericComponentSpec `json:"housekeeping,omitempty"`
	// +optional
	RealmManagement AstarteGenericComponentSpec `json:"realmManagement,omitempty"`
	// +optional
	Pairing AstarteGenericComponentSpec `json:"pairing,omitempty"`
	// +optional
	DataUpdaterPlant AstarteDataUpdaterPlantSpec `json:"dataUpdaterPlant,omitempty"`
	// +optional
	AppengineAPI AstarteAppengineAPISpec `json:"appengineApi,omitempty"`
	// +optional
	TriggerEngine AstarteTriggerEngineSpec `json:"triggerEngine,omitempty"`
	// +optional
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

type AstarteCFSSLCSRRootCACASpec struct {
	Expiry string `json:"expiry"`
}

type AstarteCFSSLCSRRootCASpec struct {
	CN    string                           `json:"CN"`
	Key   *AstarteCFSSLCSRRootCAKeySpec    `json:"key"`
	Names []AstarteCFSSLCSRRootCANamesSpec `json:"names"`
	CA    *AstarteCFSSLCSRRootCACASpec     `json:"ca"`
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

type AstarteCFSSLCARootConfigSigningSpec struct {
	Default *AstarteCFSSLCARootConfigSigningDefaultSpec `json:"default"`
}

type AstarteCFSSLCARootConfigSpec struct {
	Signing *AstarteCFSSLCARootConfigSigningSpec `json:"signing"`
}

type AstarteCFSSLSpec struct {
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
	// +optional
	URL string `json:"url,omitempty"`
	// +optional
	CaExpiry string `json:"caExpiry,omitempty"`
	// +optional
	CASecret v1.LocalObjectReference `json:"caSecret,omitempty"`
	// +optional
	CertificateExpiry string `json:"certificateExpiry,omitempty"`
	// +optional
	DBConfig *AstarteCFSSLDBConfigSpec `json:"dbConfig,omitempty"`
	// Compute Resources for this Component.
	// +optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
	// +optional
	Version string `json:"version,omitempty"`
	// +optional
	Image string `json:"image,omitempty"`
	// +optional
	Storage *AstartePersistentStorageSpec `json:"storage,omitempty"`
	// +optional
	CSRRootCa *AstarteCFSSLCSRRootCASpec `json:"csrRootCa,omitempty"`
	// +optional
	CARootConfig *AstarteCFSSLCARootConfigSpec `json:"caRootConfig,omitempty"`
}

// astarteSystemKeyspace configures the main system keyspace for Astarte. As of now, these settings
// have effect only upon cluster initialization, and will be ignored otherwise.
type AstarteSystemKeyspaceSpec struct {
	// The Replication Factor for the keyspace
	// +optional
	ReplicationFactor int `json:"replicationFactor,omitempty"`
}

// AstarteFeatures enables/disables selectively a set of global features in Astarte
type AstarteFeatures struct {
	// +optional
	RealmDeletion bool `json:"realmDeletion,omitempty"`
}

// AstarteSpec defines the desired state of Astarte
type AstarteSpec struct {
	// The Astarte Version for this Resource
	Version string `json:"version"`
	// +optional
	ImagePullPolicy *v1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// +optional
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// +optional
	DistributionChannel string `json:"distributionChannel,omitempty"`
	// +optional
	DeploymentStrategy *appsv1.DeploymentStrategy `json:"deploymentStrategy,omitempty"`
	// +optional
	Features AstarteFeatures `json:"features,omitempty"`
	// +optional
	/// +kubebuilder:default=true
	RBAC *bool `json:"rbac,omitempty"`
	// +optional
	StorageClassName string         `json:"storageClassName,omitempty"`
	API              AstarteAPISpec `json:"api"`
	// +optional
	/// +kubebuilder:default={"deploy":true}
	RabbitMQ AstarteRabbitMQSpec `json:"rabbitmq"`
	// +optional
	Cassandra AstarteCassandraSpec `json:"cassandra"`
	VerneMQ   AstarteVerneMQSpec   `json:"vernemq"`
	// +optional
	CFSSL AstarteCFSSLSpec `json:"cfssl"`
	// +optional
	Components AstarteComponentsSpec `json:"components"`
	// +optional
	AstarteSystemKeyspace AstarteSystemKeyspaceSpec `json:"astarteSystemKeyspace,omitempty"`
}

// TODO: Remove all omitempty from AstarteStatus in v1beta1

// AstarteStatus defines the observed state of Astarte
type AstarteStatus struct {
	ReconciliationPhase ReconciliationPhase  `json:"phase,omitempty"`
	AstarteVersion      string               `json:"astarteVersion,omitempty"`
	OperatorVersion     string               `json:"operatorVersion,omitempty"`
	Health              AstarteClusterHealth `json:"health,omitempty"`
	BaseAPIURL          string               `json:"baseAPIURL,omitempty"`
	BrokerURL           string               `json:"brokerURL,omitempty"`
}

// +kubebuilder:object:root=true

// Astarte is the Schema for the astartes API
// +kubebuilder:subresource:status
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

func init() {
	SchemeBuilder.Register(&Astarte{}, &AstarteList{})
}
