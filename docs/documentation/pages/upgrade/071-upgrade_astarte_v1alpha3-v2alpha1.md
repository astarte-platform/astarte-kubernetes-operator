# Astarte CR Changes from v1alpha3 to v2alpha1

## Scylla

The most significant change is that **the Astarte operator no longer manages the Cassandra cluster deployment**. Cassandra is now treated as a fully external service, and the CRD only contains the necessary connection details.

This results in the following key changes:

- **Removal of Deployment Fields**: All fields related to deploying a Cassandra cluster (e.g., `deploy`, `replicas`, `image`, `storage`) have been removed from the `cassandra` spec. You are now responsible for deploying and managing your Cassandra or ScyllaDB cluster separately.
- **Restructured `nodes`**: The `nodes` field, which was a single comma-separated string, is now a structured list of `host` and `port` objects.
- **Relocated `astarteSystemKeyspace`**: The `astarteSystemKeyspace` configuration has been moved from the top level of the `spec` into `spec.cassandra`. It also includes new fields for more advanced replication strategies.
- **Standardized Connection Spec**: The `connection` spec has been changed, renaming `secret` to `credentialsSecret` and removing deprecated or unused fields.

### Top-Level Cassandra Fields (`spec.cassandra`)

| v1alpha3 Path | v2alpha1 Path | Action Required |
| --- | --- | --- |
| `deploy` | **Removed** | **Removed.** |
| `replicas` | **Removed** | **Removed**. |
| `image` | **Removed** | **Removed**. |
| `version` | **Removed** | **Removed**. |
| `storage` | **Removed** | **Removed**. |
| `maxHeapSize` | **Removed** | **Removed**. |
| `heapNewSize` | **Removed** | **Removed**. |
| `resources` | **Removed** | **Removed**. |
| `nodes` | `connection.nodes` | **Restructured**. Convert the comma-separated string into a list of objects. For example, `"host1:9042,host2:9042"` becomes `[{"host": "host1", "port": 9042}, {"host": "host2", "port": 9042}]`. |
| `connection` | `connection` | **Kept**. Sub-fields have changed. See the section below. |
| `(none)` | `astarteSystemKeyspace` | **Moved**. The `astarteSystemKeyspace` object is now nested under `cassandra`. |

### Connection Fields (`spec.cassandra.connection`)

| v1alpha3 Path | v2alpha1 Path | Action Required |
| --- | --- | --- |
| `connection.autodiscovery` | **Removed** | **Removed**. This field is no longer used. |
| `connection.username` | **Removed** | **Removed**. Credentials must now be provided via a secret. |
| `connection.password` | **Removed** | **Removed**. Credentials must now be provided via a secret. |
| `connection.secret` | `connection.credentialsSecret` | **Renamed**. The field `secret` is now `credentialsSecret`. |
| `(none)` | `connection.enableKeepalive` | **New field**. This is a new optional boolean field that defaults to `true`. |

### Keyspace Fields

The entire `astarteSystemKeyspace` object has been moved and enhanced.

| v1alpha3 Path | v2alpha1 Path | Action Required |
| --- | --- | --- |
| `spec.astarteSystemKeyspace` | `spec.cassandra.astarteSystemKeyspace` | **Moved & Enhanced**. Move the object under `spec.cassandra`. |
| `(none)` | `astarteSystemKeyspace.replicationStrategy` | **New field**. You can now specify the replication strategy (`"SimpleStrategy"` or `"NetworkTopologyStrategy"`). Defaults to `"SimpleStrategy"`. |
| `(none)` | `astarteSystemKeyspace.dataCenterReplication` | **New field**. Use this to specify per-datacenter replication factors when using `"NetworkTopologyStrategy"`. For example: `"dc1:3,dc2:3"`. |

## RabbitMQ

RabbitMQ is now treated as an external dependency. All fields for deploying a cluster (`deploy`, `replicas`, `image`, `storage`, etc.) have been removed. The connection details are now more structured and standardized, aligning with how other external services like Cassandra are handled.

Before proceeding, ensure that you have a running RabbitMQ instance and all the necessary connection details.

### Top-Level RabbitMQ Fields (`spec.rabbitmq`)

Most fields at this level have been removed because the operator no longer deploys RabbitMQ.

