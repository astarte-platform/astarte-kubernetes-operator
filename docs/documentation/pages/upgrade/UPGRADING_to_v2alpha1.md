# Upgrading from v1alpha2/v1alpha3 to v2alpha1

This guide details how to perform an **in-place upgrade** of an Astarte instance from `v1alpha2` or `v1alpha3` to `v2alpha1`.

Following this procedure allows you to upgrade the Astarte Operator and your Astarte instance without deleting the existing `Astarte` Custom Resource (CR). This approach is crucial for preserving data, state, and broker connections during the upgrade.

The process involves three main phases:
1.  **Prepare and Upgrade the CRD:** The Astarte Custom Resource Definition (CRD) is updated to support both old and new API versions simultaneously, facilitating a smooth transition.
2.  **Upgrade the Operator:** The Astarte Operator is upgraded to the new version using Helm.
3.  **Migrate the Astarte CR:** The Astarte CR is manually converted to the `v2alpha1` format and reapplied to the cluster, finalizing the upgrade.

## Prerequisites

Before you begin, ensure you have the following:

-   A running Astarte cluster using `v1alpha2` or `v1alpha3` of the Astarte CR.
-   `kubectl` access to the Kubernetes cluster where Astarte is running.
-   `helm` v3 installed and configured.
-   `yq` (version 4 or later) installed for advanced YAML manipulation.

### Important: Back Up Your Astarte CR

Before making any changes, create a backup of your current Astarte CR.

```bash
kubectl get astarte <your-astarte-name> -n <your-namespace> -o yaml > astarte-backup-v1alpha.yaml
```

## Step 1: Prepare and Upgrade the CRD

The standard `helm upgrade` process fails because the new operator's CRD is not compatible with the `status.storedVersions` of the existing CRD in the cluster. To solve this, we will create a temporary, merged CRD that supports both old and new API versions.

#### 1.1 Render the New CRD from the Helm Chart

First, render the `v2alpha1` CRD from the target Astarte Operator Helm chart into a local file.

```bash
helm template astarte-crd <chart-repo>/astarte-operator --version <new-version> \
  --show-only templates/crds/astartes.api.astarte-platform.org.yaml > astarte-crd-new.yaml
```
*Replace `<chart-repo>` and `<new-version>` with the appropriate values.*

#### 1.2 Get the Current CRD from the Cluster

Next, fetch the CRD that is currently active in your cluster.

```bash
kubectl get crd astartes.api.astarte-platform.org -o yaml > astarte-crd-old.yaml
```

#### 1.3 Merge API Versions

Manually edit `astarte-crd-new.yaml` to include the older API versions from `astarte-crd-old.yaml`. Open the file and locate the `spec.versions` section. Copy the entire `v1alpha2` and/or `v1alpha3` blocks from `astarte-crd-old.yaml` into this section.

Then, ensure that only `v2alpha1` is marked as the storage version. The final `versions` block should look like this:

```yaml
spec:
  # ... other fields
  versions:
  - name: v2alpha1
    # ... fields for v2alpha1
    storage: true
    served: true
  - name: v1alpha3
    # ... fields for v1alpha3
    storage: false
    served: true
  - name: v1alpha2
    # ... fields for v1alpha2
    storage: false
    served: true
```

#### 1.4 Optimize and Apply the Merged CRD

The merged CRD file can be too large for `etcd`, causing an error on apply. To fix this, strip the non-functional `description` fields using `yq`.

```bash
yq e 'del(.. | select(has("description")).description)' astarte-crd-new.yaml > astarte-crd-final.yaml
```

Now, apply the final, optimized CRD to the cluster. The `--server-side` and `--force-conflicts` flags are essential for correctly updating the CRD managed by Helm.

```bash
kubectl apply -f astarte-crd-final.yaml --server-side --force-conflicts
```
**Expected Output:**
```
customresourcedefinition.apiextensions.k8s.io/astartes.api.astarte-platform.org serverside-applied
```

## Step 2: Upgrade the Astarte Operator

With the CRD correctly updated, you can proceed with the Helm upgrade.

#### 2.1 (Optional) Update the Helm Chart
To prevent Helm from trying to apply its own (non-merged) CRD, you can replace the CRD file in your local chart directory with your final version.

```bash
cp astarte-crd-final.yaml <path-to-your-chart>/templates/crds/astartes.api.astarte-platform.org.yaml
```

#### 2.2 Execute the Helm Upgrade

Run the `helm upgrade` command, pointing to the chart of the new operator version.

