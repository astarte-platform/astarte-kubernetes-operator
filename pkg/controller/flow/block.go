package flow

import (
	"context"
	"encoding/json"
	"fmt"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/pkg/apis/api/v1alpha1"
	"github.com/astarte-platform/astarte-kubernetes-operator/pkg/misc"
	"github.com/imdario/mergo"
	"github.com/openlyinc/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func ensureBlock(cr *apiv1alpha1.Flow, block apiv1alpha1.ContainerBlockSpec, astarte *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) error {
	// In this, we'll have to orchestrate all workers, including bringing them up/down on demand
	baseLabels := map[string]string{
		"component":  "astarte-flow",
		"flow-block": block.BlockID,
		"flow-name":  cr.Name,
	}

	// Reconcile the main Secret for User Configuration
	blockSecretLabels := map[string]string{
		"flow-component":          "block",
		"flow-configuration-type": "block-configuration",
	}
	if err := mergo.Merge(&blockSecretLabels, baseLabels); err != nil {
		return err
	}
	if _, err := misc.ReconcileSecretStringWithLabels(generateBlockName(cr, block, astarte), map[string]string{"config.json": block.Configuration},
		baseLabels, cr, c, scheme, log); err != nil {
		return err
	}

	// Start creating the common configuration
	defaultRmqConfig, err := generateDefaultRabbitMQConfiguration(astarte, c)
	if err != nil {
		return err
	}

	// Reconcile all secrets for workers. But first, ensure we list them.
	baseWorkerLabels := map[string]string{
		"flow-component":          "worker",
		"flow-configuration-type": "worker-configuration",
	}
	if e := mergo.Merge(&baseWorkerLabels, baseLabels); e != nil {
		return e
	}
	workerSecretList := &v1.SecretList{}
	if e := c.List(context.TODO(), workerSecretList, client.InNamespace(cr.Namespace), client.MatchingLabels(baseWorkerLabels)); e != nil {
		return e
	}
	existingWorkerSecrets := map[string]v1.Secret{}
	for _, s := range workerSecretList.Items {
		existingWorkerSecrets[s.Name] = s
	}

	// Time to iterate over the Workers to generate/reconcile individual configurations
	existingWorkerSecrets, workerContainers, err := getFlowWorkerContainers(existingWorkerSecrets, baseLabels, defaultRmqConfig, cr, block, astarte, c, scheme)
	if err != nil {
		return err
	}

	// Set up the Block Deployment and reconcile.
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: generateBlockName(cr, block, astarte), Namespace: cr.Namespace}}
	result, err := controllerutil.CreateOrUpdate(context.TODO(), c, deployment, func() error {
		if e := controllerutil.SetControllerReference(cr, deployment, scheme); e != nil {
			return e
		}

		blockLabels := map[string]string{
			"flow-component": "block",
		}
		if e := mergo.Merge(&blockLabels, baseLabels); e != nil {
			return e
		}

		// Assign the Spec.
		deployment.ObjectMeta.Labels = blockLabels
		deployment.Spec = appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: blockLabels},
			// Use Recreate, as we don't want to be in the situation where multiple replicas are alive.
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: blockLabels,
				},
				Spec: v1.PodSpec{
					Containers:       workerContainers,
					Volumes:          generateVolumesFor(cr, block, astarte),
					ImagePullSecrets: block.ImagePullSecrets,
				},
			},
			// This is always 1. We're controlling sharding on our own.
			Replicas: pointy.Int32(1),
		}

		return nil
	})
	if err != nil {
		return err
	}

	misc.LogCreateOrUpdateOperationResult(log, result, cr, deployment)

	// Do we need to clean up any secrets?
	if len(existingWorkerSecrets) > 0 {
		for _, wS := range existingWorkerSecrets {
			if err := c.Delete(context.TODO(), &wS); err != nil {
				return err
			}
		}
	}

	return nil
}

func getFlowWorkerContainers(existingWorkerSecrets map[string]v1.Secret, baseLabels map[string]string, defaultRmqConfig apiv1alpha1.RabbitMQConfig,
	cr *apiv1alpha1.Flow, block apiv1alpha1.ContainerBlockSpec, astarte *apiv1alpha1.Astarte, c client.Client, scheme *runtime.Scheme) (map[string]v1.Secret, []v1.Container, error) {
	workerContainers := []v1.Container{}
	for _, w := range block.Workers {
		workerName := generateWorkerName(cr, block, w, astarte)
		workerLabels := map[string]string{
			"flow-component":          "worker",
			"flow-configuration-type": "worker-configuration",
			"flow-worker":             w.WorkerID,
		}
		if err := mergo.Merge(&workerLabels, baseLabels); err != nil {
			return nil, nil, err
		}
		workerConfiguration, err := generateWorkerConfigurationFor(w, defaultRmqConfig)
		if err != nil {
			return nil, nil, err
		}
		if _, err := misc.ReconcileSecretStringWithLabels(workerName, workerConfiguration, workerLabels, cr, c, scheme, log); err != nil {
			return nil, nil, err
		}

		// Remove from the "existing" list
		delete(existingWorkerSecrets, workerName)

		workerContainers = append(workerContainers, ensureWorkerContainer(block, w))
	}

	return existingWorkerSecrets, workerContainers, nil
}

