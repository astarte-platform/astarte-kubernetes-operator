# Advanced operations

This section provides guidance on some delicate operations that must be performed manually as they
could potentially result in data loss or other types of irrecoverable damage.

*Always be careful while performing these operations!*

Advanced operations are described in the following sections:
- [How to backup your Astarte resources](#backup-your-astarte-resources)
- [How to restore your backed up Astarte instance](#restore-your-backed-up-astarte-instance)

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

and, assuming that the name of your Astarte is `astarte` and that it is deployed within the
`astarte` namespace, it can be done simply executing the following commands:
```bash
kubectl get astarte -n astarte -o yaml > astarte-backup.yaml
kubectl get adi -n astarte -o yaml > adi-backup.yaml
kubectl get secret astarte-devices-ca -n astarte -o yaml > astarte-devices-ca-backup.yaml
```

---

## Restore your backed up Astarte instance

To restore your Astarte instance simply apply the resources you saved as described
[here](#backup-your-astarte-resources). Please, be aware that the order of the operations matters.

```bash
kubectl apply -f astarte-devices-ca-backup.yaml
kubectl apply -f astarte-backup.yaml
```

And when your Astarte resource is ready, to restore your AstarteDefaultIngress resource:

```bash
kubectl apply -f adi-backup.yaml
```

At the end of this step, your cluster is restored. Please, notice that the external IP of the
ingress services might have changed. Take action to ensure that the changes of the IP are reflected
anywhere appropriate in your deployment.
