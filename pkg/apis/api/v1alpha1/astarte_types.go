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
	switch *a {
	case Dashboard:
		return "astarte-dashboard"
	}
	return "astarte_" + a.String()
}

// ServiceName returns the Kubernetes Service Name associated to this Astarte component,
// if any, otherwise returns an empty string.
// This will return a meaningful value only for API components or the Dashboard.
func (a *AstarteComponent) ServiceName() string {
	if !strings.Contains(a.String(), "api") && a.String() != "dashboard" {
		return ""
	}
	return strings.Replace(a.DashedString(), "-api", "", -1)
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
	Version string `json:"version,omitempty"`
	// +optional
	Image string `json:"image,omitempty"`
	// Compute Resources for this Component.
	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
}

// AstarteGenericAPISpec represents a generic Astarte API Component in the Deployment spec
type AstarteGenericAPISpec struct {
	GenericClusteredResource AstarteGenericClusteredResource `json:",inline"`
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

type AstarteRabbitMQConnectionSecretSpec struct {
	Name        string `json:"name"`
	UsernameKey string `json:"usernameKey"`
	PasswordKey string `json:"passwordKey"`
}

type AstarteRabbitMQConnectionSpec struct {
	Host string `json:"host"`
	Port *int16 `json:"port"`
	// +optional
	Username string `json:"username"`
	// +optional
	Password string `json:"password"`
	// +optional
	Secret *AstarteRabbitMQConnectionSecretSpec `json:"secret"`
}

type AstarteRabbitMQSpec struct {
	GenericClusteredResource AstarteGenericClusteredResource `json:",inline"`
	// +optional
	Connection *AstarteRabbitMQConnectionSpec `json:"connection,omitempty"`
	// +optional
	/// +kubebuilder:default={"size": "4Gi"}
	Storage *AstartePersistentStorageSpec `json:"storage,omitempty"`
	// +optional
	AdditionalPlugins []string `json:"additionalPlugins,omitempty"`
}

type AstarteCassandraSpec struct {
	GenericClusteredResource AstarteGenericClusteredResource `json:",inline"`
	// +optional
	Nodes string `json:"nodes,omitempty"`
	// +optional
	MaxHeapSize string `json:"maxHeapSize,omitempty"`
	// +optional
	HeapNewSize string `json:"heapNewSize,omitempty"`
	// +optional
	Storage *AstartePersistentStorageSpec `json:"storage,omitempty"`
}

type AstarteVerneMQSpec struct {
	GenericClusteredResource AstarteGenericClusteredResource `json:",inline"`
	Host                     string                          `json:"host"`
	// +optional
	Port *int16 `json:"port,omitempty"`
	// +optional
	// +optional
	CaSecret string `json:"caSecret,omitempty"`
	// +optional
	Storage *AstartePersistentStorageSpec `json:"storage,omitempty"`
}

type AstarteGenericComponentSpec struct {
	// +optional
	API AstarteGenericAPISpec `json:"api,omitempty"`
	// +optional
	Backend AstarteGenericClusteredResource `json:"backend,omitempty"`
}

type AstarteDataUpdaterPlantSpec struct {
	GenericClusteredResource AstarteGenericClusteredResource `json:",inline"`
	// +optional
	DataQueueCount *int `json:"dataQueueCount,omitempty"`
}

type AstarteAppengineAPISpec struct {
	GenericAPISpec AstarteGenericAPISpec `json:",inline"`
	// +kubebuilder:validation:Minimum=100
	// +optional
	MaxResultsLimit *int `json:"maxResultsLimit,omitempty"`
}

type AstarteDashboardConfigAuthSpec struct {
	Type string `json:"type"`
}

type AstarteDashboardConfigSpec struct {
	// +optional
	RealmManagementAPIURL string `json:"realmManagementApiUrl,omitempty"`
	// +optional
	DefaultRealm string `json:"defaultRealm,omitempty"`
	// +optional
	DefaultAuth string `json:"defaultAuth,omitempty"`
	// +optional
	Auth []AstarteDashboardConfigAuthSpec `json:"auth,omitempty"`
}

type AstarteDashboardSpec struct {
	GenericClusteredResource AstarteGenericClusteredResource `json:",inline"`
	// +optional
	SSL *bool `json:"ssl,omitempty"`
	// +optional
	Host string `json:"host,omitempty"`
	// +optional
	Config AstarteDashboardConfigSpec `json:",inline"`
}

type AstarteComponentsSpec struct {
	// Compute Resources for this Component.
	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
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
	TriggerEngine AstarteGenericClusteredResource `json:"triggerEngine,omitempty"`
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
	IsCA           bool `json:"is_ca"`
	MaxPathLen     int  `json:"max_path_len"`
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
	CertificateExpiry string `json:"certificateExpiry,omitempty"`
	// +optional
	DBConfig *AstarteCFSSLDBConfigSpec `json:"dbConfig,omitempty"`
	// Compute Resources for this Component.
	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
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
	DeploymentStrategy appsv1.DeploymentStrategy `json:"deploymentStrategy,omitempty"`
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
}

// AstarteStatus defines the observed state of Astarte
type AstarteStatus struct {
	ReconciliationPhase ReconciliationPhase `json:"phase"`
	AstarteVersion      string              `json:"astarteVersion"`
	OperatorVersion     string              `json:"operatorVersion"`
	Health              string              `json:"health"`
	BaseAPIURL          string              `json:"baseAPIURL"`
	BrokerURL           string              `json:"brokerURL"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Astarte is the Schema for the astartes API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=astartes,scope=Namespaced
type Astarte struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AstarteSpec   `json:"spec,omitempty"`
	Status AstarteStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AstarteList contains a list of Astarte
type AstarteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Astarte `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Astarte{}, &AstarteList{})
}
