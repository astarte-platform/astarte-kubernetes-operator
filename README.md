# Astarte Kubernetes Operator

![CI](https://github.com/astarte-platform/astarte-kubernetes-operator/workflows/Operator%20e2e%20tests/badge.svg?branch=release-0.11)
[![Go Report Card](https://goreportcard.com/badge/github.com/astarte-platform/astarte-kubernetes-operator)](https://goreportcard.com/report/github.com/astarte-platform/astarte-kubernetes-operator)
[![License](http://img.shields.io/:license-apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.html)
![Docker Pulls](https://img.shields.io/docker/pulls/astarte/astarte-kubernetes-operator)

Astarte Kubernetes Operator runs and manages an Astarte Cluster in a Kubernetes Cluster. It is meant to
work on any Managed Kubernetes installation, and leverages a number of Kubernetes features to ensure
Astarte runs as smooth as possible. It also handles upgrades, monitoring, and more.

Astarte Operator is the foundation of any Astarte installation, and you can find more information about it
and how to use it once installed in the
[Astarte Administration guide](https://docs.astarte-platform.org/0.11/001-intro_administrator.html).

## Getting started

The preferred way to install and manage Astarte Operator is through [astartectl](https://github.com/astarte-platform/astartectl).
Simply run `astartectl cluster install-operator` to install the Operator in your cluster.

`astartectl` also intermediates all Operator interactions, including generation of `Astarte` resources and upgrades.
Run `astartectl cluster instance deploy` to get started with your Astarte instance immediately.
You can find more information about `astartectl` installations in the
[relevant chapter of Astarte's documentation](https://docs.astarte-platform.org/latest/060-setup_cluster.html#using-astartectl).

On the other hand, if you feel like handling all of this on your own (or if you just want to learn more about the process),
you should follow the [Astarte Administration guide](https://docs.astarte-platform.org/0.11/001-intro_administrator.html)
in Astarte's Documentation.

## Kubernetes support

| Kubernetes Version | Supported | Tested by CI |
| --- | --- | --- |
| v1.11.x  | :x: | :x: |
| v1.12.x  | :large_orange_diamond: | :x: |
| v1.13.x  | :large_orange_diamond: | :x: |
| v1.14.x  | :white_check_mark: | :x: |
| v1.15.x  | :white_check_mark: | :x: |
| v1.16.x  | :white_check_mark: | :white_check_mark: |
| v1.17.x  | :white_check_mark: | :white_check_mark: |
| v1.18.x  | :white_check_mark: | :white_check_mark: |

Key:

* :white_check_mark: : Supported and stable
* :large_orange_diamond: : Partially supported / known to run in production, but not being targeted by the release.
* :x: : Not supported. Run at your own risk

## Development

Astarte's Operator is written in Go and built upon [Operator SDK](https://github.com/operator-framework/operator-sdk).
It depends on Go 1.14.x, and requires Go Modules.
