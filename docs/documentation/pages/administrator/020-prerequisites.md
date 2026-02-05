# Prerequisites

As much as Astarte's Operator is capable of creating a completely self-contained installation,
there's a number of prerequisites to be fulfilled depending on the use case.

## On your machine

The following tools are required within your local machine:

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/): must be compatible with your
  target Kubernetes version,
- [astartectl](https://github.com/astarte-platform/astartectl): your version must be the same of the
  Astarte Operator running in your cluster,
- [helm](https://helm.sh/): v3 is required.

## HAProxy
Astarte Operator is capable of interacting with HAProxy Ingress Controller through its dedicated
`AstarteDefaultIngress` resource, as long as an [HAProxy ingress
controller](https://www.haproxy.com/documentation/kubernetes-ingress/) is installed. 

Please, be aware that trying to deploy multiple ingress controllers in your cluster may result in all
of them trying simultaneously to handle the Astarte ingress resource. Consider using ingress classes
for avoiding confusing situations as outlined
[here](https://kubernetes.github.io/ingress-nginx/user-guide/multiple-ingress/).

### Helm installation via CLI
Installing the ingress controller is as simple as running a few `helm` commands:
```bash
helm repo add haproxytech https://haproxytech.github.io/helm-charts
helm install haproxy-kubernetes-ingress haproxytech/kubernetes-ingress \
  --create-namespace \
  --namespace haproxy-controller \
  --set controller.service.externalTrafficPolicy=Local \
  --set controller.service.type=LoadBalancer \
  --set controller.service.enablePorts.quic=false \
  --set controller.service.loadBalancerIP=<your-desired-static-ip>
```

## NGINX
Starting from Astarte Operator `v25.5.x`, HAProxy is the default and preferred Ingress Controller
for Astarte deployments. This is the last version of the Astarte Operator that will support NGINX as Ingress
Controller due to [Ingress NGINX retirement.](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/)

The annotation `ingress.astarte-platform.org/ingress-controller-selector` in the ADI CR can be used to specify which Ingress Controller
the Astarte Operator should use. 

By default, the Operator assumes the HAPROXY Ingress Controller is in use, in fact the annotation is set by default to:
```yaml
ingress.astarte-platform.org/ingress-controller-selector: "haproxy.org"
```
If you want to still use NGINX, you will have to set the annotation to:
```yaml
ingress.astarte-platform.org/ingress-controller-selector: "nginx.ingress.kubernetes.io"
```

Depending on the value of this annotation, the Operator will create Ingress resources compatible with the selected Ingress Controller.

## cert-manager

Astarte requires [`cert-manager`](https://cert-manager.io/) to be installed in the cluster in its
default configuration (installed in namespace `cert-manager` as `cert-manager`). If you are using
`cert-manager` in your cluster already you don't need to take any action - otherwise, you will need
to install it.

Astarte is actively tested with `cert-manager` 1.16.3, but should work with any 1.0+ releases of
`cert-manager`. If your `cert-manager` release is outdated, please consider upgrading to a newer
version according to [this guide](https://cert-manager.io/docs/installation/upgrading/).

[`cert-manager` documentation](https://cert-manager.io/docs/installation/) details all needed steps
to have a working instance on your cluster. However, in case you won't be using `cert-manager` for
other components beyond Astarte or, in general, if you don't have very specific requirements, it is
advised to install it through its Helm chart. To do so, run the following commands:

```bash
$ helm repo add jetstack https://charts.jetstack.io
$ helm repo update
$ kubectl create namespace cert-manager
$ helm install \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --version v1.16.3 \
  --set crds.enabled=true
```

This will install `cert-manager` and its CRDs in the cluster.

## External RabbitMQ

Starting from Astarte Operator `v25.5.x`, RabbitMQ is no longer deployed by the Astarte Operator.
If you previously relied on a RabbitMQ instance managed by the Astarte Operator, upgrading shall be preceded by
provisioning an external RabbitMQ instance, otherwise, after the upgrade you will end up without any AMQP
broker in place. Consider using RabbitMQ deployed by the [RabbitMQ Cluster Operator]
(https://www.rabbitmq.com/kubernetes/operator/operator-overview) or any other managed solution that
you prefer.

## External Cassandra / Scylla

Starting from Astarte Operator `v25.5.x`, Cassandra / Scylla is no longer deployed by the Astarte
Operator. If you previously relied on a Cassandra instance managed by the Astarte Operator, you shall provision an external
instance before upgrading, otherwise the upgrade will leave you without any database backing Astarte.
It is strongly advised to deploy a separate Cassandra cluster, a VM-based installation
or a managed solution.

## Kubernetes and external components

When deploying external components, it is important to take in consideration how Kubernetes behaves
with the underlying infrastructure. Most modern Cloud Providers have a concept of Virtual Private
Cloud, by which the internal Kubernetes Network stack directly integrates with their Network stack.
This, in short, enables deploying Pods in a shared private network, in which other components (such
as Virtual Machines) can be deployed.

This is the preferred, advised and supported configuration. In this scenario, there's literally no
difference between interacting with a VM or a Pod, enabling a hybrid infrastructure without having
to pay the performance cost.
