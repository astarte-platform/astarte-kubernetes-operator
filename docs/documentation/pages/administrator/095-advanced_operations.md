# Advanced operations

This section provides guidance on some delicate operations that must be performed manually as they
could potentially result in data loss or other types of irrecoverable damage.

*Always be careful while performing these operations!*

Advanced operations are described in the following sections:
- [Advanced operations](#advanced-operations)
  - [Backup your Astarte resources](#backup-your-astarte-resources)
  - [Restore your backed up Astarte instance](#restore-your-backed-up-astarte-instance)
  - [Assign an `astarteInstanceID` to an existing Astarte instance](#assign-an-astarteinstanceid-to-an-existing-astarte-instance)

---

## Backup your Astarte resources

Backing up your Astarte resources is crucial in all those cases when your Astarte instance has to be
restored after an unforeseen event (e.g. accidental deletion of resources, deletion of the
Operator - as it will be discussed later on - etc.).

A full recovery of your Astarte instance along with all the persisted data is possible **if and only
if** your Cassandra/Scylla instance is deployed independently from Astarte, i.e. it must be deployed
outside of the Astarte CR scope. Provided that this condition is met, all the data persist in the
database even when Astarte is deleted from your cluster.

To restore your Astarte instance all you have to do is saving the following resources:
+ Astarte CR;
+ AstarteDefaultIngress CR (if deployed);
+ CA certificate and key;
+ Housekeeping private and public keys;

and, assuming that the name of your Astarte is `astarte` and that it is deployed within the
`astarte` namespace, it can be done simply executing the following commands:
```bash
kubectl get astarte -n astarte -o yaml > astarte-backup.yaml
kubectl get adi -n astarte -o yaml > adi-backup.yaml
kubectl get secret astarte-devices-ca -n astarte -o yaml > astarte-devices-ca-backup.yaml
kubectl get secret astarte-housekeeping-public-key -n astarte -o yaml > astarte-housekeeping-public-key-backup.yaml
kubectl get secret astarte-housekeeping-private-key -n astarte -o yaml > astarte-housekeeping-private-key-backup.yaml
```

---

## Restore your backed up Astarte instance

To restore your Astarte instance simply apply the resources you saved as described
[here](#backup-your-astarte-resources). Please, be aware that the order of the operations matters.

```bash
kubectl apply -f astarte-devices-ca-backup.yaml
kubectl apply -f astarte-housekeeping-public-key-backup.yaml
kubectl apply -f astarte-housekeeping-private-key-backup.yaml
kubectl apply -f astarte-backup.yaml
```

And when your Astarte resource is ready, to restore your AstarteDefaultIngress resource:

```bash
kubectl apply -f adi-backup.yaml
```

At the end of this step, your cluster is restored. Please, notice that the external IP of the
ingress services might have changed. Take action to ensure that the changes of the IP are reflected
anywhere appropriate in your deployment.

---

## Assign an `astarteInstanceID` to an existing Astarte instance
Starting from Astarte Operator v24.5 and Astarte 1.2.x, it is possible to assign an `astarteInstanceID` trough the field `spec.astarteInstanceID` in the Astarte CR. This ID is used to uniquely identify an Astarte instance, especially in scenarios where multiple instances are deployed within the same Kubernetes cluster and share the same backend services (e.g., Cassandra/Scylla, RabbitMQ).

- It is mandatory to assign an `astarteInstanceID` when the same Scylla instance is used by multiple Astarte deployments.
- The `astarteInstanceID` must be unique across all Astarte instances sharing the same backend services.

> [!CAUTION]
> The procedure described in this section involves downtime for your Astarte instance and should only be performed if your Scylla instance is shared among multiple Astarte deployments.

Steps to assign an `astarteInstanceID` to an existing Astarte deployment:

**Step 1**

Choose a unique `astarteInstanceID` for your Astarte instance. This can be any string that uniquely identifies your instance. Since each Astarte Realm is associated with a Scylla keyspace, you will need to define a new keyspace name for each realm using this `astarteInstanceID`.

- The keyspace names must follow the pattern: `<astarteInstanceID><realm_name>`.
- Dashes (`-`) and underscores (`_`) characters are not allowed.
- The maximum length for keyspace names is 48 characters.
- Ensure that there are no keyspace name collisions with existing keyspaces in your Scylla instance.

**Step 2**

Backup your Astarte resources as described in the [Backup your Astarte resources](#backup-your-astarte-resources) section. This includes backing up the Astarte CR, devices CA secrets and Housekeeping keys.

**Step 3**

Delete your existing Astarte CR from the cluster. The Astarte instance will be deleted from the cluster.

**Step 4**

For each realm associated with your Astarte instance, handle the corresponding Scylla keyspaces to include the new `astarteInstanceID` prefix. Scylla does not support renaming keyspaces, so, for each Realm, you will need to:
- Create a new keyspace with the new name following the pattern `<astarteInstanceID><realm_name>` and the same replication settings as the old keyspace.
- Migrate the data from the old keyspace to the new keyspace.
- Drop the old keyspace once the data migration is complete.

**Step 5**

Edit the Astarte CR to include the `astarteInstanceID` field under the `spec` section with the value you chose in step 1.

**Step 6**

Restore the devices CA and Housekeeping keys by re-importing the secrets backed up in step 2 by following the instructions in the [Restore your backed up Astarte instance](#restore-your-backed-up-astarte-instance) section. Then, reapply the modified Astarte CR to the cluster. This will redeploy the Astarte instance with the assigned `astarteInstanceID`.
