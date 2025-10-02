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

package misc

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"go.openly.dev/pointy"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
)

const (
	// RabbitMQDefaultUserCredentialsUsernameKey is the default Username key for RabbitMQ Secret
	RabbitMQDefaultUserCredentialsUsernameKey = "admin-username"
	// RabbitMQDefaultUserCredentialsPasswordKey is the default Password key for RabbitMQ Secret
	RabbitMQDefaultUserCredentialsPasswordKey = "admin-password"
	// CassandraDefaultUserCredentialsUsernameKey is the default Username key for Cassandra Secret
	CassandraDefaultUserCredentialsUsernameKey = "username"
	// CassandraDefaultUserCredentialsPasswordKey is the default Password key for Cassandra Secret
	CassandraDefaultUserCredentialsPasswordKey = "password"
)

type allocationCoefficients struct {
	CPUCoefficient    float64
	MemoryCoefficient float64
}

var defaultComponentAllocations = map[apiv2alpha1.AstarteComponent]allocationCoefficients{
	apiv2alpha1.AppEngineAPI:     {CPUCoefficient: 0.18, MemoryCoefficient: 0.18},
	apiv2alpha1.DataUpdaterPlant: {CPUCoefficient: 0.21, MemoryCoefficient: 0.21},
	apiv2alpha1.FlowComponent:    {CPUCoefficient: 0.10, MemoryCoefficient: 0.10},
	apiv2alpha1.Housekeeping:     {CPUCoefficient: 0.08, MemoryCoefficient: 0.08},
	apiv2alpha1.Pairing:          {CPUCoefficient: 0.19, MemoryCoefficient: 0.19},
	apiv2alpha1.RealmManagement:  {CPUCoefficient: 0.12, MemoryCoefficient: 0.12},
	apiv2alpha1.TriggerEngine:    {CPUCoefficient: 0.07, MemoryCoefficient: 0.07},
	apiv2alpha1.Dashboard:        {CPUCoefficient: 0.05, MemoryCoefficient: 0.05},
}

// ReconcileConfigMap creates or updates a ConfigMap through controllerutil through its data map
func ReconcileConfigMap(objName string, data map[string]string, cr metav1.Object, c client.Client, scheme *runtime.Scheme, log logr.Logger) (controllerutil.OperationResult, error) {
	configMap := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: objName, Namespace: cr.GetNamespace()}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, configMap, func() error {
		if err := controllerutil.SetControllerReference(cr, configMap, scheme); err != nil {
			return err
		}
		// Set the ConfigMap data to the requested map
		configMap.Data = data
		return nil
	})

	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	LogCreateOrUpdateOperationResult(log, result, cr, configMap)
	return result, err
}

// ReconcileTLSSecret creates or updates a TLS Secret through controllerutil through its data
func ReconcileTLSSecret(objName string, cert, key string, cr metav1.Object, c client.Client, scheme *runtime.Scheme, log logr.Logger) (controllerutil.OperationResult, error) {
	secret := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: objName, Namespace: cr.GetNamespace()}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, secret, func() error {
		if err := controllerutil.SetControllerReference(cr, secret, scheme); err != nil {
			return err
		}

		secret.Type = v1.SecretTypeTLS
		secret.StringData = map[string]string{
			v1.TLSCertKey:       cert,
			v1.TLSPrivateKeyKey: key,
		}
		return nil
	})

	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	LogCreateOrUpdateOperationResult(log, result, cr, secret)
	return result, err
}

// ReconcileSecret creates or updates a Secret through controllerutil through its data
func ReconcileSecret(objName string, data map[string][]byte, cr metav1.Object, c client.Client, scheme *runtime.Scheme, log logr.Logger) (controllerutil.OperationResult, error) {
	secret := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: objName, Namespace: cr.GetNamespace()}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, secret, func() error {
		if err := controllerutil.SetControllerReference(cr, secret, scheme); err != nil {
			return err
		}

		secret.Type = v1.SecretTypeOpaque
		secret.Data = data
		return nil
	})

	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	LogCreateOrUpdateOperationResult(log, result, cr, secret)
	return result, err
}

// ReconcileSecretString creates or updates a Secret through controllerutil by using StringData
func ReconcileSecretString(objName string, data map[string]string, cr metav1.Object, c client.Client, scheme *runtime.Scheme, log logr.Logger) (controllerutil.OperationResult, error) {
	return ReconcileSecretStringWithLabels(objName, data, map[string]string{}, cr, c, scheme, log)
}

