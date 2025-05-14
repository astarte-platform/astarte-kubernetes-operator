package defaultingress

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha2"
	ingressv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/ingress/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/lib/misc"
)

func EnsureIngressControllerConfiguration(cr *ingressv1alpha1.AstarteDefaultIngress, parent *apiv1alpha2.Astarte, c client.Client, scheme *runtime.Scheme, log logr.Logger) error {
	// Create a ConfigMap with custom headers (global)
	customHeadersConfigName := "custom-nginx-headers"
	customHeaders := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: customHeadersConfigName, Namespace: getIngressControllerNamespace(cr)}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, customHeaders, func() error {
		customHeaders.Data = map[string]string{
			"X-XSS-Protection":       "1; mode=block",              // stops pages from loading when they detect reflected cross-site scripting (XSS) attacks.
			"X-Content-Type-Options": "nosniff",                    // prevent web browser from sniffing a response away from the declared Content-Type.
			"Referrer-Policy":        "no-referrer-when-downgrade", // the URL is sent as a referrer when the protocol security level stays the same
		}

		return nil
	})

	if err != nil {
		return err
	}

	misc.LogCreateOrUpdateOperationResult(log, result, cr, customHeaders)

	ingressController := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: getIngressControllerName(cr), Namespace: getIngressControllerNamespace(cr)}}
	result, err = controllerutil.CreateOrUpdate(context.TODO(), c, ingressController, func() error {

		serverSnippetValue := fmt.Sprint("location ~* \"/(appengine|flow|housekeeping|pairing|realmmanagement)/metrics\" {\n" +
			"  deny all;\n" +
			"  return 404;\n" +
			"}")

		ingressController.Data = map[string]string{
			"proxy-set-headers": customHeadersConfigName,
			"server-snippet":    serverSnippetValue,
		}

		return nil
	})

	if err != nil {
		return err
	}

	misc.LogCreateOrUpdateOperationResult(log, result, cr, ingressController)

	return err
}

func getIngressControllerNamespace(cr *ingressv1alpha1.AstarteDefaultIngress) string {
	return cr.Namespace + "-ingress-nginx"
}

func getIngressControllerName(cr *ingressv1alpha1.AstarteDefaultIngress) string {
	return cr.Name + "-ingress-nginx"
}
