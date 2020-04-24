package utils

import (
	goctx "context"
	"fmt"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operator "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

// AstarteDeleteTest deletes an Astarte instance and tests whether it was deleted and cleaned up
func AstarteDeleteTest(f *framework.Framework, ctx *framework.Context) error {
	namespace, err := ctx.GetWatchNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}
	installedAstarte := &operator.Astarte{}
	// use Context's helper to Get the object
	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: AstarteTestResource.GetName(), Namespace: namespace}, installedAstarte); err != nil {
		return err
	}

	// Delete the object
	if err := f.Client.Delete(goctx.TODO(), installedAstarte); err != nil {
		return err
	}

	// Wait until everything in the namespace is erased. Finalizers should do the job.
	if err := wait.Poll(DefaultRetryInterval, DefaultTimeout, func() (done bool, err error) {
		deployments := &appsv1.DeploymentList{}
		if err = f.Client.List(goctx.TODO(), deployments, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if len(deployments.Items) > 0 {
			return false, nil
		}

		statefulSets := &appsv1.StatefulSetList{}
		if err = f.Client.List(goctx.TODO(), statefulSets, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if len(statefulSets.Items) > 0 {
			return false, nil
		}

		configMaps := &v1.ConfigMapList{}
		if err = f.Client.List(goctx.TODO(), configMaps, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		if len(configMaps.Items) > 0 {
			return false, nil
		}

		secrets := &v1.SecretList{}
		if err = f.Client.List(goctx.TODO(), secrets, client.InNamespace(namespace)); err != nil {
			return false, err
		}
		// The Default Token is acceptable.
		if len(secrets.Items) > 1 {
			return false, nil
		}

		pvcs := &v1.PersistentVolumeClaimList{}
		if err = f.Client.List(goctx.TODO(), pvcs, client.InNamespace(namespace)); err != nil {
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
