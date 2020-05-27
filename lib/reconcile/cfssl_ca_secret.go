/*
  This file is part of Astarte.

  Copyright 2020 Ispirata Srl

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

// TODO: Deprecate and kill this file when we are no longer supporting < 1.0

package reconcile

import (
	"context"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/openlyinc/pointy"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EnsureCFSSLCASecret reconciles CFSSL's CA Secret
func EnsureCFSSLCASecret(cr *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	jobName := cr.Name + "-cfssl-ca-secret-job"
	secretName := cr.Name + "-cfssl-ca"
	serviceAccountName := jobName
	// First of all, ensure we have the right roles.
	if pointy.BoolValue(cr.Spec.RBAC, true) {
		if err := reconcileStandardRBACForClusteringForApp(jobName, getCFSSLCAJobPolicyRules(), cr, c, scheme); err != nil {
			return err
		}
	} else {
		serviceAccountName = ""
	}

	// Now - is the secret there?
	secretThere := false
	theSecret := &v1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: cr.Namespace}, theSecret); err == nil {
		// The secret is on.
		secretThere = true
	}
	// Is the Job up and running?
	jobThere := false
	theJob := &batchv1.Job{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: jobName, Namespace: cr.Namespace}, theJob); err == nil {
		// The Job is on.
		jobThere = true
	}

	// Let's see what to do.
	reqLogger := log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	switch {
	case secretThere && jobThere:
		// Delete the Job.
		reqLogger.Info("Deleting stale CFSSL CA Job")
		return c.Delete(context.TODO(), theJob)
	case !secretThere && !jobThere:
		// Create the Job
		reqLogger.Info("Creating CFSSL CA Job")
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: cr.Namespace},
			Spec: batchv1.JobSpec{
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Name: jobName},
					Spec: v1.PodSpec{
						ServiceAccountName: serviceAccountName,
						ImagePullSecrets:   cr.Spec.ImagePullSecrets,
						RestartPolicy:      v1.RestartPolicyNever,
						Containers: []v1.Container{{
							Name:            jobName,
							Image:           getAstarteImageFromChannel("cfssl-kubernetes-secret", "latest", cr),
							ImagePullPolicy: getImagePullPolicy(cr),
							Env: []v1.EnvVar{
								{
									Name:  "CFSSL_URL",
									Value: getCFSSLURL(cr),
								},
								{
									Name:  "SECRET_NAME",
									Value: secretName,
								},
							},
						}},
					},
				},
			},
		}
		if err := controllerutil.SetControllerReference(cr, job, scheme); err != nil {
			return err
		}

		return c.Create(context.TODO(), job)
	case !secretThere && jobThere:
		// If the job failed, take action. Otherwise, just skip this.
		for _, condition := range theJob.Status.Conditions {
			if condition.Type == batchv1.JobFailed && condition.Status == v1.ConditionTrue {
				// The Job has failed, but no secret is available. Let's delete the job and wait
				// for the next reconciliation.
				reqLogger.Info("CFSSL CA Job failed. Deleting it, and waiting for next reconciliation to recreate it")
				return c.Delete(context.TODO(), theJob)
			}
		}
	}

	// We should not be here ever. But just in case.
	return nil
}

func getCFSSLCAJobPolicyRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"list"},
		},
	}
}
