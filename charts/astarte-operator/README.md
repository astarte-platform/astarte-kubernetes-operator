# astarte-operator

![Version: 26.7.0-rc.2](https://img.shields.io/badge/Version-26.7.0--rc.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 26.7.0-rc.2](https://img.shields.io/badge/AppVersion-26.7.0--rc.2-informational?style=flat-square)

The Astarte Kubernetes Operator Helm Chart.

**Homepage:** <https://github.com/astarte-platform/astarte-kubernetes-operator>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| matt-mazzucato | mattia.mazzucato@secomind.com |  |
| annopaolo | arnaldo.cesco@secomind.com |  |
| lucamarchiori | luca.marchiori@secomind.com |  |
| guicrocetti | guilherme.crocetti@secomind.com |  |

## Source Code

* <https://github.com/astarte-platform/astarte-kubernetes-operator>

## Requirements

Kubernetes: `>= 1.24.0-0`

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"astarte/astarte-kubernetes-operator"` |  |
| image.tag | string | `"26.7.0-rc.2"` | Overrides the image tag whose default is the chart appVersion. |
| installCRDs | bool | `true` | Whether or not to install Astarte CRDs. |
| replicaCount | int | `1` | The number of Astarte Operator replicas in your cluster. |
| resources | object | `{"limits":{"cpu":"100m","memory":"256Mi"},"requests":{"cpu":"100m","memory":"128Mi"}}` | Resources to assign to each Astarte Operator instance. |
