/*
This file is part of Astarte.

Copyright 2024 SECO Mind Srl.

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

package reconcile

import (
	"context"

	v1 "k8s.io/api/core/v1"
	scheduling "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/misc"
)

const (
	AstarteHighPriorityName string = "astarte-high-priority-non-preemptive"
	AstarteMidPriorityName  string = "astarte-mid-priority-non-preemptive"
	AstarteLowPriorityName  string = "astarte-low-priority-non-preemptive"
)

var defaultComponentPriorityClass = map[apiv1alpha2.AstarteComponent]string{
	apiv1alpha2.AppEngineAPI:       AstarteMidPriorityName,
	apiv1alpha2.DataUpdaterPlant:   AstarteHighPriorityName,
	apiv1alpha2.FlowComponent:      AstarteMidPriorityName,
	apiv1alpha2.Housekeeping:       AstarteLowPriorityName,
	apiv1alpha2.HousekeepingAPI:    AstarteLowPriorityName,
	apiv1alpha2.Pairing:            AstarteMidPriorityName,
	apiv1alpha2.PairingAPI:         AstarteMidPriorityName,
	apiv1alpha2.RealmManagement:    AstarteLowPriorityName,
	apiv1alpha2.RealmManagementAPI: AstarteLowPriorityName,
	apiv1alpha2.TriggerEngine:      AstarteLowPriorityName,
	apiv1alpha2.Dashboard:          AstarteLowPriorityName,
}

func EnsureAstartePriorityClasses(instance *apiv1alpha2.Astarte, c client.Client, scheme *runtime.Scheme) error {
	// Shall we use priorityClasses?
	if instance.Spec.Features.AstartePodPriorities.IsEnabled() {

		// we don't want to preempt other pods
		preemptNever := v1.PreemptNever

		priorityClassHigh := &scheduling.PriorityClass{
			ObjectMeta: metav1.ObjectMeta{Name: AstarteHighPriorityName},
		}

		// Reconcile the high priorityClass
		if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, priorityClassHigh, func() error {
			priorityClassHigh.GlobalDefault = false
			priorityClassHigh.PreemptionPolicy = &preemptNever
			// default value makes sure this pointer is not nil
			priorityClassHigh.Value = int32(*instance.Spec.Features.AstartePodPriorities.AstarteHighPriority)
			priorityClassHigh.Description = "Astarte high-priority pods (e.g. RabbitMQ, VerneMQ, Astarte Data Updater Plant) should be in this priority class."
			return nil
		}); err != nil {
			return err
		} else {
			misc.LogCreateOrUpdateOperationResult(log, result, instance, priorityClassHigh)
		}

		// Mid priorityClass
		priorityClassMid := &scheduling.PriorityClass{
			ObjectMeta: metav1.ObjectMeta{Name: AstarteMidPriorityName},
		}

		// Reconcile the priorityClass
		if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, priorityClassMid, func() error {
			priorityClassMid.GlobalDefault = false
			priorityClassMid.PreemptionPolicy = &preemptNever
			// default value makes sure this pointer is not nil
			priorityClassMid.Value = int32(*instance.Spec.Features.AstartePodPriorities.AstarteMidPriority)
			priorityClassMid.Description = "Astarte mid-priority pods should be in this priority class."
			return nil
		}); err != nil {
			return err
		} else {
			misc.LogCreateOrUpdateOperationResult(log, result, instance, priorityClassMid)
		}

		// Low priorityClass
		priorityClassLow := &scheduling.PriorityClass{
			ObjectMeta: metav1.ObjectMeta{Name: AstarteLowPriorityName},
		}

		// Reconcile the priorityClass
		if result, err := controllerutil.CreateOrUpdate(context.TODO(), c, priorityClassLow, func() error {
			priorityClassLow.GlobalDefault = false
			priorityClassLow.PreemptionPolicy = &preemptNever
			// default value makes sure this pointer is not nil
			priorityClassLow.Value = int32(*instance.Spec.Features.AstartePodPriorities.AstarteLowPriority)
			priorityClassLow.Description = "Astarte low-priority pods should be in this priority class."
			return nil
		}); err != nil {
			return err
		} else {
			misc.LogCreateOrUpdateOperationResult(log, result, instance, priorityClassLow)
		}
	}
	return nil
}

func GetDefaultAstartePriorityClassNameForComponent(component apiv1alpha2.AstarteComponent) string {
	return defaultComponentPriorityClass[component]
}