// ReconcileSecretStringWithLabels creates or updates a Secret through controllerutil by using StringData, and adding a set of Labels
func ReconcileSecretStringWithLabels(objName string, data, labels map[string]string, cr metav1.Object, c client.Client, scheme *runtime.Scheme, log logr.Logger) (controllerutil.OperationResult, error) {
	secret := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: objName, Namespace: cr.GetNamespace(), Labels: labels}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, secret, func() error {
		if err := controllerutil.SetControllerReference(cr, secret, scheme); err != nil {
			return err
		}

		secret.Type = v1.SecretTypeOpaque
		secret.StringData = data
		return nil
	})

	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	LogCreateOrUpdateOperationResult(log, result, cr, secret)
	return result, err
}

// LogCreateOrUpdateOperationResult logs conveniently a controllerutil Operation
func LogCreateOrUpdateOperationResult(log logr.Logger, result controllerutil.OperationResult, cr metav1.Object, obj metav1.Object) {
	reqLogger := log.WithValues("Request.Namespace", cr.GetNamespace(), "Request.Name", cr.GetName())
	switch result {
	case controllerutil.OperationResultCreated:
		reqLogger.Info("Resource created", "Resource", obj.GetName())
	case controllerutil.OperationResultUpdated:
		reqLogger.Info("Resource updated", "Resource", obj.GetName())
	case controllerutil.OperationResultNone:
		// Debug level logging, we don't want to clutter
		reqLogger.V(1).Info("Resource unchanged", "Resource", obj.GetName())
	}
}

// GetVerneMQBrokerURL returns the complete URL for VerneMQ (MQTT) for an Astarte resource
func GetVerneMQBrokerURL(cr *apiv2alpha1.Astarte) string {
	return fmt.Sprintf("mqtts://%s:%d", cr.Spec.VerneMQ.Host, pointy.Int32Value(cr.Spec.VerneMQ.Port, 8883))
}

// GetResourcesForAstarteComponent returns the allocated resources for a given Astarte component, taking into account both the
// directive from Components, and the directive from the individual component (if any).
// It will compute a ResourceRequirements for the component based on said values and internal logic.
func GetResourcesForAstarteComponent(cr *apiv2alpha1.Astarte, requestedResources *v1.ResourceRequirements, component apiv2alpha1.AstarteComponent) v1.ResourceRequirements {
	if requestedResources != nil {
		// There has been an explicit allocation, so return that
		return *requestedResources
	}

	// Do we have any resources set?
	if cr.Spec.Components.Resources == nil {
		// All burst. If you say so...
		return v1.ResourceRequirements{}
	}

	// Ok, let's do the distribution dance.
	a := getWeightedDefaultAllocationFor(cr, component)

	if a.MemoryCoefficient == 0 || a.CPUCoefficient == 0 {
		return v1.ResourceRequirements{}
	}

	cpuLimits := getCpuScaledQuantity(cr.Spec.Components.Resources.Limits.Cpu(), a.CPUCoefficient)
	cpuRequests := getCpuScaledQuantity(cr.Spec.Components.Resources.Requests.Cpu(), a.CPUCoefficient)
	memoryLimits := getMemoryScaledQuantity(cr.Spec.Components.Resources.Limits.Memory(), a.MemoryCoefficient)
	memoryRequests := getMemoryScaledQuantity(cr.Spec.Components.Resources.Requests.Memory(), a.MemoryCoefficient)

	realRequests := v1.ResourceList{}

	// Adjust the requests to ensure we don't starve our services.
	// If the CPU is <150m, we're better off bursting the component. Otherwise, add it to requests.
	if cpuRequests.MilliValue() < 150 {
		cpuRequests = resource.NewScaledQuantity(0, resource.Milli)
	}
	realRequests[v1.ResourceCPU] = *cpuRequests
	// If the RAM is less than 128M, increase it even though we're getting out of boundaries.
	// We can't really afford to trigger the OOM killer in the cluster, better making some components unschedulable.
	if memoryRequests.ScaledValue(resource.Mega) < 128 {
		memoryRequests = resource.NewScaledQuantity(128, resource.Mega)
	}
	realRequests[v1.ResourceMemory] = *memoryRequests

	// Ensure limits aren't out of boundaries if we changed the requests
	if cpuLimits.Cmp(*cpuRequests) < 0 {
		cpuLimits = cpuRequests
	}
	if memoryLimits.Cmp(*memoryRequests) < 0 {
		memoryLimits = memoryRequests
	}

	// Same goes for limits (for CPU). Instead, though, set a higher value. That would be 300m.
	// For memory, we have to trust the user here.
	if cpuLimits.MilliValue() < 300 {
		cpuLimits = resource.NewScaledQuantity(300, resource.Milli)
	}

	return v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceCPU:    cpuLimits.DeepCopy(),
			v1.ResourceMemory: memoryLimits.DeepCopy(),
		},
		Requests: realRequests,
	}
}

