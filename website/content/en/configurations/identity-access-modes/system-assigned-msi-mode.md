---
type: docs
title: "System-assigned Managed Identity"
linkTitle: "System-assigned Managed Identity"
weight: 4
description: >
  Use a System-assigned Managed Identity to access Keyvault.
---

> Supported with Linux and Windows

<details>
<summary>Examples</summary>

- `SecretProviderClass`
```yaml
# This is a SecretProviderClass example using system-assigned identity to access Key Vault
apiVersion: secrets-store.csi.x-k8s.io/v1alpha1
kind: SecretProviderClass
metadata:
  name: azure-kvname-system-msi
spec:
  provider: azure
  parameters:
    usePodIdentity: "false"
    useVMManagedIdentity: "true"
    userAssignedIdentityID: ""      # If empty, then defaults to use the system assigned identity on the VM
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

# This is a sample pod definition for using SecretProviderClass and system-assigned identity to access Key Vault
kind: Pod
apiVersion: v1
metadata:
  name: nginx-secrets-store-inline-system-msi
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
          secretProviderClass: "azure-kvname-system-msi"
```
</details>

## Configure System-assigned Managed Identity to access Keyvault

AKS uses system-assigned managed identity as [cluster managed identity](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity). This managed identity shouldn't be used to access Keyvault. You should consider using a [User-assigned managed identity](./user-assigned-msi-mode) instead.

Before this step, you need to turn on system-assigned managed identity on your cluster VM/VMSS.

1. Verify that the nodes have their own system-assigned managed identity

    For VMSS:
    ```bash
    az vmss identity show -g <resource group>  -n <vmss scalset name> -o yaml
    ```

    If the cluster is using `AvailabilitySet`, then check the system-assigned identity exists on all the VM instances:
    ```bash
    az vm identity show -g <resource group> -n <vm name> -o yaml
    ```
    The output should contain `type: SystemAssigned`. Take a note of the `principalId`.

2. Grant Azure Managed Identity permission to access Keyvault

   Ensure that your Azure Identity has the role assignments required to see your Key Vault instance and to access its content. Run the following Azure CLI commands to assign these roles if needed:

   ```bash
   # set policy to access keys in your Key Vault
   az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --object-id <YOUR AZURE VMSS PRINCIPALID>
   # set policy to access secrets in your Key Vault
   az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --object-id <YOUR AZURE VMSS PRINCIPALID>
   # set policy to access certs in your Key Vault
   az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --object-id <YOUR AZURE VMSS PRINCIPALID>
   ```

3. Deploy your application. Specify `useVMManagedIdentity` to `true`.

    ```yaml
    useVMManagedIdentity: "true"
    ```