func ensureWorkerContainer(block apiv1alpha1.ContainerBlockSpec, worker apiv1alpha1.BlockWorker) v1.Container {
	// Set up a Deployment and reconcile.
	return v1.Container{
		// The worker ID, pretty simple
		Name:            worker.WorkerID,
		Image:           block.Image,
		Resources:       block.Resources,
		ImagePullPolicy: v1.PullAlways,
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "block-config",
				ReadOnly:  true,
				MountPath: "/config/block",
			},
			{
				Name:      fmt.Sprintf("worker-%s-config", worker.WorkerID),
				ReadOnly:  true,
				MountPath: "/config/worker",
			},
		},
		Env: block.Environment,
		// Flow Containers are a good way to snitch malicious code into the Cluster, and potentially allowing
		// Container breakout to the node. For this reason, do not trust this container and downscale its privileges
		// to the bottom.
		SecurityContext: &v1.SecurityContext{
			Privileged:               pointy.Bool(false),
			RunAsNonRoot:             pointy.Bool(true),
			AllowPrivilegeEscalation: pointy.Bool(false),
			ReadOnlyRootFilesystem:   pointy.Bool(true),
		},
	}
}

func generateVolumesFor(cr *apiv1alpha1.Flow, block apiv1alpha1.ContainerBlockSpec, astarte *apiv1alpha1.Astarte) []v1.Volume {
	// Start with the block-config volume
	ret := []v1.Volume{
		{
			Name: "block-config",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: generateBlockName(cr, block, astarte),
				},
			},
		},
	}

	// Add every worker configuration.
	for _, w := range block.Workers {
		ret = append(ret, v1.Volume{
			Name: fmt.Sprintf("worker-%s-config", w.WorkerID),
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: generateWorkerName(cr, block, w, astarte),
				},
			},
		})
	}

	return ret
}

func generateWorkerConfigurationFor(worker apiv1alpha1.BlockWorker, defaultRmqConfig apiv1alpha1.RabbitMQConfig) (map[string]string, error) {
	ret := map[string]string{}

	// Check and configure all Data Providers
	if worker.DataProvider.RabbitMQ != nil {
		rmqConfig := worker.DataProvider.RabbitMQ.DeepCopy()
		// Shall we use the Default configuration?
		if rmqConfig.RabbitMQConfig == nil {
			rmqConfig.RabbitMQConfig = defaultRmqConfig.DeepCopy()
		}

		// Re-Marshal it
		jsonBytes, err := json.Marshal(*rmqConfig)
		if err != nil {
			return ret, err
		}

		// Ready
		ret["rabbitmq.conf"] = string(jsonBytes)
	}

	return ret, nil
}

func generateBlockName(cr *apiv1alpha1.Flow, block apiv1alpha1.ContainerBlockSpec, astarte *apiv1alpha1.Astarte) string {
	workerName := fmt.Sprintf("%s-flow-%s-block-%s", astarte.Name, cr.Name, block.BlockID)

	// 253 is the "magic limit" for names in Kubernetes. Ensure we're not past that.
	// If we are, use a hash to determine the name
	if len(workerName) > 253 {
		// TODO: Do we even want to handle this?
		return workerName
	}

	return workerName
}

func generateWorkerName(cr *apiv1alpha1.Flow, block apiv1alpha1.ContainerBlockSpec, worker apiv1alpha1.BlockWorker, astarte *apiv1alpha1.Astarte) string {
	workerName := fmt.Sprintf("%s-flow-%s-block-%s-%s", astarte.Name, cr.Name, block.BlockID, worker.WorkerID)

	// 253 is the "magic limit" for names in Kubernetes. Ensure we're not past that.
	// If we are, use a hash to determine the name
	if len(workerName) > 253 {
		// TODO: Do we even want to handle this?
		return workerName
	}

	return workerName
}

func generateDefaultRabbitMQConfiguration(astarte *apiv1alpha1.Astarte, c client.Client) (apiv1alpha1.RabbitMQConfig, error) {
	_, _, username, password, err := misc.GetRabbitMQCredentialsFor(astarte, c)
	if err != nil {
		return apiv1alpha1.RabbitMQConfig{}, err
	}
	host, port := misc.GetRabbitMQHostnameAndPort(astarte)

	return apiv1alpha1.RabbitMQConfig{
		Host: host,
		Port: port,
		// TODO: Change when we start supporting SSL connections to RabbitMQ
		SSL:      pointy.Bool(false),
		Username: username,
		Password: password,
	}, nil
}
