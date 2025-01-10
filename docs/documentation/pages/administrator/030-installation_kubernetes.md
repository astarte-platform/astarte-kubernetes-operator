# Installing Astarte Operator

The most simple and common installation procedure exploits the [Astarte Operator's Helm
chart](https://artifacthub.io/packages/helm/astarte/astarte-operator).

Helm is intended to be used as the operator's lifecycle management tool, thus make sure you are
ready with a working [Helm installation](https://helm.sh/docs/intro/install/).

Please, before starting with the Operator's install procedure make sure that any
[prerequisite](020-prerequisites.html) has been satisfied.

## Installation

Installing the Operator is as simple as

```bash
$ helm repo add astarte https://helm.astarte-platform.org
$ helm repo update
$ helm install astarte-operator astarte/astarte-operator -n astarte-operator
```

This command will take care of installing all needed components for the Operator to run. This
includes all the RBAC roles, Custom Resource Definitions, Webhooks, and the Operator itself.

You can use the `--version` switch to specify a version to install. When not specified, the latest
stable version will be installed instead.

## Upgrading the Operator

The procedure for upgrading the Operator depends on the version of the Operator you want to upgrade
from. Please refer to the [Upgrade Guide](000-upgrade_index.html) section that fits your needs.

## Uninstalling the Operator

Uninstalling the Operator is as simple as:

```bash
$ helm uninstall astarte-operator -n astarte-operator
```

Starting from v24.5.0, the removal of the Operator preserves the Astarte, AstarteDefaultIngress and
Flow CRDs. To prevent unwanted deletion of the deployed custom resources, the removal of the CRDs
must be performed manually.

Please be aware that the Operator is meant to handle the full lifecycle of the Astarte,
AstarteDefaultIngress and Flow resources. If your services are still up and running when the
Operator is uninstalled you might experience limited functionalities (e.g. even if flow creation
succeeds, there is no guarantee that the flow will actually start).
