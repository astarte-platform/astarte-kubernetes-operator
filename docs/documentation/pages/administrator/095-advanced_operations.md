# Advanced operations

This section provides guidance on some delicate operations that must be performed manually as they
could potentially result in data loss or other types of irrecoverable damage.

*Always be careful while performing these operations!*

Advanced operations are described in the following sections:
- [How to backup your Astarte resources](#backup-your-astarte-resources)
- [How to restore your backed up Astarte instance](#restore-your-backed-up-astarte-instance)
- [Handling Astarte when uninstalling the
  Operator](#handling-astarte-when-uninstalling-the-operator)

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
+ AstarteVoyagerIngress CR (if deployed);
+ AstarteDefaultIngress CR (if deployed);
+ CA certificate and key;

and, assuming that the name of your Astarte is `astarte` and that it is deployed within the
`astarte` namespace, it can be done simply executing the following commands:
```bash
kubectl get astarte -n astarte -o yaml > astarte-backup.yaml
kubectl get avi -n astarte -o yaml > avi-backup.yaml
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

And when your Astarte resource is ready, to restore your AstarteVoyagerIngress:

```bash
kubectl apply -f avi-backup.yaml
```

while to restore your AstarteDefaultIngress resource:

```bash
kubectl apply -f adi-backup.yaml
```

At the end of this step, your cluster is restored. Please, notice that the external IP of the
ingress services might have changed. Take action to ensure that the changes of the IP are reflected
anywhere appropriate in your deployment.

---

## Handling Astarte when uninstalling the Operator

Installing the Astarte Operator is as simple as installing its Helm chart. Even if the
install and upgrade procedures are very simple and straightforward, the design choices behind the
development of the Operator must be taken into account to avoid undesired effects while handling the
Operator's lifecycle.

The installation of the Operator's Helm chart is responsible for the creation of RBACs, the creation
of the Operator's deployment and the installation of Astarte CRDs. The fact that all the CRDs
installed with the Helm chart are templated has some important consequences: if on one hand this
characteristic ensures great flexibility in configuring your Astarte instance, on the other hand it
entails the possibility of deleting the CRDs by simply uninstalling the Operator.

The following sections will highlight what happens under the hood while uninstalling the Operator
and show the suggested path to restore your Astarte instance after the removal of the Operator.

Please, read carefully the following sections before taking any actions on your cluster and be aware
that improper operations may have catastrophic effects on your Astarte instance.

### What happens when uninstalling the Operator

The Operator's installation procedure marks all the Astarte CRDs as owned by the Operator itself.
Therefore, when the Operator is uninstalled all the CRDs are seen as orphaned and the Kubernetes
controller automatically sets them as ready to be deleted. Thus, when the Operator is uninstalled
you end up with the following situation:
- Flow and AstarteVoyagerIngress CRDs are deleted, along with the custom resources depending on
  said CRDs;
- Astarte CRD is marked for deletion, but its removal is postponed until the moment in which the
  Astarte finalizer is executed.

### Backup your resources

Even if removing the Operator can potentially destroy your Astarte instance, there is a way to
restore it avoiding any data loss. Please, refer to [this dedicated
section](#backup-your-astarte-resources) to understand how to backup your resources.

### Uninstall the Operator

Once the backup of your resources is completed you can `helm uninstall` the Operator as explained
[here](030-installation_kubernetes.html#uninstalling-the-operator).

Once the Operator is deleted your Astarte instance will be marked for deletion. You can see it
simply checking the `Deletion timestamp` field in the output of:
```bash
kubectl describe astarte -n astarte
```

### Reinstalling the Operator

Reinstalling the Operator is crucial to have a correct management of your Astarte instance. The
installation is handled simply with an `helm install` command as explained
[here](030-installation_kubernetes.html#installation).

When the first reconciliation loop is executed, the Operator becomes aware that the Astarte resource
is marked for deletion, so it executes the Astarte finalizer and eventually destroys Astarte's CRD
and its resources.

Even if it might look like the status of the cluster is compromised, a simple command reestablishes
order:
```bash
helm upgrade --install astarte-operator astarte/astarte-operator -n kube-system
```
This command simply upgrades the Operator and, as a result, installs the missing CRDs. Now it is
time to restore the Astarte resources.

### Apply backed up resources

To restore your Astarte instance simply follow the instructions outlined
[here](#restore-your-backed-up-astarte-instance).

### Conclusion

The procedure presented in the current section allows to handle the deletion of the Operator from
your cluster without losing any of Astarte's data. Currently some manual intervention is required to
ensure that the integrity of your instance is not compromised by the uninstall procedure.
