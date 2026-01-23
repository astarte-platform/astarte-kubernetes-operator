# ADI (Astarte Default Ingress) CR Changes from v1alpha1 to v2alpha1

# Metrics
Starting with ADI v2alpha1, metrics are no longer exposed via the Astarte Default Ingress. If metrics are required to be exposed, a custom ingress must be provided. Prometheus ServiceMonitors remain the recommended collection method.

### Metrics Fields (`spec.api`)

| **Field**        | **v1alpha1**           | **v2alpha1** | **Description**                                                  |
|------------------|------------------------|--------------|------------------------------------------------------------------|
| `ServeMetrics`   | **Present** (`*bool`)  | **Removed**  | The option to serve metrics on a dedicated ingress has been removed. |
| `ServeMetricsToSubnet` | **Present** (`string`) | **Removed**  | The option to restrict metrics access to a specific subnet has been removed. |


# Ingress Controller Annotation
ADI v2alpha1 is the last version supporting NGINX as Ingress Controller due to [Ingress NGINX retirement.](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/) It is highly recommended to migrate to HAProxy Ingress Controller. To allow users to still use NGINX, the annotation `ingress.astarte-platform.org/ingress-controller-selector` has been introduced in ADI v1alpha1. Depending on the value of this annotation, the Operator will create Ingress resources compatible with the selected Ingress Controller.

By default, the Operator assumes the HAPROXY Ingress Controller is in use, in fact the annotation is set by default to:
```yaml
ingress.astarte-platform.org/ingress-controller-selector: "haproxy.org"
```

Deployments migrating from ADI v1alpha1 to v2alpha1 that are still using NGINX as Ingress Controller must set the annotation to:
```yaml
ingress.astarte-platform.org/ingress-controller-selector: "nginx.ingress.kubernetes.io"
``` 
