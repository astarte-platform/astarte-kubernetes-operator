package defaultingress

import (
	"fmt"
	"testing"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
	ingressv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestDefaultIngress() *ingressv1alpha1.AstarteDefaultIngress {
	return &ingressv1alpha1.AstarteDefaultIngress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testname",
			Namespace: "testnamespace",
		},
	}
}

func TestAPIIngressName(t *testing.T) {
	cr := createTestDefaultIngress()
	expectedName := "testname-api-ingress"
	actualName := getAPIIngressName(cr)
	if actualName != expectedName {
		t.Errorf("Expected %q, got %q", expectedName, actualName)
	}
}

func TestGetConfigMapName(t *testing.T) {
	cr := createTestDefaultIngress()
	expectedName := "testname-api-ingress-config"
	actualName := getConfigMapName(cr)
	if actualName != expectedName {
		t.Errorf("Expected %q, got %q", expectedName, actualName)
	}
}

func TestGetDashboardHost(t *testing.T) {
	tests := []struct {
		name     string
		cr       *ingressv1alpha1.AstarteDefaultIngress
		parent   *apiv1alpha2.Astarte
		expected string
	}{
		{
			name: "returns parent API host when dashboard host is empty",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						Host: "",
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						Host: "api.example.com",
					},
				},
			},
			expected: "api.example.com",
		},
		{
			name: "returns dashboard host when set",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						Host: "dashboard.example.com",
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						Host: "api.example.com",
					},
				},
			},
			expected: "dashboard.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDashboardHost(tt.cr, tt.parent); got != tt.expected {
				t.Errorf("getDashboardHost() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestGetDashboardServiceRelativePath(t *testing.T) {
	tests := []struct {
		name     string
		cr       *ingressv1alpha1.AstarteDefaultIngress
		expected string
	}{
		{
			name: "host is empty returns formatted service relative path",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						Host: "",
					},
				},
			},
			expected: fmt.Sprintf("/%s(/|$)(.*)", "dashboard"),
		},
		{
			name: "host is set returns default regex",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						Host: "dashboard.example.com",
					},
				},
			},
			expected: "/()(.*)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDashboardServiceRelativePath(tt.cr); got != tt.expected {
				t.Errorf("getDashboardServiceRelativePath() = %q, expected %q", got, tt.expected)
			}
		})
	}
}
