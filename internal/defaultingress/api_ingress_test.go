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

package defaultingress

import (
	"testing"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	ingressv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v2alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHAProxyAPIIngressRulesFDORoute(t *testing.T) {
	parent := &apiv2alpha1.Astarte{
		Spec: apiv2alpha1.AstarteSpec{
			API: apiv2alpha1.AstarteAPISpec{
				Host: "api.astarte.example.com",
			},
			Features: apiv2alpha1.AstarteFeatures{
				FDO: &apiv2alpha1.AstarteFDOSpec{
					Enable: true,
				},
			},
		},
	}
	cr := &ingressv2alpha1.AstarteDefaultIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				ingressv2alpha1.AnnotationIngressControllerSelector: ingressv2alpha1.HAProxySelectorValue,
			},
		},
		Spec: ingressv2alpha1.AstarteDefaultIngressSpec{
			Astarte: "test-astarte",
		},
	}

	rules := getHAProxyAPIIngressRules(cr, parent)
	if len(rules) == 0 {
		t.Fatalf("Expected ingress rules, got none")
	}

	foundFDO := false
	for _, rule := range rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Path == "/fdo" {
				foundFDO = true
				break
			}
		}
	}

	if !foundFDO {
		t.Errorf("Expected /fdo ingress route in HAProxy rules when FDO is enabled, but none was found")
	}
}

func TestNginxAPIIngressRulesFDORoute(t *testing.T) {
	parent := &apiv2alpha1.Astarte{
		Spec: apiv2alpha1.AstarteSpec{
			API: apiv2alpha1.AstarteAPISpec{
				Host: "api.astarte.example.com",
			},
			Features: apiv2alpha1.AstarteFeatures{
				FDO: &apiv2alpha1.AstarteFDOSpec{
					Enable: true,
				},
			},
		},
	}
	cr := &ingressv2alpha1.AstarteDefaultIngress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				ingressv2alpha1.AnnotationIngressControllerSelector: ingressv2alpha1.NGINXSelectorValue,
			},
		},
		Spec: ingressv2alpha1.AstarteDefaultIngressSpec{
			Astarte: "test-astarte",
		},
	}

	rules := getNgnixAPIIngressRules(cr, parent)
	if len(rules) == 0 {
		t.Fatalf("Expected ingress rules, got none")
	}

	foundFDO := false
	for _, rule := range rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Path == "/fdo(/|$)(.*)" {
				foundFDO = true
				break
			}
		}
	}

	if !foundFDO {
		t.Errorf("Expected /fdo(/|$)(.*) ingress route in Nginx rules when FDO is enabled, but none was found")
	}
}
