package defaultingress

import (
	"testing"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v1alpha2"
	ingressv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v1alpha1"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	networkingv1 "k8s.io/api/networking/v1"
)

func TestGetCommonIngressAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		cr       *ingressv1alpha1.AstarteDefaultIngress
		parent   *apiv1alpha2.Astarte
		expected map[string]string
	}{
		{
			name: "API SSL enabled, Dashboard SSL enabled, CORS disabled",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					API: ingressv1alpha1.AstarteDefaultIngressAPISpec{
						Cors: pointy.Bool(false),
					},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						SSL: pointy.Bool(true),
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						SSL: pointy.Bool(true),
					},
				},
			},
			expected: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
				"nginx.ingress.kubernetes.io/use-regex":      "true",
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
			},
		},
		{
			name: "API SSL disabled, Dashboard SSL disabled, CORS enabled",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					API: ingressv1alpha1.AstarteDefaultIngressAPISpec{
						Cors: pointy.Bool(true),
					},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						SSL: pointy.Bool(false),
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						SSL: pointy.Bool(false),
					},
				},
			},
			expected: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-redirect":   "false",
				"nginx.ingress.kubernetes.io/use-regex":      "true",
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
				"nginx.ingress.kubernetes.io/enable-cors":    "true",
			},
		},
		{
			name: "Default values (nil pointers) - should default API SSL to true",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					API:       ingressv1alpha1.AstarteDefaultIngressAPISpec{},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{},
				},
			},
			expected: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
				"nginx.ingress.kubernetes.io/use-regex":      "true",
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
			},
		},
		{
			name: "API SSL enabled via parent, Dashboard SSL disabled, CORS enabled",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					API: ingressv1alpha1.AstarteDefaultIngressAPISpec{
						Cors: pointy.Bool(true),
					},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						SSL: pointy.Bool(false),
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						SSL: pointy.Bool(true),
					},
				},
			},
			expected: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
				"nginx.ingress.kubernetes.io/use-regex":      "true",
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
				"nginx.ingress.kubernetes.io/enable-cors":    "true",
			},
		},
		{
			name: "Dashboard SSL enabled overrides API SSL disabled",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					API: ingressv1alpha1.AstarteDefaultIngressAPISpec{
						Cors: pointy.Bool(false),
					},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						SSL: pointy.Bool(true),
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						SSL: pointy.Bool(false),
					},
				},
			},
			expected: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
				"nginx.ingress.kubernetes.io/use-regex":      "true",
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := getCommonIngressAnnotations(tt.cr, tt.parent)
			g.Expect(result).To(Equal(tt.expected), "Expected annotations do not match actual result")
		})
	}
}

