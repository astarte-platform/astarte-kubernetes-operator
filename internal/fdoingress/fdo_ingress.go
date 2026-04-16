package fdoingress

import (
	"context"
	"fmt"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/internal/misc"
	networkingv1 "k8s.io/api/networking/v1"

	fdoingress "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v2alpha1"
	"github.com/go-logr/logr"
	"go.openly.dev/pointy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func getFDOIngressName(cr *fdoingress.AstarteFDOIngress) string {
	return cr.Name + "-fdo-ingress"
}

func EnsureFDOIngress(cr *fdoingress.AstarteFDOIngress, parent *apiv2alpha1.Astarte, c client.Client, scheme *runtime.Scheme, log logr.Logger) (err error) {
	ingressName := getFDOIngressName(cr)

	// Reconcile the Ingress
	ingress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: ingressName, Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, ingress, func() error {
		if e := controllerutil.SetControllerReference(cr, ingress, scheme); e != nil {
			return e
		}

		ingress.SetAnnotations(getHAProxyIngressAnnotations())
		ingress.Spec = getFDOIngressSpec(cr, parent)

		return nil
	})

	if err == nil {
		misc.LogCreateOrUpdateOperationResult(log, result, cr, ingress)
	}

	return err
}

func getFDOIngressSpec(cr *fdoingress.AstarteFDOIngress, parent *apiv2alpha1.Astarte) networkingv1.IngressSpec {
	ingressSpec := networkingv1.IngressSpec{
		// define which ingress controller will implement the ingress
		IngressClassName: pointy.String(cr.Spec.IngressClass),
		TLS:              getIngressTLS(cr, parent),
		Rules:            getFDOIngressRules(parent),
	}

	return ingressSpec
}

func getHAProxyIngressAnnotations() map[string]string {
	annotations := map[string]string{
		"haproxy.org/backend-config-snippet": getHAProxyBackendConfig(),
	}
	return annotations
}

func getHAProxyBackendConfig() string {
	return "http-request set-var(txn.subdomain) req.hdr(host),field(1,'.')\n" +
		"http-request replace-path ^/(.*) /v1/%[var(txn.subdomain)]/\\1\n"
}

func getFDOIngressRules(parent *apiv2alpha1.Astarte) (ingressRules []networkingv1.IngressRule) {
	pathTypePrefix := networkingv1.PathTypePrefix
	pairingComponent := apiv2alpha1.Pairing

	// Generate API Paths
	var apiPaths []networkingv1.HTTPIngressPath

	apiPaths = append(apiPaths, networkingv1.HTTPIngressPath{
		Path:     "/",
		PathType: &pathTypePrefix,
		Backend: networkingv1.IngressBackend{
			Service: &networkingv1.IngressServiceBackend{
				Name: parent.Name + "-" + pairingComponent.ServiceName(),
				Port: networkingv1.ServiceBackendPort{Name: "http"},
			},
		},
	})

	// Add API host with all API paths
	ingressRules = append(ingressRules, networkingv1.IngressRule{
		// We have only one rule, used for the FDO pairing service
		Host: getFDOHost(parent),
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: apiPaths,
			},
		},
	})

	return ingressRules
}

func getIngressTLS(cr *fdoingress.AstarteFDOIngress, parent *apiv2alpha1.Astarte) (ingressTLS []networkingv1.IngressTLS) {
	if cr.Spec.TLSSecret == "" {
		return nil
	}

	ingressTLS = append(ingressTLS, networkingv1.IngressTLS{
		Hosts:      []string{getFDOHost(parent)},
		SecretName: cr.Spec.TLSSecret,
	})

	return ingressTLS
}

// getFDOHost returns the host for the FDO pairing service, which is a wildcard subdomain of the API host.
func getFDOHost(parent *apiv2alpha1.Astarte) string {
	return fmt.Sprintf("*.%s", parent.Spec.API.Host)
}
