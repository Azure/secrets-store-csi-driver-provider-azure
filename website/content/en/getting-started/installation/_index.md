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

> Important: It's recommended to install the Azure Key Vault Provider for Secrets Store CSI Driver in the `kube-system` namespace using Helm.

```bash
helm install csi csi-secrets-store-provider-azure/csi-secrets-store-provider-azure --namespace kube-system
```

{{% alert title="Why kube-system" color="warning" %}}

1. The driver and provider are installed as a *DaemonSet* with the ability to mount kubelet hostPath volumes and view pod service account tokens. It should be treated as privileged and regular cluster users should not have permissions to deploy or modify the driver.
1. For AKS clusters with [limited egress traffic](https://docs.microsoft.com/en-us/azure/aks/limit-egress-traffic), installing the driver and provider in `kube-system` is required to be able to establish connectivity to the `kube-apiserver`. Refer to [#488](https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/488) for more details.
2. The driver pods need to run as root to mount the volume as tmpfs in the pod. Deploying the driver and provider in `kube-system` will prevent [ASC](https://docs.microsoft.com/en-us/azure/security-center/container-security) from generating alert "Running containers as root user should be avoided". Refer to [#327](https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/327) for more details.

{{% /alert %}}

The helm charts hosted in [Azure/secrets-store-csi-driver-provider-azure](https://github.com/Azure/secrets-store-csi-driver-provider-azure/tree/master/charts/csi-secrets-store-provider-azure) repo include the Secrets Store CSI Driver helm charts as a dependency. Running the above `helm install` command will install both the Secrets Store CSI Driver and Azure Key Vault provider.

> Refer to [doc](../../configurations/deploy-in-openshift.md) for installing the Azure Key Vault Provider for Secrets Store CSI Driver on Azure RedHat OpenShift (ARO)

##### Values

For a list of customizable values that can be injected when invoking helm install, please see the [Helm chart configurations](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/charts/csi-secrets-store-provider-azure/README.md#configuration).

#### Using Deployment yamls

1. **Install the Secrets Store CSI Driver**

    💡 Follow the [Installation guide for the Secrets Store CSI Driver](https://secrets-store-csi-driver.sigs.k8s.io/getting-started/installation.html) to install the driver.

    <details>
    <summary>Result</summary>

    ```bash
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

### Uninstall

#### Using Helm

If you deployed the Secrets Store CSI Driver and Azure Key Vault provider using the helm charts from [Azure/secrets-store-csi-driver-provider-azure](https://github.com/Azure/secrets-store-csi-driver-provider-azure/tree/master/charts/csi-secrets-store-provider-azure), then run the following command to uninstall:

```bash
helm delete <release name>
```

> Refer to [doc](../../configurations/deploy-in-openshift.md) to uninstall the Azure Key Vault Provider for Secrets Store CSI Driver on Azure RedHat OpenShift (ARO)

##### Using deployment yamls

If the driver and provider were installed using deployment yamls, then you can delete all the components with the following commands:

```bash
# To delete AKV provider pods from Linux nodes
kubectl delete -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/deployment/provider-azure-installer.yaml

# To delete AKV provider pods from Windows nodes
kubectl delete -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/deployment/provider-azure-installer-windows.yaml
```

Delete the Secrets Store CSI Driver by running `kubectl delete` with all the manifests in [here](https://github.com/kubernetes-sigs/secrets-store-csi-driver/tree/master/deploy). If the Secrets Store CSI Driver was installed using the helm charts hosted in [kubernetes-sigs/secrets-store-csi-driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver/tree/master/charts/secrets-store-csi-driver), then run the following command to delete the driver components:

```bash
helm delete <release name>
```