// getCpuScaledQuantity scales a resource quantity by a coefficient.
// It works directly with the canonical milli representation to avoid precision loss
// from unit conversions.
// The returned quantity will be in milli format
func getCpuScaledQuantity(qty *resource.Quantity, coefficient float64) *resource.Quantity {
	scaledValue := int64(float64(qty.MilliValue()) * coefficient)
	return resource.NewMilliQuantity(scaledValue, qty.Format)
}

// getMemoryScaledQuantity scales a resource quantity by a coefficient.
// It works directly with the canonical byte/bibyte representations to avoid precision loss
// from unit conversions.
// The returned quantity will be in bytes format
func getMemoryScaledQuantity(qty *resource.Quantity, coefficient float64) *resource.Quantity {
	scaledValue := int64(float64(qty.Value()) * coefficient)
	return resource.NewQuantity(scaledValue, qty.Format)
}

func getNumberOfDeployedAstarteComponentsAsFloat(cr *apiv2alpha1.Astarte) float64 {
	var deployedComponents int

	if pointy.BoolValue(cr.Spec.Components.AppengineAPI.Deploy, true) {
		deployedComponents++
	}
	if pointy.BoolValue(cr.Spec.Components.Dashboard.Deploy, true) {
		deployedComponents++
	}
	if pointy.BoolValue(cr.Spec.Components.DataUpdaterPlant.Deploy, true) {
		deployedComponents++
	}
	if pointy.BoolValue(cr.Spec.Components.Flow.Deploy, true) {
		deployedComponents++
	}
	if pointy.BoolValue(cr.Spec.Components.Housekeeping.Deploy, true) {
		deployedComponents++
	}
	if pointy.BoolValue(cr.Spec.Components.Pairing.Deploy, true) {
		deployedComponents++
	}
	if pointy.BoolValue(cr.Spec.Components.RealmManagement.Deploy, true) {
		deployedComponents++
	}
	if pointy.BoolValue(cr.Spec.Components.TriggerEngine.Deploy, true) {
		deployedComponents++
	}

	return float64(deployedComponents)
}

func getLeftoverCoefficients(cr *apiv2alpha1.Astarte) allocationCoefficients {
	aC := allocationCoefficients{}

	aC = checkComponentForLeftoverAllocations(cr.Spec.Components.AppengineAPI.AstarteGenericClusteredResource, apiv2alpha1.AppEngineAPI, aC)
	aC = checkComponentForLeftoverAllocations(cr.Spec.Components.Dashboard.AstarteGenericClusteredResource, apiv2alpha1.Dashboard, aC)
	aC = checkComponentForLeftoverAllocations(cr.Spec.Components.DataUpdaterPlant.AstarteGenericClusteredResource, apiv2alpha1.DataUpdaterPlant, aC)
	aC = checkComponentForLeftoverAllocations(cr.Spec.Components.Flow.AstarteGenericClusteredResource, apiv2alpha1.FlowComponent, aC)
	aC = checkComponentForLeftoverAllocations(cr.Spec.Components.Housekeeping.AstarteGenericClusteredResource, apiv2alpha1.Housekeeping, aC)
	aC = checkComponentForLeftoverAllocations(cr.Spec.Components.Pairing.AstarteGenericClusteredResource, apiv2alpha1.Pairing, aC)
	aC = checkComponentForLeftoverAllocations(cr.Spec.Components.RealmManagement.AstarteGenericClusteredResource, apiv2alpha1.RealmManagement, aC)
	aC = checkComponentForLeftoverAllocations(cr.Spec.Components.TriggerEngine.AstarteGenericClusteredResource, apiv2alpha1.TriggerEngine, aC)

	return aC
}

func checkComponentForLeftoverAllocations(clusteredResource apiv2alpha1.AstarteGenericClusteredResource,
	component apiv2alpha1.AstarteComponent, aC allocationCoefficients) allocationCoefficients {
	if !pointy.BoolValue(clusteredResource.Deploy, true) {
		aC.CPUCoefficient += defaultComponentAllocations[component].CPUCoefficient
		aC.MemoryCoefficient += defaultComponentAllocations[component].MemoryCoefficient
	}

	return aC
}

