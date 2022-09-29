package utils

import (
	goctx "context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	operator "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha2"
)

// AstarteDeleteTest deletes an Astarte instance and tests whether it was deleted and cleaned up
//nolint
func AstarteDeleteTest(c client.Client, namespace string) error {
	installedAstarte := &operator.Astarte{}
	// use Context's helper to Get the object
	if err := c.Get(goctx.TODO(), types.NamespacedName{Name: AstarteTestResource.GetName(), Namespace: namespace}, installedAstarte); err != nil {
		return err
	}

	// Delete the object
	if err := c.Delete(goctx.TODO(), installedAstarte); err != nil {
		return err
	}

	// Wait until everything in the namespace is erased. Finalizers should do the job.
	if err := wait.Poll(DefaultRetryInterval, DefaultTimeout, func() (done bool, err error) {
		deployments := &appsv1.DeploymentList{}
		if err = c.List(goctx.TODO(), deployments, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if len(deployments.Items) > 0 {
			return false, nil
		}

		statefulSets := &appsv1.StatefulSetList{}
		if err = c.List(goctx.TODO(), statefulSets, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if len(statefulSets.Items) > 0 {
			return false, nil
		}

		configMaps := &v1.ConfigMapList{}
		if err = c.List(goctx.TODO(), configMaps, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if l := len(configMaps.Items); l > 1 {
			return false, nil
		} else if l == 1 {
			// From Kubernetes 1.20+, a configmap named "kube-root-ca.crt" is created by default
			// in every namespace. If it's the only item left, let it be.
			if configMaps.Items[0].Name != "kube-root-ca.crt" {
				return false, nil
			}
		}

		secrets := &v1.SecretList{}
		if err = c.List(goctx.TODO(), secrets, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		// The Default Token is acceptable.
		if len(secrets.Items) > 1 {
			return false, nil
		}

		pvcs := &v1.PersistentVolumeClaimList{}
		if err = c.List(goctx.TODO(), pvcs, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if len(pvcs.Items) > 0 {
			return false, nil
		}

		return true, nil
	}); err != nil {
		return err
	}

	return nil
}
