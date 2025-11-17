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

package reconcile

import (
	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// getAstarteComponentStartupProbe returns the custom startup probe if set, nil otherwise
func getAstarteComponentStartupProbe(res apiv2alpha1.AstarteGenericClusteredResource) *v1.Probe {
	if res.StartupProbe != nil {
		return res.StartupProbe
	}

	// We do not set any StartupProbe by default
	return nil
}

// getAstarteBackendReadinessProbe returns the custom readyness probe if set, the default readyness probe otherwise
func getAstarteComponentReadinessProbe(component apiv2alpha1.AstarteComponent, res apiv2alpha1.AstarteGenericClusteredResource) *v1.Probe {
	if res.ReadinessProbe != nil {
		return res.ReadinessProbe
	}

	return getAstarteDefaultProbe(component)
}

// getAstarteBackendLivenessProbe returns the custom liveness probe if set, the default liveness probe otherwise
func getAstarteComponentLivenessProbe(component apiv2alpha1.AstarteComponent, res apiv2alpha1.AstarteGenericClusteredResource) *v1.Probe {
	if res.LivenessProbe != nil {
		return res.LivenessProbe
	}

	return getAstarteDefaultProbe(component)
}

// getDefaultProbeWithThreshold returns a default probe for Astarte components with a custom failure threshold
func getDefaultProbeWithThreshold(path string, threshold int32) *v1.Probe {
	return &v1.Probe{
		ProbeHandler: v1.ProbeHandler{
			HTTPGet: &v1.HTTPGetAction{
				Path: path,
				Port: intstr.FromString("http"),
			},
		},
		InitialDelaySeconds: 10,
		TimeoutSeconds:      5,
		PeriodSeconds:       30,
		FailureThreshold:    threshold,
	}
}

// getDefaultProbe returns a default probe for Astarte components
func getAstarteDefaultProbe(component apiv2alpha1.AstarteComponent) *v1.Probe {
	// Custom components
	if component == apiv2alpha1.Housekeeping {
		// We need a much longer timeout, as we have an initialization which happens 3 times
		return getDefaultProbeWithThreshold("/health", 15)
	}

	// The rest are generic probes on /health
	return getDefaultProbeWithThreshold("/health", 5)
}
