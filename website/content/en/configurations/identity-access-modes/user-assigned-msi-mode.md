---
type: docs
title: "User-assigned Managed Identity"
linkTitle: "User-assigned Managed Identity"
weight: 3
description: >
  Use a User-assigned Managed Identity to access Keyvault.
---

> Supported with Linux and Windows

<details>
<summary>Examples</summary>

- `SecretProviderClass`
```yaml
# This is a SecretProviderClass example using user-assigned identity to access Key Vault
apiVersion: secrets-store.csi.x-k8s.io/v1alpha1
kind: SecretProviderClass
metadata:
  name: azure-kvname-user-msi
spec:
  provider: azure
  parameters:
    usePodIdentity: "false"
    useVMManagedIdentity: "true"
    userAssignedIdentityID: "<client id of user assigned identity>"
    keyvaultName: "kvname"
    cloudName: ""                   # [OPTIONAL for Azure] if not provided, azure environment will default to AzurePublicCloud
    objects:  |
      array:
        - |
          objectName: secret1
          objectType: secret        # object types: secret, key or cert
          objectVersion: ""         # [OPTIONAL] object versions, default to latest if empty
        - |
          objectName: key1
          objectType: key
          objectVersion: ""
    tenantId: "tid"                 # the tenant ID of the KeyVault  
``` 

- `Pod` yaml
```yaml
# This is a sample pod definition for using SecretProviderClass and user-assigned identity to access Key Vault
kind: Pod
apiVersion: v1
metadata:
  name: nginx-secrets-store-inline-user-msi
spec:
  containers:
    - name: nginx
      image: nginx
      volumeMounts:
      - name: secrets-store01-inline
        mountPath: "/mnt/secrets-store"
        readOnly: true
  volumes:
    - name: secrets-store01-inline
      csi:
        driver: secrets-store.csi.k8s.io
        readOnly: true
        volumeAttributes:
          secretProviderClass: "azure-kvname-user-msi"
```
</details>

## Configure User-assigned Managed Identity to access Keyvault

In AKS you can use the [User-assigned Kubelet managed identity](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity) (doesn't support BYO today) or create your own user-assigned managed identity as described below.

> You can create an AKS cluster with [managed identities](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity) now and then skip steps 1 and 2. To get the `clientID` of the managed identity, run the following command:
>
>```bash
>az aks show -g <resource group> -n <aks cluster name> --query identityProfile.kubeletidentity.clientId -o tsv
>```

1. Create Azure User-assigned Managed Identity

    ```bash
    az identity create -g <RESOURCE GROUP> -n <IDENTITY NAME>
    ```

2. Assign Azure User-assigned Managed Identity to VM/VMSS

    For VMSS:
    ```bash
    az vmss identity assign -g <RESOURCE GROUP> -n <K8S-AGENT-POOL-VMSS> --identities <USER ASSIGNED IDENTITY RESOURCE ID>
    ```

    If the cluster is using `AvailabilitySet`, then assign the identity to each of the VM instances:
    ```bash
    az vm identity assign -g <RESOURCE GROUP> -n <K8S-AGENT-POOL-VM> --identities <USER ASSIGNED IDENTITY RESOURCE ID>
    ```

3. Grant User-assigned Managed Identity permission to access Keyvault

   Ensure that your User-assigned Managed Identity has the role assignments required to access content in keyvault instance. Run the following Azure CLI commands to assign the roles if required:

   ```bash
   # set policy to access keys in your Keyvault
   az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --spn <USER-ASSIGNED MANAGED IDENTITY CLIENTID>
   # set policy to access secrets in your Keyvault
   az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --spn <USER-ASSIGNED MANAGED IDENTITY CLIENTID>
   # set policy to access certs in your Keyvault
   az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --spn <USER-ASSIGNED MANAGED IDENTITY CLIENTID>
   ```

4. Deploy your application. Specify `useVMManagedIdentity` to `true` and provide `userAssignedIdentityID`.

    ```yaml
    useVMManagedIdentity: "true"
    userAssignedIdentityID: "<client id of the managed identity>"
    ```