func getWeightedDefaultAllocationFor(cr *apiv2alpha1.Astarte, component apiv2alpha1.AstarteComponent) allocationCoefficients {
	// Add other percentages proportionally
	leftovers := getLeftoverCoefficients(cr)
	defaultAllocation := defaultComponentAllocations[component]

	if leftovers.CPUCoefficient > 0 {
		defaultAllocation.CPUCoefficient += (leftovers.CPUCoefficient / getNumberOfDeployedAstarteComponentsAsFloat(cr))
	}
	if leftovers.MemoryCoefficient > 0 {
		defaultAllocation.MemoryCoefficient += (leftovers.MemoryCoefficient / getNumberOfDeployedAstarteComponentsAsFloat(cr))
	}

	return defaultAllocation
}

// IsAstarteComponentDeployed returns whether an Astarte component is deployed by cr
func IsAstarteComponentDeployed(cr *apiv2alpha1.Astarte, component apiv2alpha1.AstarteComponent) bool {
	switch component {
	case apiv2alpha1.AppEngineAPI:
		return pointy.BoolValue(cr.Spec.Components.AppengineAPI.Deploy, true)
	case apiv2alpha1.Dashboard:
		return pointy.BoolValue(cr.Spec.Components.Dashboard.Deploy, true)
	case apiv2alpha1.DataUpdaterPlant:
		return pointy.BoolValue(cr.Spec.Components.DataUpdaterPlant.Deploy, true)
	case apiv2alpha1.FlowComponent:
		return pointy.BoolValue(cr.Spec.Components.Flow.Deploy, false)
	case apiv2alpha1.Housekeeping:
		return pointy.BoolValue(cr.Spec.Components.Housekeeping.Deploy, true)
	case apiv2alpha1.Pairing:
		return pointy.BoolValue(cr.Spec.Components.Pairing.Deploy, true)
	case apiv2alpha1.RealmManagement:
		return pointy.BoolValue(cr.Spec.Components.RealmManagement.Deploy, true)
	case apiv2alpha1.TriggerEngine:
		return pointy.BoolValue(cr.Spec.Components.TriggerEngine.Deploy, true)
	}

	// We should not have gotten here
	return false
}

// GetRabbitMQHostnameAndPort returns the Cluster-accessible Hostname and AMQP port for RabbitMQ
func GetRabbitMQHostnameAndPort(cr *apiv2alpha1.Astarte) (string, int32) {
	return cr.Spec.RabbitMQ.Connection.Host, pointy.Int32Value(cr.Spec.RabbitMQ.Connection.Port, 5672)
}

// GetRabbitMQUserCredentialsSecret gets the secret holding RabbitMQ credentials in the form <secret name>, <username key>, <password key>
func GetRabbitMQUserCredentialsSecret(cr *apiv2alpha1.Astarte) (string, string, string) {
	// TODO: allow `connectionStringSecret` to be used too

	if cr.Spec.RabbitMQ.Connection.CredentialsSecret != nil {
		return cr.Spec.RabbitMQ.Connection.CredentialsSecret.Name, cr.Spec.RabbitMQ.Connection.CredentialsSecret.UsernameKey, cr.Spec.RabbitMQ.Connection.CredentialsSecret.PasswordKey
	}
	return cr.Name + "-rabbitmq-user-credentials", RabbitMQDefaultUserCredentialsUsernameKey, RabbitMQDefaultUserCredentialsPasswordKey
}

// GetRabbitMQCredentialsFor returns the RabbitMQ host, username and password for a given CR. This information
// can be used for connecting to RabbitMQ from the Operator or an external agent, and it should not be used for
// any other purpose.
func GetRabbitMQCredentialsFor(cr *apiv2alpha1.Astarte, c client.Client) (string, int32, string, string, error) {
	host, port := GetRabbitMQHostnameAndPort(cr)
	secretName, usernameKey, passwordKey := GetRabbitMQUserCredentialsSecret(cr)

	// Fetch the Secret
	secret := &v1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: cr.Namespace}, secret); err != nil {
		return "", 0, "", "", err
	}

	return host, port, string(secret.Data[usernameKey]), string(secret.Data[passwordKey]), nil
}

// GetCassandraUserCredentialsSecret gets the secret holding Cassandra credentials in the form <secret name>, <username key>, <password key>
func GetCassandraUserCredentialsSecret(cr *apiv2alpha1.Astarte) (string, string, string) {
	// TODO: allow `connectionStringSecret` to be used too
	if cr.Spec.Cassandra.Connection.CredentialsSecret != nil {
		return cr.Spec.Cassandra.Connection.CredentialsSecret.Name, cr.Spec.Cassandra.Connection.CredentialsSecret.UsernameKey, cr.Spec.Cassandra.Connection.CredentialsSecret.PasswordKey
	}
	return cr.Name + "-cassandra-user-credentials", CassandraDefaultUserCredentialsUsernameKey, CassandraDefaultUserCredentialsPasswordKey
}