func TestGetIngressTLS(t *testing.T) {
	tests := []struct {
		name             string
		cr               *ingressv1alpha1.AstarteDefaultIngress
		parent           *apiv1alpha2.Astarte
		includeDashboard bool
		expected         []networkingv1.IngressTLS
	}{
		{
			name: "API SSL enabled with default TLS secret, no dashboard",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					TLSSecret: "default-tls-secret",
					API:       ingressv1alpha1.AstarteDefaultIngressAPISpec{},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						SSL: pointy.Bool(true),
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						Host: "api.example.com",
						SSL:  pointy.Bool(true),
					},
				},
			},
			includeDashboard: false,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"api.example.com"},
					SecretName: "default-tls-secret",
				},
			},
		},
		{
			name: "API SSL enabled with custom TLS secret",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					TLSSecret: "default-tls-secret",
					API: ingressv1alpha1.AstarteDefaultIngressAPISpec{
						TLSSecret: "api-custom-tls-secret",
					},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						SSL: pointy.Bool(false),
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						Host: "api.example.com",
						SSL:  pointy.Bool(true),
					},
				},
			},
			includeDashboard: true,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"api.example.com"},
					SecretName: "api-custom-tls-secret",
				},
			},
		},
		{
			name: "Dashboard SSL enabled with include dashboard true",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					TLSSecret: "default-tls-secret",
					API:       ingressv1alpha1.AstarteDefaultIngressAPISpec{},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						Deploy:    pointy.Bool(true),
						SSL:       pointy.Bool(true),
						Host:      "dashboard.example.com",
						TLSSecret: "dashboard-custom-tls-secret",
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						Host: "api.example.com",
						SSL:  pointy.Bool(true),
					},
				},
			},
			includeDashboard: true,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"api.example.com"},
					SecretName: "default-tls-secret",
				},
				{
					Hosts:      []string{"dashboard.example.com"},
					SecretName: "dashboard-custom-tls-secret",
				},
			},
		},
		{
			name: "Dashboard SSL enabled but include dashboard false",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					TLSSecret: "default-tls-secret",
					API:       ingressv1alpha1.AstarteDefaultIngressAPISpec{},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						Deploy:    pointy.Bool(true),
						SSL:       pointy.Bool(true),
						Host:      "dashboard.example.com",
						TLSSecret: "dashboard-custom-tls-secret",
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						Host: "api.example.com",
						SSL:  pointy.Bool(true),
					},
				},
			},
			includeDashboard: false,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"api.example.com"},
					SecretName: "default-tls-secret",
				},
			},
		},
		{
			name: "Dashboard deploy disabled",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					TLSSecret: "default-tls-secret",
					API:       ingressv1alpha1.AstarteDefaultIngressAPISpec{},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						Deploy:    pointy.Bool(false),
						SSL:       pointy.Bool(true),
						Host:      "dashboard.example.com",
						TLSSecret: "dashboard-custom-tls-secret",
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						Host: "api.example.com",
						SSL:  pointy.Bool(true),
					},
				},
			},
			includeDashboard: true,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"api.example.com"},
					SecretName: "default-tls-secret",
				},
			},
		},
		{
			name: "Dashboard SSL disabled",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					TLSSecret: "default-tls-secret",
					API:       ingressv1alpha1.AstarteDefaultIngressAPISpec{},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						Deploy:    pointy.Bool(true),
						SSL:       pointy.Bool(false),
						Host:      "dashboard.example.com",
						TLSSecret: "dashboard-custom-tls-secret",
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						Host: "api.example.com",
						SSL:  pointy.Bool(true),
					},
				},
			},
			includeDashboard: true,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"api.example.com"},
					SecretName: "default-tls-secret",
				},
			},
		},
		{
			name: "Dashboard host empty",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					TLSSecret: "default-tls-secret",
					API:       ingressv1alpha1.AstarteDefaultIngressAPISpec{},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						Deploy:    pointy.Bool(true),
						SSL:       pointy.Bool(true),
						Host:      "",
						TLSSecret: "dashboard-custom-tls-secret",
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						Host: "api.example.com",
						SSL:  pointy.Bool(true),
					},
				},
			},
			includeDashboard: true,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"api.example.com"},
					SecretName: "default-tls-secret",
				},
			},
		},
		{
			name: "Both API and Dashboard SSL disabled",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					TLSSecret: "default-tls-secret",
					API:       ingressv1alpha1.AstarteDefaultIngressAPISpec{},
					Dashboard: ingressv1alpha1.AstarteDefaultIngressDashboardSpec{
						Deploy: pointy.Bool(true),
						SSL:    pointy.Bool(false),
						Host:   "dashboard.example.com",
					},
				},
			},
			parent: &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					API: apiv1alpha2.AstarteAPISpec{
						Host: "api.example.com",
						SSL:  pointy.Bool(false),
					},
				},
			},
			includeDashboard: true,
			expected:         []networkingv1.IngressTLS{},
		},
		{
			name: "Default values (nil pointers) - should default API SSL to true",
			cr: &ingressv1alpha1.AstarteDefaultIngress{
				Spec: ingressv1alpha1.AstarteDefaultIngressSpec{
					TLSSecret: "default-tls-secret",
					API:       ingressv1alpha1.AstarteDefaultIngressAPISpec{},
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
			includeDashboard: true,
			expected: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"api.example.com"},
					SecretName: "default-tls-secret",
				},
				{
					Hosts:      []string{"dashboard.example.com"},
					SecretName: "default-tls-secret",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := getIngressTLS(tt.cr, tt.parent, tt.includeDashboard)
			g.Expect(result).To(Equal(tt.expected), "Expected TLS configuration does not match actual result")
		})
	}
}
