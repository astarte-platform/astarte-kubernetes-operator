# astarte-operator

![Version: 24.11.0-dev](https://img.shields.io/badge/Version-24.11.0--dev-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 24.11.0-dev](https://img.shields.io/badge/AppVersion-24.11.0--dev-informational?style=flat-square)

The Astarte Kubernetes Operator Helm Chart.

**Homepage:** <https://github.com/astarte-platform/astarte-kubernetes-operator>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| drf | dario.freddi@ispirata.com |  |
| matt-mazzucato | mattia.mazzucato@secomind.com |  |
| annopaolo | arnaldo.cesco@secomind.com |  |

## Source Code

* <https://github.com/astarte-platform/astarte-kubernetes-operator>

## Requirements

Kubernetes: `>= 1.19.0-0`

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"astarte/astarte-kubernetes-operator"` |  |
| image.tag | string | `"snapshot"` | Overrides the image tag whose default is the chart appVersion. |
| installCRDs | bool | `true` | Whether or not to install Astarte CRDs. |
| replicaCount | int | `1` | The number of Astarte Operator replicas in your cluster. |
| resources | object | `{"limits":{"cpu":"100m","memory":"256Mi"},"requests":{"cpu":"100m","memory":"128Mi"}}` | Resources to assign to each Astarte Operator instance. |

