# Astarte Kubernetes Operator

![CI](https://github.com/astarte-platform/astarte-kubernetes-operator/workflows/Operator%20e2e%20tests/badge.svg?branch=release-1.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/astarte-platform/astarte-kubernetes-operator)](https://goreportcard.com/report/github.com/astarte-platform/astarte-kubernetes-operator)
[![License](http://img.shields.io/:license-apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.html)
![Docker Pulls](https://img.shields.io/docker/pulls/astarte/astarte-kubernetes-operator)

<img src="mascotte.svg" align="right" width="160px" />
Astarte Kubernetes Operator runs and manages an Astarte Cluster in a Kubernetes Cluster. It is meant to
work on any Managed Kubernetes installation, and leverages a number of Kubernetes features to ensure
Astarte runs as smooth as possible. It also handles upgrades, monitoring, and more.

Astarte Operator is the foundation of any Astarte installation, and you can find more information about it
and how to use it once installed in the
[Astarte Administration guide](https://docs.astarte-platform.org/1.0/001-intro_administrator.html).

## Getting started

The preferred way to install and manage Astarte Operator leverages its [Helm
chart](https://artifacthub.io/packages/helm/astarte/astarte-operator).

Astarte Operator requires [`cert-manager`](https://cert-manager.io/) (`v1.7+`) to be installed in
the cluster in its default configuration. If you are using `cert-manager` in your cluster already
you don't need to take any action - otherwise, you will need to install it. A complete overview on
prerequisites can be found
[here](https://docs.astarte-platform.org/1.0/020-prerequisites.html#content).

To install `cert-manager` simply run:
```bash
helm repo add jetstack https://charts.jetstack.io
helm repo update
kubectl create namespace cert-manager
helm install \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --version v1.7.0 \
  --set installCRDs=true
```

Installing the operator is as simple as:
```bash
helm repo add astarte https://helm.astarte-platform.org
helm repo update
helm install astarte-operator astarte/astarte-operator
```

These instructions will take care of installing all needed components for the Operator to run. This
includes all the RBAC roles, Custom Resource Definitions, Webhooks, and the Operator itself.

Moreover, Helm is responsible for upgrading the Astarte Operator. To do so, run:
```bash
helm upgrade astarte-operator astarte/astarte-operator
```

If you are interested in having a deeper understanding on how the Astarte Operator works you should
follow the [Astarte Administration
guide](https://docs.astarte-platform.org/1.0/001-intro_administrator.html#content) in Astarte's
Documentation.

### What's next?

Once your Astarte Operator is up and running in your cluster, it will be time to deploy your Astarte
instance! All you have to do is [Setting up the
Cluster](https://docs.astarte-platform.org/1.0/060-setup_cluster.html#content) as described in
Astarte's Documentation.

## Kubernetes support

| Kubernetes Version | Supported                        | Tested by CI                     |
| ------------------ | -------------------------------- | -------------------------------- |
| v1.18.x            | :x:                              | :x:                              |
| v1.19.x            | :large_orange_diamond: :warning: | :large_orange_diamond: :warning: |
| v1.20.x            | :white_check_mark: :warning:     | :white_check_mark: :warning:     |
| v1.21.x            | :white_check_mark: :warning:     | :white_check_mark: :warning:     |
| v1.22.x            | :white_check_mark:               | :white_check_mark:               |
| v1.23.x            | :white_check_mark:               | :white_check_mark:               |

Key:

* :white_check_mark: : Supported and stable.
* :large_orange_diamond: : Partially supported / known to run in production, but not being targeted
  by the release.
* :x: : Not supported. Run at your own risk.
* :warning: : Kubernetes version supporting AstarteVoyagerIngress. Please, be aware that the
  AstarteVoyagerIngress is deprecated and the new AstarteDefaultIngress should be used.

## Development

Astarte's Operator is written in Go and built upon [Operator
SDK](https://github.com/operator-framework/operator-sdk). It depends on Go 1.17.x, requires Go
Modules and Kubernetes v1.20+.

The project is built with kustomize v3.8.7, controller-gen v0.8.0 and conversion-gen v0.19.16.
