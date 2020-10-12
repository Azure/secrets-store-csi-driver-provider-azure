# csi-secrets-store-provider-azure

Azure Key Vault provider for Secrets Store CSI driver allows you to get secret contents stored in Azure Key Vault instance and use the Secrets Store CSI driver interface to mount them into Kubernetes pods.

## Helm chart, Secrets Store CSI Driver and Key Vault Provider versions

| Helm Chart Version | Secrets Store CSI Driver Version | Azure Key Vault Provider Version |
| ------------------ | -------------------------------- | -------------------------------- |
| `0.0.5`            | `0.0.9`                          | `0.0.5`                          |
| `0.0.6`            | `0.0.10`                         | `0.0.5`                          |
| `0.0.7`            | `0.0.11`                         | `0.0.6`                          |
| `0.0.8`            | `0.0.11`                         | `0.0.7`                          |
| `0.0.9`            | `0.0.12`                         | `0.0.7`                          |
| `0.0.10`           | `0.0.13`                         | `0.0.8`                          |
| `0.0.11`           | `0.0.14`                         | `0.0.9`                          |
| `0.0.12`           | `0.0.15`                         | `0.0.9`                          |
| `0.0.13`           | `0.0.16`                         | `0.0.9`                          |

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

> Refer to [doc](https://github.com/kubernetes-sigs/secrets-store-csi-driver/tree/master/charts/secrets-store-csi-driver/README.md) for configurable parameters of the secrets-store-csi-driver chart.

| Parameter                                                        | Description                                                                                                                   | Default                                                                                          |
| ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `nameOverride`                                                   | String to partially override csi-secrets-store-provider-azure.fullname template with a string (will prepend the release name) | `""`                                                                                             |
| `fullnameOverride`                                               | String to fully override csi-secrets-store-provider-azure.fullname template with a string                                     | `""`                                                                                             |
| `image.repository`                                               | Image repository                                                                                                              | `mcr.microsoft.com/oss/azure/secrets-store/provider-azure`                                       |
| `image.pullPolicy`                                               | Image pull policy                                                                                                             | `IfNotPresent`                                                                                   |
| `image.tag`                                                      | Azure Keyvault Provider image                                                                                                 | `0.0.9`                                                                                          |
| `imagePullSecrets`                                               | Secrets to be used when pulling images                                                                                        | `[]`                                                                                             |
| `linux.enabled`                                                  | Install azure keyvault provider on linux nodes                                                                                | true                                                                                             |
| `linux.nodeSelector`                                             | Node Selector for the daemonset on linux nodes                                                                                | `{}`                                                                                             |
| `linux.tolerations`                                              | Tolerations for the daemonset on linux nodes                                                                                  | `{}`                                                                                             |
| `linux.resources`                                                | Resource limit for provider pods on linux nodes                                                                               | `requests.cpu: 50m`<br>`requests.memory: 100Mi`<br>`limits.cpu: 50m`<br>`limits.memory: 100Mi`   |
| `windows.enabled`                                                | Install azure keyvault provider on windows nodes                                                                              | false                                                                                            |
| `windows.nodeSelector`                                           | Node Selector for the daemonset on windows nodes                                                                              | `{}`                                                                                             |
| `windows.tolerations`                                            | Tolerations for the daemonset on windows nodes                                                                                | `{}`                                                                                             |
| `windows.resources`                                              | Resource limit for provider pods on windows nodes                                                                             | `requests.cpu: 100m`<br>`requests.memory: 200Mi`<br>`limits.cpu: 100m`<br>`limits.memory: 200Mi` |
| `secrets-store-csi-driver.install`                               | Install secrets-store-csi-driver with this chart                                                                              | true                                                                                             |
| `secrets-store-csi-driver.linux.enabled`                         | Install secrets-store-csi-driver on linux nodes                                                                               | true                                                                                             |
| `secrets-store-csi-driver.linux.kubeletRootDir`                  | Configure the kubelet root dir                                                                                                | `/var/lib/kubelet`                                                                               |
| `secrets-store-csi-driver.linux.metricsAddr`                     | The address the metric endpoint binds to                                                                                      | `:8080`                                                                                          |
| `secrets-store-csi-driver.linux.image.repository`                | Driver Linux image repository                                                                                                 | `mcr.microsoft.com/oss/kubernetes-csi/secrets-store/driver`                                      |
| `secrets-store-csi-driver.linux.image.pullPolicy`                | Driver Linux image pull policy                                                                                                | `IfNotPresent`                                                                                   |
| `secrets-store-csi-driver.linux.image.tag`                       | Driver Linux image tag                                                                                                        | `v0.0.16`                                                                                        |
| `secrets-store-csi-driver.linux.registrarImage.repository`       | Driver Linux node-driver-registrar image repository                                                                           | `mcr.microsoft.com/oss/kubernetes-csi/csi-node-driver-registrar`                                 |
| `secrets-store-csi-driver.linux.registrarImage.pullPolicy`       | Driver Linux node-driver-registrar image pull policy                                                                          | `IfNotPresent`                                                                                   |
| `secrets-store-csi-driver.linux.registrarImage.tag`              | Driver Linux node-driver-registrar image tag                                                                                  | `v1.2.0`                                                                                         |
| `secrets-store-csi-driver.linux.livenessProbeImage.repository`   | Driver Linux liveness-probe image repository                                                                                  | `mcr.microsoft.com/oss/kubernetes-csi/livenessprobe`                                             |
| `secrets-store-csi-driver.linux.livenessProbeImage.pullPolicy`   | Driver Linux liveness-probe image pull policy                                                                                 | `IfNotPresent`                                                                                   |
| `secrets-store-csi-driver.linux.livenessProbeImage.tag`          | Driver Linux liveness-probe image tag                                                                                         | `v2.0.0`                                                                                         |
| `secrets-store-csi-driver.windows.enabled`                       | Install secrets-store-csi-driver on windows nodes                                                                             | false                                                                                            |
| `secrets-store-csi-driver.windows.kubeletRootDir`                | Configure the kubelet root dir                                                                                                | `C:\var\lib\kubelet`                                                                             |
| `secrets-store-csi-driver.windows.metricsAddr`                   | The address the metric endpoint binds to                                                                                      | `:8080`                                                                                          |
| `secrets-store-csi-driver.windows.image.repository`              | Driver Windows image repository                                                                                               | `mcr.microsoft.com/oss/kubernetes-csi/secrets-store/driver`                                      |
| `secrets-store-csi-driver.windows.image.pullPolicy`              | Driver Windows image pull policy                                                                                              | `IfNotPresent`                                                                                   |
| `secrets-store-csi-driver.windows.image.tag`                     | Driver Windows image tag                                                                                                      | `v0.0.16`                                                                                        |
| `secrets-store-csi-driver.windows.registrarImage.repository`     | Driver Windows node-driver-registrar image repository                                                                         | `mcr.microsoft.com/oss/kubernetes-csi/csi-node-driver-registrar`                                 |
| `secrets-store-csi-driver.windows.registrarImage.pullPolicy`     | Driver Windows node-driver-registrar image pull policy                                                                        | `IfNotPresent`                                                                                   |
| `secrets-store-csi-driver.windows.registrarImage.tag`            | Driver Windows node-driver-registrar image tag                                                                                | `v1.2.1-alpha.1-windows-1809-amd64`                                                              |
| `secrets-store-csi-driver.windows.livenessProbeImage.repository` | Driver Windows liveness-probe image repository                                                                                | `mcr.microsoft.com/oss/kubernetes-csi/livenessprobe`                                             |
| `secrets-store-csi-driver.windows.livenessProbeImage.pullPolicy` | Driver Windows liveness-probe image pull policy                                                                               | `IfNotPresent`                                                                                   |
| `secrets-store-csi-driver.windows.livenessProbeImage.tag`        | Driver Windows liveness-probe image tag                                                                                       | `v2.0.1-alpha.1-windows-1809-amd64`                                                              |
| `secrets-store-csi-driver.logLevel.debug`                        | Enable debug logging                                                                                                          | `true`                                                                                           |
| `secrets-store-csi-driver.grpcSupportedProviders`                | ; delimited list of providers that support grpc for driver-provider [alpha]                                                   | `azure`                                                                                          |
| `secrets-store-csi-driver.enableSecretRotation`                  | Enable secret rotation feature [alpha]                                                                                        | `false`                                                                                          |
| `secrets-store-csi-driver.rotationPollInterval`                  | Secret rotation poll interval duration                                                                                        | `2m`                                                                                             |
| `rbac.install`                                                   | Install default service account                                                                                               | true                                                                                             |
