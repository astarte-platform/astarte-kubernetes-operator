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

## NGINX

Astarte currently features only one supported Managed Ingress, based on
[NGINX](https://nginx.org/en/). NGINX provides routing, SSL termination and more,
and as of today is the preferred/advised way to run Astarte in production.

Astarte Operator is capable of interacting with NGINX through its dedicated
`AstarteDefaultIngress` resource, as long as an [NGINX ingress
controller](https://kubernetes.github.io/ingress-nginx/) is installed. Installing the ingress
controller is as simple as running a few `helm` commands:
```bash
$ helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
$ helm repo update
$ helm install ingress-nginx ingress-nginx/ingress-nginx -n ingress-nginx \
    --set controller.service.externalTrafficPolicy=Local \
    --create-namespace
```

Please, be aware that trying to deploy multiple ingress controllers in your cluster may result in all
of them trying simultaneously to handle the Astarte ingress resource. Consider using ingress classes
for avoiding confusing situations as outlined
[here](https://kubernetes.github.io/ingress-nginx/user-guide/multiple-ingress/).

In the end, you won't need to create NGINX ingresses yourself: the Astarte Operator itself will take
care of this task.

## RabbitMQ

For production environments, consider using RabbitMQ deployed by the [RabbitMQ Cluster Operator]
(https://www.rabbitmq.com/kubernetes/operator/operator-overview) or any other managed solution that 
you prefer. The Astarte Operator includes only basic management of RabbitMQ, which is deprecated since 
v24.5 and as such it should not be relied upon when dealing with production environments. Futher details 
can be found [here](https://github.com/astarte-platform/astarte-kubernetes-operator/issues/287).

## cert-manager

Astarte requires [`cert-manager`](https://cert-manager.io/) to be installed in the cluster in its
default configuration (installed in namespace `cert-manager` as `cert-manager`). If you are using
`cert-manager` in your cluster already you don't need to take any action - otherwise, you will need
to install it.

Astarte is actively tested with `cert-manager` 1.13, but should work with any 1.0+ releases of
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
  --version v1.14.7 \
  --set installCRDs=true
```

This will install `cert-manager` and its CRDs in the cluster.

## External Cassandra / Scylla

In production deployments, it is strongly advised to have a separate Cassandra cluster interacting
with the Kubernetes installation. This is due to the fact that Cassandra Administration is a
critical topic, especially with mission critical workloads.

Astarte Operator includes only basic management of Cassandra, which is deprecated since v1.0 and as
such it should not be relied upon when dealing with production environments. Furthermore, in the
near future, Cassandra support is planned to be removed from Astarte Operator in favor of the
adoption of a dedicated Kubernetes Operator (e.g. [Scylla
Operator](https://operator.docs.scylladb.com/stable/generic.html)).

In case an external Cassandra cluster is deployed, be aware that Astarte lives on the assumption it
will be the only application managing the Cluster - as such, it is strongly advised to have a
dedicated cluster for Astarte.

## Kubernetes and external components

When deploying external components, it is important to take in consideration how Kubernetes behaves
with the underlying infrastructure. Most modern Cloud Providers have a concept of Virtual Private
Cloud, by which the internal Kubernetes Network stack directly integrates with their Network stack.
This, in short, enables deploying Pods in a shared private network, in which other components (such
as Virtual Machines) can be deployed.

This is the preferred, advised and supported configuration. In this scenario, there's literally no
difference between interacting with a VM or a Pod, enabling a hybrid infrastructure without having
to pay the performance cost.