```bash
helm upgrade <your-release-name> <chart-repo>/astarte-operator --version <new-version> -n <your-namespace>
```

After this step, the new operator will be running, and it will automatically start managing your existing Astarte instance. The Kubernetes API will now serve the Astarte resource as `v2alpha1`.

## Step 3: Migrate the Astarte CR to a Clean v2alpha1 State

After the operator upgrade, your Astarte instance is running, but its underlying configuration (`last-applied-configuration`) is still based on the old `v1alpha` schema. Attempting to modify the CR will lead to a validation error due to a limitation in the operator's webhook logic for this specific upgrade path.

To work around this, we will temporarily disable the validation webhook, apply the new CR, and then immediately re-enable it.

#### 3.1 Migrate Credentials to Secrets

First, ensure your database and broker credentials are in Kubernetes Secrets, as required by the `v2alpha1` API. Replace placeholder values with your actual credentials.

```bash
# For Cassandra/ScyllaDB
kubectl create secret generic scylladb-connection-secret --namespace <your-namespace> \
  --from-literal=username=cassandra \
  --from-literal=password=cassandra

# For RabbitMQ
kubectl create secret generic rabbitmq-connection-secret --namespace <your-namespace> \
  --from-literal=username=guest \
  --from-literal=password=guest
```

#### 3.2 Find and Back Up the Webhook Configuration

The validation webhook is a cluster-level resource. Its name depends on your Helm release name.

First, find your Helm release name:
```bash
helm list -n <your-namespace>
# Note the name from the NAME column
```

Next, construct the webhook name (e.g., `my-astarte-operator-validating-webhook-configuration`) and back it up to a file. **This is a critical safety step.**

```bash
# Replace <release-name> with your Helm release name
export WEBHOOK_NAME=<release-name>-validating-webhook-configuration
kubectl get validatingwebhookconfiguration $WEBHOOK_NAME -o yaml > webhook-backup.yaml
```

#### 3.3 Temporarily Disable the Astarte Validation

Patch the webhook configuration to remove the validation rule for Astarte resources. This is a targeted change that leaves other validations intact.

```bash
kubectl patch validatingwebhookconfiguration $WEBHOOK_NAME --type='json' \
  -p='[{"op": "remove", "path": "/webhooks/0"}]'
```
*This command assumes the Astarte webhook is the first in the list, which is the default.*

#### 3.4 Craft and Apply the Final v2alpha1 Manifest

Now, create a new YAML file, `astarte-final-v2alpha1.yaml`, to define your Astarte instance in the `v2alpha1` format.

-   Base the manifest on your backed-up `v1alpha` CR.
-   Update `apiVersion` to `api.astarte-platform.org/v2alpha1`.
-   Restructure the `spec` for `v2alpha1`, pointing to the new credential secrets.
-   **Crucially, explicitly define `spec.cassandra.astarteSystemKeyspace`**. Set its values to match the defaults Astarte was using previously.

Here is a complete example:
```yaml
apiVersion: api.astarte-platform.org/v2alpha1
kind: Astarte
metadata:
  name: astarte # Use your instance name
  namespace: astarte # Use your namespace
spec:
  version: "1.3.0" # Specify the new Astarte version
  api:
    host: api.astarte.192.168.49.100.sslip.io # Your API host

  # Restructure Cassandra configuration
  cassandra:
    connection:
      nodes:
        - host: "192.168.49.3"
          port: 9042
      credentialsSecret:
        name: scylladb-connection-secret
        usernameKey: username
        passwordKey: password
    # Explicitly set the keyspace spec to prevent immutability errors
    astarteSystemKeyspace:
      replicationStrategy: "SimpleStrategy"
      replicationFactor: 1
  
  # Restructure RabbitMQ configuration
  rabbitmq:
    connection:
      host: 192.168.49.4
      port: 5672
      credentialsSecret:
        name: rabbitmq-connection-secret
        usernameKey: username
        passwordKey: password
        
  # VerneMQ spec remains largely the same
  vernemq:
    host: broker.astarte.192.168.49.101.sslip.io
```

With the webhook disabled, apply this manifest. It should now succeed.
```bash
kubectl apply -f astarte-final-v2alpha1.yaml
```

#### 3.5 Re-enable the Validation Webhook

**Immediately after applying the Astarte CR**, restore the original webhook configuration from your backup. This is essential for maintaining cluster security and stability.

```bash
kubectl apply -f webhook-backup.yaml
```

Your Astarte instance is now fully upgraded to `v2alpha1` and can be managed and modified as usual.
