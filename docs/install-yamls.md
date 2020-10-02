# csi-secrets-store-provider-azure

Azure Key Vault provider for Secrets Store CSI driver allows you to get secret contents stored in Azure Key Vault instance and use the Secrets Store CSI driver interface to mount them into Kubernetes pods.

## Installation

Quick start instructions for the setup and configuration of secrets-store-csi-driver and azure keyvault provider using deployment yamls.


### Install the Secrets Store CSI Driver

💡 Follow the [Installation guide for the Secrets Store CSI Driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver#usage) to install the driver.

> **NOTE:** `0.0.9+` release of the Azure Key Vault provider is incompatible with the Secrets Store CSI Driver versions < `v0.0.14`. While installing the Secrets Store CSI Driver using yamls, add the following flag `--grpc-supported-providers=azure` to the [Linux](https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/master/deploy/secrets-store-csi-driver.yaml) and [Windows](https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/master/deploy/secrets-store-csi-driver-windows.yaml) daemonset manifests.
> - The flag needs to be added to the `secrets-store` container args

To validate the driver is running as expected, run the following command:

```bash
kubectl get pods -l app=csi-secrets-store
```

You should see the driver pods running on each agent node:

```bash
NAME                                     READY   STATUS    RESTARTS   AGE
csi-secrets-store-jlls6                  1/1     Running   0          10s
csi-secrets-store-qt2l7                  1/1     Running   0          10s
```

To validate the `--grpc-supported-providers=azure` arg has been configured correctly, run the following command:

```bash
kubectl get ds -l app=csi-secrets-store -o jsonpath='{range .items[*]}{.spec.template.spec.containers[1].args}{"\n"}'
```

You should see the args for the `secrets-store` container in the driver pods for each node:
```bash
["--debug=true","--endpoint=$(CSI_ENDPOINT)","--nodeid=$(KUBE_NODE_NAME)","--provider-volume=/etc/kubernetes/secrets-store-csi-providers","--grpc-supported-providers=azure","--metrics-addr=:8080"]
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