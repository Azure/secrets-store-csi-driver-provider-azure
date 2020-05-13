# csi-secrets-store-provider-azure

Azure Key Vault provider for Secrets Store CSI driver allows you to get secret contents stored in Azure Key Vault instance and use the Secrets Store CSI driver interface to mount them into Kubernetes pods.

## Installation

Quick start instructions for the setup and configuration of secrets-store-csi-driver and azure keyvault provider using deployment yamls.


### Install the Secrets Store CSI Driver

💡 Follow the [Installation guide for the Secrets Store CSI Driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver#usage) to install the driver.

To validate the driver is running as expected, run the following commands:

```bash
kubectl get pods -l app=csi-secrets-store
```

You should see the driver pods running on each agent node:

```bash
NAME                                     READY   STATUS    RESTARTS   AGE
csi-secrets-store-jlls6                  1/1     Running   0          10s
csi-secrets-store-qt2l7                  1/1     Running   0          10s
```

### Install the Azure Key Vault Provider

For linux nodes
```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/deployment/provider-azure-installer.yaml
```

For windows nodes
```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/deployment/provider-azure-installer-windows.yaml
```

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