#!/bin/bash

ASTARTE_OP_RELEASE_NAME="astarte-operator" # default name
ASTARTE_OP_RELEASE_NAMESPACE="kube-system" # default namespace

function usage() {
    echo "usage: $1 -d <dirname> [-n <release-name>] [-N <release-namespace>];"
    echo "    release-name defaults to astarte-operator"
    echo "    release-namespace defaults to kube-system"
    exit 1;
}

function update_resources() {
    echo $ASTARTE_OP_TEMPLATE_DIR
    cd $ASTARTE_OP_TEMPLATE_DIR/astarte-operator/templates

    # crds.yaml contains all the CRDs defined by Astarte Operator in a single file.
    # Split each Custom Resource Definition and redirect it to a dedicated file whose name
    # is formatted as crd_n.yaml.
    gawk '/^# Source:/{n++}{print > "crd_" n ".yaml"}' crds.yaml

    # Check which CRD is defined within each crd_n.yaml file and rename the file accordingly
    # e.g.: if crd_1.yaml implements the Astarte CRD, rename it as crd_astarte.yaml
    mv $(grep 'singular: astarte$' ./crd_* | gawk '{print $1;}' | cut -f 1 -d ":") crd_astarte.yaml
    mv $(grep 'singular: astartevoyageringress$' ./crd_* | gawk '{print $1;}' | cut -f 1 -d ":") crd_astartevoyageringress.yaml
    mv $(grep 'singular: flow$' ./crd_* | gawk '{print $1;}' | cut -f 1 -d ":") crd_flow.yaml

    kubectl replace -f crd_astarte.yaml
    kubectl replace -f crd_astartevoyageringress.yaml
    kubectl apply -f crd_flow.yaml

    kubectl apply -f rbac.yaml
    kubectl apply -f webhook.yaml
}

function annotate_resources() {
    kubectl annotate crd astartes.api.astarte-platform.org meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate crd astartes.api.astarte-platform.org meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    crd astartes.api.astarte-platform.org app.kubernetes.io/managed-by=Helm

    kubectl annotate crd astartevoyageringresses.api.astarte-platform.org meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate crd astartevoyageringresses.api.astarte-platform.org meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    crd astartevoyageringresses.api.astarte-platform.org app.kubernetes.io/managed-by=Helm

    kubectl annotate crd flows.api.astarte-platform.org meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate crd flows.api.astarte-platform.org meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    crd flows.api.astarte-platform.org app.kubernetes.io/managed-by=Helm

    kubectl annotate clusterroles.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-manager-role meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate clusterroles.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-manager-role meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    clusterroles.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-manager-role app.kubernetes.io/managed-by=Helm

    kubectl annotate clusterroles.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-metrics-reader meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate clusterroles.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-metrics-reader meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    clusterroles.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-metrics-reader app.kubernetes.io/managed-by=Helm

    kubectl annotate clusterroles.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-proxy-role meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate clusterroles.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-proxy-role meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    clusterroles.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-proxy-role app.kubernetes.io/managed-by=Helm

    kubectl annotate roles.rbac.authorization.k8s.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-leader-election-role meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate roles.rbac.authorization.k8s.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-leader-election-role meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    roles.rbac.authorization.k8s.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-leader-election-role app.kubernetes.io/managed-by=Helm

    kubectl annotate rolebindings.rbac.authorization.k8s.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-leader-election-rolebinding meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate rolebindings.rbac.authorization.k8s.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-leader-election-rolebinding meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    rolebindings.rbac.authorization.k8s.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-leader-election-rolebinding app.kubernetes.io/managed-by=Helm

    kubectl annotate clusterrolebindings.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-manager-rolebinding meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate clusterrolebindings.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-manager-rolebinding meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    clusterrolebindings.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-manager-rolebinding app.kubernetes.io/managed-by=Helm

    kubectl annotate clusterrolebindings.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-proxy-rolebinding meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate clusterrolebindings.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-proxy-rolebinding meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    clusterrolebindings.rbac.authorization.k8s.io $ASTARTE_OP_RELEASE_NAME-proxy-rolebinding app.kubernetes.io/managed-by=Helm

    kubectl annotate service -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-webhook-service meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate service -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-webhook-service meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    service -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-webhook-service app.kubernetes.io/managed-by=Helm

    kubectl annotate certificates.cert-manager.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-manager-webhook meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate certificates.cert-manager.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-manager-webhook meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    certificates.cert-manager.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-manager-webhook app.kubernetes.io/managed-by=Helm

    kubectl annotate issuers.cert-manager.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-selfsigned-issuer meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate issuers.cert-manager.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-selfsigned-issuer meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    issuers.cert-manager.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-selfsigned-issuer app.kubernetes.io/managed-by=Helm

    kubectl annotate service -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-controller-manager-metrics-service meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate service -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-controller-manager-metrics-service meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    service -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-controller-manager-metrics-service app.kubernetes.io/managed-by=Helm

    kubectl annotate mutatingwebhookconfigurations.admissionregistration.k8s.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-mutating-webhook-configuration meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate mutatingwebhookconfigurations.admissionregistration.k8s.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-mutating-webhook-configuration meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    mutatingwebhookconfigurations.admissionregistration.k8s.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-mutating-webhook-configuration app.kubernetes.io/managed-by=Helm

    kubectl annotate validatingwebhookconfigurations.admissionregistration.k8s.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-validating-webhook-configuration meta.helm.sh/release-name=$ASTARTE_OP_RELEASE_NAME
    kubectl annotate validatingwebhookconfigurations.admissionregistration.k8s.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-validating-webhook-configuration meta.helm.sh/release-namespace=$ASTARTE_OP_RELEASE_NAMESPACE
    kubectl label    validatingwebhookconfigurations.admissionregistration.k8s.io -n $ASTARTE_OP_RELEASE_NAMESPACE $ASTARTE_OP_RELEASE_NAME-validating-webhook-configuration app.kubernetes.io/managed-by=Helm
}

while getopts "hd:n:N:" opt; do
    case ${opt} in
        d) ASTARTE_OP_TEMPLATE_DIR=$OPTARG ;;
        n) ASTARTE_OP_RELEASE_NAME=$OPTARG ;;
        N) ASTARTE_OP_RELEASE_NAMESPACE=$OPTARG ;;
        ?) usage
    esac
done


if [[ -z $ASTARTE_OP_TEMPLATE_DIR ]]
then
    echo "The directory must be specified"
    usage
    exit 1
fi

echo "The directory where the helm template output was saved is: $ASTARTE_OP_TEMPLATE_DIR"
echo "The chosen operator name is:                               $ASTARTE_OP_RELEASE_NAME"
echo "The operator will be installed in namespace:               $ASTARTE_OP_RELEASE_NAMESPACE"
echo
echo "Please check your choices and be aware that a malformed configuration may lead to unwanted outcomes."
read -p "Continue (y/n)?" choice
case ${choice} in
    y|Y) echo "yes";;
    n|N) echo "no"
          exit -1 ;;
    *) echo "invalid"
        exit -1 ;;
esac

echo "Ok... let's go!"

echo
echo "[1/2] Start upgrading CRDs..."
update_resources
echo "[1/2] Done."
echo
echo "[2/2] Start annotating resources..."
annotate_resources
echo "[2/2] Done."
