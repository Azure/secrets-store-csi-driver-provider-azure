---
title: "Custom Azure Environments"
linkTitle: "Custom Azure Environments"
weight: 5
description: >
  Pull secret content from KeyVault instances hosted on air-gapped and/or on-prem Azure clouds
---

In order to pull secret content from Key Vault instances hosted on air-gapped and/or on-prem Azure clouds,
your `SecretProviderClass` resource must include the following:

```yaml
parameters:
  cloudName: "AzureStackCloud"
  cloudEnvFileName: "/path/to/custom/environment.json
```

Parameter `cloudEnvFileName` should be the path to a JSON file that contains the custom cloud environment details that
[azure-sdk-for-go](https://github.com/Azure/azure-sdk-for-go) needs to interact with the target Key Vault instance.

Typically, the custom cloud environment file is stored in the file system of the Kubernetes node
and accessible to the `secrets-store-csi-driver` pods through a mounted volume.

Even if the target cloud is not an Azure Stack Hub cloud, cloud name must be set to `"AzureStackCloud"`
to signal `azure-sdk-for-go` to load the custom cloud environment details from `cloudEnvFileName`.

## Environment files

The custom cloud environment sample below shows the minimum set of properties required by `secrets-store-csi-driver-provider-azure`.

```json
{
  "name": "AzureStackCloud",
  "activeDirectoryEndpoint": "https://login.microsoftonline.com/",
  "keyVaultEndpoint": "https://vault.azure.net/",
  "keyVaultDNSSuffix": "vault.azure.net"
}
```
