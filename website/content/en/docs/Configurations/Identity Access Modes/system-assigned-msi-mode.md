---
title: "VMSS System Assigned Managed Identity"
linkTitle: "VMSS System Assigned Managed Identity"
weight: 4
description: >
  Allows Azure KeyVault to use the system assigned managed identity on the k8s cluster VMSS directly
---

> Supported with Linux and Windows

This option allows azure KeyVault to use the system assigned managed identity on the k8s cluster VMSS directly.

AKS uses system-assigned managed identity as [cluster managed identity](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity). This managed identity shouldn't be used to authenticate with KeyVault. You should consider using a [user-assigned managed identity](user-assigned-msi-mode.md) instead.

Before this step, you need to turn on system assigned managed identity on your VMSS clsuter configuration.

1. Verify that the nodes have its own system assigned managed identity

```bash
az vmss identity show -g <resource group>  -n <vmss scalset name> -o yaml
```

The output should contain `type: SystemAssigned` and note `principalId`.

2. Grant Azure Managed Identity KeyVault permissions

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
useVMManagedIdentity: "true"            # [OPTIONAL available for version > 0.0.4] if not provided, will default to "false"
```
