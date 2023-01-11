---
type: docs
title: "System-assigned Managed Identity"
linkTitle: "System-assigned Managed Identity"
weight: 2
description: >
  Use a System-assigned Managed Identity to access Keyvault.
---

<details>
<summary>Examples</summary>

- `SecretProviderClass`
```yaml
# This is a SecretProviderClass example using system-assigned identity to access Key Vault
apiVersion: secrets-store.csi.x-k8s.io/v1
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
    tenantID: "tid"                 # the tenant ID of the KeyVault
``` 

- `Pod` yaml
```yaml

# This is a sample pod definition for using SecretProviderClass and system-assigned identity to access Key Vault
kind: Pod
apiVersion: v1
metadata:
  name: busybox-secrets-store-inline-system-msi
spec:
  containers:
    - name: busybox
      image: k8s.gcr.io/e2e-test-images/busybox:1.29
      command:
        - "/bin/sleep"
        - "10000"
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

Before this step, you need to [enable system-assigned managed identity](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/qs-configure-cli-windows-vm#enable-system-assigned-managed-identity-on-an-existing-azure-vm) in your cluster VM/VMSS.

1. Verify that the nodes have their own system-assigned managed identity

    For VMSS:
    ```bash
    az vmss identity show -g <resource group>  -n <vmss scalset name> -o yaml
    ```

    If the cluster is using `AvailabilitySet`, then check the system-assigned identity exists on all the VM instances:
    ```bash
    az vm identity show -g <resource group> -n <vm name> -o yaml
    ```
    The output should contain `type: SystemAssigned`. Make a note of the `principalId`.

2. Grant System-assigned Managed Identity permission to access Keyvault

   Ensure that your System-assigned Managed Identity has the role assignments required to access content in keyvault instance. Run the following Azure CLI commands to assign the roles if required:

   ```bash
   # set policy to access keys in your Keyvault
   az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --object-id <SYSTEM-ASSIGNED MANAGED IDENTITY PRINCIPALID>
   # set policy to access secrets in your Keyvault
   az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --object-id <SYSTEM-ASSIGNED MANAGED IDENTITY PRINCIPALID>
   # set policy to access certs in your Keyvault
   az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --object-id <SYSTEM-ASSIGNED MANAGED IDENTITY PRINCIPALID>
   ```

3. Deploy your application. Specify `useVMManagedIdentity` to `true`.

    ```yaml
    useVMManagedIdentity: "true"
    ```

## Pros:
1. Supported on both Windows and Linux.