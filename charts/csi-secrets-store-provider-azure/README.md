# csi-secrets-store-provider-azure

Azure Key Vault provider for Secrets Store CSI driver allows you to get secret contents stored in Azure Key Vault instance and use the Secrets Store CSI driver interface to mount them into Kubernetes pods.

## Installation

Quick start instructions for the setup and configuration of secrets-store-csi-driver and azure keyvault provider using Helm.

### Prerequisites

- [Helm3](https://helm.sh/docs/intro/quickstart/#install-helm)

### Installing the Chart

- This chart installs the [secrets-store-csi-driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver) and the azure keyvault provider for the driver

```shell
$ helm repo add csi-secrets-store-provider-azure https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/charts
$ helm install csi-secrets-store-provider-azure/csi-secrets-store-provider-azure --generate-name
```

### Configuration

The following table lists the configurable parameters of the csi-secrets-store-provider-azure chart and their default values.

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `nameOverride` | String to partially override csi-secrets-store-provider-azure.fullname template with a string (will prepend the release name) | `""` |
| `fullnameOverride` | String to fully override csi-secrets-store-provider-azure.fullname template with a string | `""` |
| `image.repository` | Image repository | `mcr.microsoft.com/k8s/csi/secrets-store/provider-azure` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Azure Keyvault Provider image | `0.0.7` |
| `linux.enabled` | Install azure keyvault provider on linux nodes | true |
| `linux.nodeSelector` | Node Selector for the daemonset on linux nodes | `{}` |
| `linux.resources` | Resource limit for provider pods on linux nodes | `requests.cpu: 50m`<br>`requests.memory: 100Mi`<br>`limits.cpu: 50m`<br>`limits.memory: 100Mi` |
| `windows.enabled` | Install azure keyvault provider on windows nodes | false |
| `windows.nodeSelector` | Node Selector for the daemonset on windows nodes | `{}` |
| `windows.resources` | Resource limit for provider pods on windows nodes | `requests.cpu: 100m`<br>`requests.memory: 200Mi`<br>`limits.cpu: 100m`<br>`limits.memory: 200Mi` |
| `secrets-store-csi-driver.install` | Install secrets-store-csi-driver with this chart | true |
| `secrets-store-csi-driver.linux.enabled` | Install secrets-store-csi-driver on linux nodes | true |
| `secrets-store-csi-driver.linux.kubeletRootDir` | Configure the kubelet root dir | `/var/lib/kubelet` |
| `secrets-store-csi-driver.linux.metricsAddr` | The address the metric endpoint binds to | `:8080` |
| `secrets-store-csi-driver.windows.enabled` | Install secrets-store-csi-driver on windows nodes | false |
| `secrets-store-csi-driver.windows.kubeletRootDir` | Configure the kubelet root dir | `C:\var\lib\kubelet` |
| `secrets-store-csi-driver.windows.metricsAddr` | The address the metric endpoint binds to | `:8080` |
| `secrets-store-csi-driver.logLevel.debug` | Enable debug logging | `true` |
| `rbac.install` | Install default service account | true |
