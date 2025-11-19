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

// ----- Astarte Component Probes -----
// getAstarteComponentStartupProbe returns the custom startup probe if set, nil otherwise
func getAstarteComponentStartupProbe(res apiv2alpha1.AstarteGenericClusteredResource) *v1.Probe {
	if res.StartupProbe != nil {
		return res.StartupProbe
	}

	// We do not set any StartupProbe by default
	return nil
}

// getAstarteBackendReadinessProbe returns the custom readiness probe if set, the default readiness probe otherwise
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

// getAstarteDefaultProbe returns the default probe for Astarte components
func getAstarteDefaultProbe(component apiv2alpha1.AstarteComponent) *v1.Probe {
	// Custom components
	if component == apiv2alpha1.Housekeeping {
		// We need a much longer timeout, as we have an initialization which happens 3 times
		return getDefaultProbeWithThreshold("/health", 15)
	}

	// The rest are generic probes on /health
	return getDefaultProbeWithThreshold("/health", 5)
}

// ----- CFSSL Probes -----
// getCFSSLDefaultProbe returns the default probe for CFSSL
func getCFSSLDefaultProbe() *v1.Probe {
	// Start checking after 10 seconds, every 20 seconds, fail after the 3rd attempt
	return &v1.Probe{
		ProbeHandler:        v1.ProbeHandler{HTTPGet: &v1.HTTPGetAction{Path: "/api/v1/cfssl/health", Port: intstr.FromString("http")}},
		InitialDelaySeconds: 10,
		TimeoutSeconds:      5,
		PeriodSeconds:       20,
		FailureThreshold:    3,
	}
}

// getCFSSLReadinessProbe returns the readiness probe for CFSSL
func getCFSSLReadinessProbe(cr *apiv2alpha1.Astarte) *v1.Probe {
	if cr.Spec.CFSSL.ReadinessProbe != nil {
		return cr.Spec.CFSSL.ReadinessProbe
	}
	return getCFSSLDefaultProbe()
}

// getCFSSLLivenessProbe returns the liveness probe for CFSSL
func getCFSSLLivenessProbe(cr *apiv2alpha1.Astarte) *v1.Probe {
	if cr.Spec.CFSSL.LivenessProbe != nil {
		return cr.Spec.CFSSL.LivenessProbe
	}
	return getCFSSLDefaultProbe()
}

// getCFSSLStartupProbe returns the startup probe for CFSSL
func getCFSSLStartupProbe(cr *apiv2alpha1.Astarte) *v1.Probe {
	if cr.Spec.CFSSL.StartupProbe != nil {
		return cr.Spec.CFSSL.StartupProbe
	}
	// We do not set any StartupProbe by default
	return nil
}

// ----- VerneMQ Probes -----
// getVerneMQDefaultProbe returns the default probe for VerneMQ
func getVerneMQDefaultProbe() *v1.Probe {
	// Start checking after 1 minute, every 20 seconds, fail after the 3rd attempt
	return &v1.Probe{
		ProbeHandler:        v1.ProbeHandler{HTTPGet: &v1.HTTPGetAction{Path: "/metrics", Port: intstr.FromInt(8888)}},
		InitialDelaySeconds: 60,
		TimeoutSeconds:      10,
		PeriodSeconds:       20,
		FailureThreshold:    3,
	}
}

// getVerneMQReadinessProbe returns the readiness probe for VerneMQ
func getVerneMQReadinessProbe(cr *apiv2alpha1.Astarte) *v1.Probe {
	if cr.Spec.VerneMQ.ReadinessProbe != nil {
		return cr.Spec.VerneMQ.ReadinessProbe
	}
	return getVerneMQDefaultProbe()
}

// getVerneMQLivenessProbe returns the liveness probe for VerneMQ
func getVerneMQLivenessProbe(cr *apiv2alpha1.Astarte) *v1.Probe {
	if cr.Spec.VerneMQ.LivenessProbe != nil {
		return cr.Spec.VerneMQ.LivenessProbe
	}
	return getVerneMQDefaultProbe()
}

// getVerneMQStartupProbe returns the startup probe for VerneMQ
func getVerneMQStartupProbe(cr *apiv2alpha1.Astarte) *v1.Probe {
	if cr.Spec.VerneMQ.StartupProbe != nil {
		return cr.Spec.VerneMQ.StartupProbe
	}
	// We do not set any StartupProbe by default
	return nil
}