| v1alpha3 Path | v2alpha1 Path | Action Required |
| --- | --- | --- |
| `deploy` | **Removed** | **Removed**. You must now manage your own RabbitMQ deployment. |
| `replicas` | **Removed** | **Removed**. |
| `image` | **Removed** | **Removed**. |
| `version` | **Removed** | **Removed**. |
| `storage` | **Removed** | **Removed**. |
| `additionalPlugins` | **Removed** | **Removed**. Ensure required plugins (like `rabbitmq_management`) are enabled in your external instance. |
| `resources`, `antiAffinity`, `customAffinity`, etc. | **Removed** | **Removed**. All deployment-related fields are gone. |
| `connection` | `spec.rabbitmq.connection` | **Kept and Restructured**. See the section below for details on its sub-fields. |

### Connection Fields (`spec.rabbitmq.connection`)

The connection spec has been updated to use a more generic and structured format.

| v1alpha3 Path | v2alpha1 Path | Action Required |
| --- | --- | --- |
| `connection.username` | **Removed** | **Removed**. Credentials must now be provided via a secret. |
| `connection.password` | **Removed** | **Removed**. Credentials must now be provided via a secret. |
| `connection.secret` | `connection.credentialsSecret` | **Renamed**. The `secret` field is now called `credentialsSecret`. |

## VerneMQ

Unlike RabbitMQ and Cassandra, the Astarte operator **still manages the VerneMQ deployment**. Most of your existing configuration will be directly transferable. No major changes have been made to the VerneMQ spec.

## CFSSL

| v1alpha3 Path | v2alpha1 Path | Action Required |
| --- | --- | --- |
| `csrRootCa` | `csrRootCa` | **Kept and Restructured**. See the section below. |
| `caRootConfig` | `caRootConfig` | **Kept and Restructured**. See the section below. |

### CSR Root CA Fields (`spec.cfssl.csrRootCa`)

The structure for defining the root CA's Certificate Signing Request has a minor change.

| v1alpha3 Path | v2alpha1 Path | Action Required |
| --- | --- | --- |
| `csrRootCa.ca.expiry` | `csrRootCa.expiry` | **Moved and Renamed**. The `expiry` string is no longer nested under a `ca` object. Move it up one level. |

### CA Root Config Fields (`spec.cfssl.caRootConfig`)

The structure for the CA's root configuration has also been slightly simplified.

| v1alpha3 Path | v2alpha1 Path | Action Required |  |
| --- | --- | --- | --- |
| `caRootConfig.signing.default` | `caRootConfig.signingDefault` | **Moved and Renamed**. The object previously at `signing.default` is now directly at `signingDefault`. |  |

## Other fields

| v1alpha3 Path | v2alpha1 Path | Action Required |
| --- | --- | --- |
| `rbac` | **Removed** | **Removed**. This field is no longer used. RBAC is now always managed by the operator. |
| `astarteSystemKeyspace` | `spec.cassandra.astarteSystemKeyspace` | **Moved**. This field is now located under the `cassandra` spec. |


## Components

The structure of the `components` spec has been significantly simplified. In `v1alpha3`, several components (`housekeeping`, `realmManagement`, `pairing`) were split into distinct `api` and `backend` objects. This separation has been removed in `v2alpha1`.

Now, **each component is a single, unified object** that combines the properties of the former `api` and `backend`. You will need to merge the configurations from both sub-objects into the new, flattened structure.

| v1alpha3 Path | v2alpha1 Path | Action Required |
| --- | --- | --- |
| `components.<name>.api` | `components.<name>` | **Merged & Flattened.** The `api` object is gone. |
| `components.<name>.backend` | `components.<name>` | **Merged & Flattened.** The `backend` object is gone. |
| `components.<name>.api.<field>` | `components.<name>.<field>` | **Moved.** Move all fields from the `api` block directly under the component's name. |
| `components.<name>.backend.<field>` | `components.<name>.<field>` | **Moved.** Move all fields from the `backend` block directly under the component's name. |

This change applies primarily to `housekeeping`, `realmManagement`, and `pairing`. Other components like `appengineApi` or `dataUpdaterPlant` already had a flatter structure and require minimal or no changes.
