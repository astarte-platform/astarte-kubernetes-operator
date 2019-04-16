# Astarte Kubernetes Operator

The Astarte Kubernetes operator lets you deploy Astarte on your favorite cloud vendor with ease.

## Getting started

This section describes all the steps you need to go from an empty Kubernetes cluster to a working Astarte deployment.

It is recommended that your cluster has at least 4 CPUs and 8 GiB of RAM, and you should be aware that resource requests can't be splitted across nodes (i.e. if Cassandra has a CPU request of 2000m, you should have at least one node with 2000m free CPU resources).

The guide assumes that you already have your cluster set up, that you connect to it with `kubectl` and that you own a domain (we'll use `example.com`). Follow your cloud provider instructions to achieve that.

Astarte uses Voyager as its Ingress, so the first step is to install it following [its instructions](https://appscode.com/products/voyager/).

Then navigate into the `deploy` directory

```
cd deploy
```

Install the service account and check that it was correctly installed

```
kubectl apply -f service_account.yaml
kubectl get ServiceAccount -n kube-system astarte-operator
```

Install the cluster role and check that it was correctly installed

```
kubectl apply -f role.yaml
kubectl get ClusterRole astarte-operator
```

Install the role binding and check that it was correctly installed

```
kubectl apply -f role_binding.yaml
kubectl get ClusterRoleBinding astarte-operator
```

Navigate into the `crds` directory

```
cd crds
```

Install Astarte Custom Resource Definitions and check they were correctly installed

```
kubectl apply -f api_v1alpha1_astarte_voyager_ingress_crd.yaml
kubectl apply -f api_v1alpha1_astarte_crd.yaml
kubectl get CustomResourceDefinition
```

Go back to the previous directory

```
cd ..
```

Install the operator and wait until it is ready

```
kubectl apply -f operator.yaml
kubectl get deployment -n kube-system astarte-operator
```

Create the `astarte` namespace

```
kubectl create namespace astarte
```

At this point, you can create your Astarte object. You can start from one of the configurations in the `examples` folder.

`api_v1alpha1_astarte_cr_minimal.yaml` contains the bare minimum to get you started, the only values that require tweaking are the hostnames of APIs and VerneMQ and the resource requests/limits.

To choose the correct value for the resource requests, you should check the current resource utilization in your cluster with

```
kubectl describe nodes
```

Near the end of each node output, there will be a table with the allocated resources:

```
  Resource   Requests    Limits
  --------   --------    ------
  cpu        702m (17%)  102m (2%)
  memory     220Mi (2%)  440Mi (5%)
```

Now, if you take the total resources of your cluster and subtract the allocated requests, you will have the maximum number or resource requests that you can use.

In the example above, if the cluster has a single node with 4 CPUs and 8192 MiB of RAM, there are 4000m - 702m = 3298m free CPUs and 8192Mi - 220Mi = 7972MiB free memory. This means that the sum of all CPU requests in the Astarte object (RabbitMQ + Cassandra + VerneMQ + CFSSL + Astarte Components) must not exceed 3298m CPU and 7972 MiB of memory. Limits, unlike requests, can instead exceed the total of your cluster resources, since they're made to handle bursts in utilization.

As a rule of thumb, you should always leave some spare resources to avoid resource exhaustion leading to pods not being scheduled.

If you want to tweak your Astarte object further, you can check all available options in the `api_v1alpha1_astarte_cr.yaml` example.

After you finish customizing you Astarte object, you can deploy it with

```
kubectl apply -f api_v1alpha1_astarte_cr_minimal.yaml
```

You can check what's happening in the cluster with

```
kubectl get pods -n astarte --watch
```

When all the pods are marked as Running (except `cfssl-ca-secret-job`, that should be marked as Completed), your Astarte deployment is ready.

After that, you have to provide an Ingress to reach the APIs and the broker. To do that, we provide an AstarteVoyagerIngress object.

You can customize your Astarte Voyager Ingress object starting from `api_v1alpha1_astarte_voyager_ingress_cr_minimal.yaml` in the `examples` folder.

The values that require tweaking are `astarte` (that must match the name contained in the metadata of the Astarte object that you previously deployed), the host for the dashboard and the `letsencrypt` configuration. The latter depends on your cloud vendor, so you should check [Voyager docs](https://appscode.com/products/voyager/9.0.0/guides/certificate/dns/providers/) to see what credentials you need. Then follow [this paragraph](https://appscode.com/products/voyager/9.0.0/guides/certificate/dns/providers/#how-to-provide-dns-provider-credential) to create the secret and put the required Certificate object `spec` in the `letsencrypt` key of your Astarte Voyager Ingress object.

If you want more control on the Astarte Voyager Ingress object, you can check all the available options in the `api_v1alpha1_astarte_voyager_ingress_cr.yaml` example.

At the end, deploy the Astarte Voyager Ingress object with

```
kubectl apply -f api_v1alpha1_astarte_voyager_ingress_cr_minimal.yaml
```

The process will take some time to complete, you can check when the services are ready with

```
kubectl get svc --namespace astarte --watch
```

After the deployment is finished, you should be able to see the IP addresses of the API and broker Ingresses in the `EXTERNAL-IP` column. Now just point `api.example.com` and `dashboard.example.com` to the API Ingress IP and `broker.example.com` to the broker Ingress IP.

Your Astarte deployment is now ready to go and you just have to retrieve the generated Housekeeping API private key.

First list the secrets in the astarte namespace with

```
kubectl get secret -n astarte
```

Take note of the secret ending with `-housekeeping-private-key` and save the key to the `housekeeping.key` file with

```
kubectl get secret -n astarte <secret-name> -o=jsonpath={.data.private-key} | base64 -d > housekeeping.key
```

You can use `housekeeping.key` to create a realm and start using Astarte. You can follow the ["Astarte in 5 minutes"](https://docs.astarte-platform.org/latest/010-astarte_in_5_minutes.html#create-a-realm) guide from this point on, just remember to use the correct URLs:

- Housekeeping base URL is `https://api.example.com/housekeeping/v1` instead of `http://localhost:4001/v1`
- Realm Management base URL is `https://api.example.com/realmmanagement/v1` instead of `http://localhost:4000/v1`
- Pairing base URL is `https://api.example.com/pairing/v1` instead of `http://localhost:4003/v1`
- AppEngine base URL is `https://api.example.com/appengine/v1` instead of `http://localhost:4002/v1`

## Troubleshooting

If you have problems with the deployment, you can check the logs of the Operator with

```
kubectl logs -f -n kube-system <operator-pod-name> --tail=20
```

If a specific pod is having problem being deployed, check its status and its logs with

```
kubectl describe pods -n astarte <pod-name>
kubectl logs -f -n astarte <pod-name
```

Feel free to open an issue if you run into unexpected problems.
