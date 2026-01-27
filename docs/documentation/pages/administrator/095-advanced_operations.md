# Advanced operations

This section provides guidance on some delicate operations that must be performed manually as they
could potentially result in data loss or other types of irrecoverable damage.

*Always be careful while performing these operations!*

Advanced operations are described in the following sections:
- [Advanced operations](#advanced-operations)
  - [Backup your Astarte resources](#backup-your-astarte-resources)
  - [Restore your backed up Astarte instance](#restore-your-backed-up-astarte-instance)
  - [Set up an instance id](#set-up-an-instance-id)

---

## Backup your Astarte resources

Backing up your Astarte resources is crucial in all those cases when your Astarte instance has to be
restored after an unforeseen event (e.g. accidental deletion of resources, deletion of the
Operator - as it will be discussed later on - etc.).

A full recovery of your Astarte instance along with all the persisted data is possible **if and only
if** your Cassandra/Scylla instance is deployed independently from Astarte, i.e. it must be deployed
outside of the Astarte CR scope. Provided that this condition is met, all the data persist in the
database even when Astarte is deleted from your cluster (for example when using an external
ScyllaDB service).

To restore your Astarte instance all you have to do is saving the following resources:
+ Astarte CR;
+ AstarteDefaultIngress CR (if deployed);
+ CA certificate and key secret;
+ Astarte Housekeeping key secrets;

When reconnecting to an external Cassandra/ScyllaDB that already holds your data, be sure to
preserve the same `AstarteInstanceID` configuration used before the deletion. Changing the instance
ID would make Astarte look for different keyspaces, preventing a smooth restoration.

and, assuming that the name of your Astarte is `astarte` and that it is deployed within the
`astarte` namespace, it can be done simply executing the following commands:
```bash
kubectl get astarte -n astarte -o yaml > astarte-backup.yaml
kubectl get adi -n astarte -o yaml > adi-backup.yaml
kubectl get secret astarte-devices-ca -n astarte -o yaml > astarte-devices-ca-backup.yaml
kubectl get secret astarte-housekeeping-public-key -n astarte -o yaml > astarte-housekeeping-public-key-backup.yaml
kubectl get secret astarte-housekeeping-private-key -n astarte -o yaml > astarte-housekeeping-private-key-backup.yaml

```

> The operator checks for the public key secret `<astarte-name>-housekeeping-public-key`. If it
> exists, both keys are preserved and nothing is regenerated. If only the private key exists, the
> operator deletes it and generates a new key pair. Always back up both housekeeping secrets to keep
> existing keys.

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

## Set up an instance id

`AstarteInstanceID` is the unique identifier associated with an Astarte instance.  
This parameter is optional and defaults to an empty string (`""`) for backward
compatibility with existing installations.  

When set, `AstarteInstanceID` allows multiple Astarte instances to share the same
database infrastructure by isolating their keyspaces. The identifier is used as a
prefix in Cassandra keyspace names.
The Operator validates AstarteInstanceID against the following regular expression:
`^[a-z]?[a-z0-9]{0,47}$` (must start with a lowercase letter, followed by up to 
47 lowercase letters or digits). If the value does not match this pattern, the 
validation webhook will reject the resource and Astarte will not start.

To enable this feature in a Kubernetes environment, the value must be provided in
the Astarte resource specification (`Spec.AstarteInstanceID`). At runtime, the
value is propagated through the environment variables:

- `ASTARTE_INSTANCE_ID`
- `DOCKER_VERNEMQ_ASTARTE_VMQ_PLUGIN__ASTARTE_INSTANCE_ID`

Note that once an `AstarteInstanceID` is configured, it cannot be changed.

Always reuse the same `AstarteInstanceID` when reinstalling or upgrading the Operator
to keep using the existing keyspaces and avoid data loss.
