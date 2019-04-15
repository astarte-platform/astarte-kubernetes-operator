# Astarte Kubernetes Operator

The Astarte Kubernetes operator lets you deploy Astarte on your favorite cloud vendor with ease.

## Getting started

This section describes all the steps you need to go from an empty Kubernetes cluster to a working Astarte deployment. The guide assumes that you already have your cluster set up and `kubectl` pointing to the correct cluster. Follow your cloud provider instructions to achieve that.

First of all, navigate into the `deploy` directory

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

Install astarte CRDs and check they were correctly installed

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

At this point, you can create your Astarte CRD. You can start from one of the configurations in the `examples` folder.

`api_v1alpha1_astarte_cr_minimal.yaml` contains the bare minimum to get you started, the only values that require tweaking are the hostnames of APIs and VerneMQ and the resource requests/limits. To decide  the resource utilization of your nodes with `kubectl describe nodes`, make sure that free resources are enough to accomodate the sum of resource requests.

If you want to tweak your Astarte CRD further, you can check all available options in the `api_v1alpha1_astarte_cr.yaml` example.

After you finish customizing you Astarte CRD, you can deploy it with

```
kubectl apply -f api_v1alpha1_astarte_cr_minimal.yaml
```

You can check what's happening in the cluster with

```
kubectl get pods -n astarte --watch
```

When all the pods are marked as Running (except `cfssl-ca-secret-job`, that should be marked as Completed), your Astarte deployment is ready.

After that, you have to provide an Ingress to reach the APIs and the broker. To do that, we provide an `AstarteVoyagerIngress` CRD.

First of all, install Voyager following [its instructions](https://appscode.com/products/voyager/).

Then you can customize your Astarte Voyager Ingress CRD starting from `api_v1alpha1_astarte_voyager_ingress_cr_minimal.yaml` in the `examples` folder.

The values that require tweaking are `astarte` (that must match the name contained in the metadata of the Astarte CRD that you previously deployed), the host for the dashboard and the `letsencrypt` configuration. The latter depends on your cloud vendor, so you should check [Voyager docs](https://appscode.com/products/voyager/9.0.0/guides/certificate/dns/providers/) to see what credentials you need. Then follow [this paragraph](https://appscode.com/products/voyager/9.0.0/guides/certificate/dns/providers/#how-to-provide-dns-provider-credential) to create the secret and put the required Certificate CRD `spec` in the `letsencrypt` key of your Astarte Voyager Ingress CRD.

If you want more control on the Astarte Voyager Ingress CRD, you can check all the available options in the `api_v1alpha1_astarte_voyager_ingress_cr.yaml` example.

At the end, deploy the Astarte Voyager Ingress CRD with

```
kubectl apply -f api_v1alpha1_astarte_voyager_ingress_cr_minimal.yaml
```

After the CRD has finished deploying, you should be able to get the IP addresses of APIs and broker with

```
kubectl get svc --namespace astarte
```

and looking at the External-IP column. Now just follow your cloud vendor instructions to make the domains (`api.yourdomain.com`, `dashboard.yourdomain.com` and `broker.yourdomain.com`) point to the correct IP address and your Astarte deployment is ready to go.

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
