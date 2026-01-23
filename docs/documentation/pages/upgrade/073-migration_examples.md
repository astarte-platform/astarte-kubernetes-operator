# Astarta CR Migration examples (v1alpha3 to v2alpha1)

## Scylla
The following is a minimal Cassandra configuration to connect to an external instance.

```yaml
# apiVersion: astarte.astarte-platform.org/v2alpha1
# ...
spec:
  # ...
  cassandra:
    connection:
      # Replace the connection details with those used by the target instance
      nodes:
        - host: "my-external-cassandra.db.svc.cluster.local"
          port: 9042
      # Create a secret in the cluster and reference it here
      credentialsSecret:
        name: astarte-cassandra-credentials
        usernameKey: username
        passwordKey: password
```

## RabbitMQ

The following is a minimal RabbitMQ configuration to connect to an external instance.

```yaml
# apiVersion: astarte.astarte-platform.org/v2alpha1
# ...
spec:
  # ...
  rabbitmq:
    connection:
  	    # Edit with the connection details in use
      host: "rabbitmq.rabbitmq.svc.cluster.local"
      port: 5672
      # Reference the secret containing login information
      credentialsSecret:
        name: "rabbitmq-connection-secret"
        usernameKey: "username"
        passwordKey: "password"
```

## Astarte Components

The following example shows how to merge the `api` and `backend` configurations for the `housekeeping` component into the new, unified structure.

### v1alpha3 (Old)

```yaml
# apiVersion: astarte.astarte-platform.org/v1alpha3
# ...
spec:
  # ...
  components:
    housekeeping:
      api:
        replicas: 1
        disableAuthentication: true
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
      backend:
        replicas: 2
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 400m
            memory: 512Mi
```

### v2alpha1 (New)

For the new version, the settings must be combined. Some fields like `replicas` and `resources`
existed in both old objects, so an appropriate, consolidated set of values must be selected for the
new, single service.

```yaml
# apiVersion: astarte.astarte-platform.org/v2alpha1
# ...
spec:
  # ...
  components:
    housekeeping:
      # Choose the appropriate replica count for the unified service.
      # In this example the backend's replica count is kept.
      replicas: 2
      # The 'disableAuthentication' field from the old 'api' object is moved here.
      disableAuthentication: true
      # Define a single, appropriate set of resources for the unified service.
      resources:
        requests:
          cpu: 200m
          memory: 256Mi
        limits:
          cpu: 400m
          memory: 512Mi
```
