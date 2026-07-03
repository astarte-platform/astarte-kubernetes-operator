# Setting up the Cluster

Once the Astarte Operator [has been installed](030-installation_kubernetes.html), and any prerequisite
[has been fulfilled](020-prerequisites.html), you can move forward and deploy an Astarte Cluster.

## Using a standard Astarte CR

The standard way of deploying an Astarte instance is by creating your own Astarte Custom Resource.
This gives you an high degree of customization, allowing you to tweak any single parameter in the
Astarte setup.

The main Astarte CRD is extensively documented and the available fields can be inspected
[here](https://docs.astarte-platform.org/astarte-kubernetes-operator/26.7/crds/index.html).

To create your Astarte resource, just create your Astarte Custom Resource, which will look something
like this:

```yaml
apiVersion: api.astarte-platform.org/v2alpha1
kind: Astarte
metadata:
  name: astarte
  namespace: astarte
spec:
  version: 1.3.0
  api:
    host: api.astarte.yourdomain.com
  cassandra:
    astarteSystemKeyspace:
      replicationFactor: 1
      replicationStrategy: SimpleStrategy
    connection:
      nodes:
        - host: "cassandra.example.com"
          port: 9042
      credentialsSecret:
        name: cassandra-connection-secret
        usernameKey: username
        passwordKey: password
  vernemq:
    deploy: true
    replicas: 1
    host: broker.astarte.yourdomain.com
    port: 1883
    sslListener: true
    sslListenerCertSecretName: astarte-tls-cert
  rabbitmq:
    connection:
      host: "rabbitmq.example.com"
      port: 5672
      credentialsSecret:
        name: rabbitmq-connection-secret
        usernameKey: username
        passwordKey: password
    managementConnection:
      host: "rabbitmq.example.com"
      port: 5672
```

Starting from Astarte v1.0.1, traffic coming to the broker is TLS terminated ad VerneMQ level. The
two fields controlling this features, namely `sslListener` and `sslListenerCertSecretName` can be
found within the `vernemq` section of the Astarte CR. In a nutshell, their meaning is:
- `sslListener` controls whether TLS termination is enabled at VerneMQ level or not,
- `sslListenerCertSecretName` is the name of TLS secret used for TLS termination (more on how to
  deal with Astarte certificates [here](050-handling_certificates.html)). When `sslListener` is
  true, the secret name **must** be set.

You can simply apply this resource in your Kubernetes cluster with `kubectl apply -f
<astarte-cr.yaml>`. The Operator will take over from there.
