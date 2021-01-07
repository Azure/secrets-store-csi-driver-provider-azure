---
type: docs
title: "Installation"
linkTitle: "Installation"
weight: 1
description: >
  How to install Secrets Store CSI Driver and Azure Key Vault Provider on your clusters.
---

### Install the Secrets Store CSI Driver and the Azure Keyvault Provider

#### Prerequisites

Recommended Kubernetes version:
- For Linux - v1.16.0+
- For Windows - v1.18.0+

> For Kubernetes version 1.15 and below, please use [Azure Keyvault Flexvolume](https://github.com/Azure/kubernetes-keyvault-flexvol)

#### Deployment using Helm

Azure Key Vault Provider for Secrets Store CSI Driver allows users to customize their installation via Helm.

> Recommended to use Helm3

```bash
helm repo add csi-secrets-store-provider-azure https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/charts
helm install csi csi-secrets-store-provider-azure/csi-secrets-store-provider-azure
```

The helm charts hosted in [Azure/secrets-store-csi-driver-provider-azure](https://github.com/Azure/secrets-store-csi-driver-provider-azure/tree/master/charts/csi-secrets-store-provider-azure) repo include the Secrets Store CSI Driver helm charts as a dependency. Running the above `helm install` command will install both the Secrets Store CSI Driver and Azure Key Vault provider.

##### Values

For a list of customizable values that can be injected when invoking helm install, please see the [Helm chart configurations](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/charts/csi-secrets-store-provider-azure/README.md#configuration).

#### Using Deployment yamls

1. **Install the Secrets Store CSI Driver**

    💡 Follow the [Installation guide for the Secrets Store CSI Driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver#usage) to install the driver.

    > **NOTE:** `0.0.9+` release of the Azure Key Vault provider is incompatible with the Secrets Store CSI Driver versions < `v0.0.14`. While installing the Secrets Store CSI Driver using yamls, add the following flag `--grpc-supported-providers=azure` to the [Linux](https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/master/deploy/secrets-store-csi-driver.yaml) and [Windows](https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/master/deploy/secrets-store-csi-driver-windows.yaml) *daemonset* manifests.
    > - The flag needs to be added to the `secrets-store` container args

    <details>
    <summary>Result</summary>

    ```
    csidriver.storage.k8s.io/secrets-store.csi.k8s.io created
    serviceaccount/secrets-store-csi-driver created
    clusterrole.rbac.authorization.k8s.io/secretproviderclasses-role created
    clusterrolebinding.rbac.authorization.k8s.io/secretproviderclasses-rolebinding created
    clusterrole.rbac.authorization.k8s.io/secretprovidersyncing-role created
    clusterrolebinding.rbac.authorization.k8s.io/secretprovidersyncing-rolebinding created
    daemonset.apps/csi-secrets-store-windows created
    daemonset.apps/csi-secrets-store created
    customresourcedefinition.apiextensions.k8s.io/secretproviderclasses.secrets-store.csi.x-k8s.io created
    customresourcedefinition.apiextensions.k8s.io/secretproviderclasspodstatuses.secrets-store.csi.x-k8s.io created
    ```

    </details><br/>

    To validate the driver is running as expected, run the following command:

    ```bash
    kubectl get pods -l app=csi-secrets-store -n kube-system
    ```

    You should see the driver pods running on each agent node:

    ```bash
    NAME                      READY   STATUS    RESTARTS   AGE
    csi-secrets-store-bp4f4   3/3     Running   0          24s
    ```

    To validate the `--grpc-supported-providers=azure` arg has been configured correctly, run the following command:

    ```bash
    kubectl get ds -l app=csi-secrets-store -o jsonpath='{range .items[*]}{.spec.template.spec.containers[1].args}{"\n"}'
    ```

    You should see the args for the `secrets-store` container in the driver pods for each node:
    ```bash
    ["--debug=true","--endpoint=$(CSI_ENDPOINT)","--nodeid=$(KUBE_NODE_NAME)","--provider-volume=/etc/kubernetes/secrets-store-csi-providers","--grpc-supported-providers=azure","--metrics-addr=:8080"]
    ```

2. **Install the Azure Key Vault provider**

    For linux nodes
    ```bash
    kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/deployment/provider-azure-installer.yaml
    ```
    For windows nodes
    ```bash
    kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/deployment/provider-azure-installer-windows.yaml
    ```

    **NOTE**: Installing the provider using the deployment yamls from master will always install the latest version. If you want to deploy a specific version of the provider use the tagged release yamls.

    To validate the provider's installer is running as expected, run the following commands:

    ```bash
    kubectl get pods -l app=csi-secrets-store-provider-azure
    ```

    You should see the provider pods running on each agent node:

    ```bash
    NAME                                     READY   STATUS    RESTARTS   AGE
    csi-secrets-store-provider-azure-4ngf4   1/1     Running   0          8s
    csi-secrets-store-provider-azure-bxr5k   1/1     Running   0          8s
    ```

**In addition, if you are using Secrets Store CSI Driver and the Azure Keyvault Provider in a cluster with [pod security policy](https://kubernetes.io/docs/concepts/policy/pod-security-policy/) enabled**, review and create the following policy that enables the spec required for Secrets Store CSI Driver and the Azure Keyvault Provider to work:

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/deployment/pod-security-policy.yaml
```

### Uninstallation

#### Using Helm

If you deployed the Secrets Store CSI Driver and Azure Key Vault provider using the helm charts from [Azure/secrets-store-csi-driver-provider-azure](https://github.com/Azure/secrets-store-csi-driver-provider-azure/tree/master/charts/csi-secrets-store-provider-azure), then run the following command to uninstall:

```bash
helm delete <release name>
```

##### Using deployment yamls

If the driver and provider were installed using deployment yamls, then you can delete all the components with the following commands:

```bash
# To delete AKV provider pods from Linux nodes
kubectl delete -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/deployment/provider-azure-installer.yaml

# To delete AKV provider pods from Windows nodes
kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/deployment/provider-azure-installer-windows.yaml
```

Delete the Secrets Store CSI Driver by running `kubectl delete` with all the manifests in [here](https://github.com/kubernetes-sigs/secrets-store-csi-driver/tree/master/deploy). If the Secrets Store CSI Driver was installed using the helm charts hosted in [kubernetes-sigs/secrets-store-csi-driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver/tree/master/charts/secrets-store-csi-driver), then run the following command to delete the driver components:

```bash
helm delete <release name>
```
