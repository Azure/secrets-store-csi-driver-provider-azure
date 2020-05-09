**OPTION 4 - VMSS System Assigned Managed Identity**

> Supported with Linux and Windows

This option allows azure KeyVault to use the system assigned managed identity on the k8s cluster VMSS directly.

For AKS clusters, system assigned managed identities can be created on your behalf. To learn more [read here about AKS managed identity support](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity).

1. Verify that the nodes have its own system assigned managed identity

```bash
az vmss identity show -g <resource group>  -n <vmss scalset name> -o yaml
```

The output should contain `type: SystemAssigned`.

2. Grant Azure Managed Identity KeyVault permissions

   Ensure that your Azure Identity has the role assignments required to see your Key Vault instance and to access its content. Run the following Azure CLI commands to assign these roles if needed:

   ```bash
   # set policy to access keys in your Key Vault
   az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --spn <YOUR AZURE MANAGED IDENTITY CLIENT ID>
   # set policy to access secrets in your Key Vault
   az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --spn <YOUR AZURE MANAGED IDENTITY CLIENT ID>
   # set policy to access certs in your Key Vault
   az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --spn <YOUR AZURE MANAGED IDENTITY CLIENT ID>
   ```

3. Deploy your application. Specify `useVMManagedIdentity` to `true`.

```yaml
useVMManagedIdentity: "true"            # [OPTIONAL available for version > 0.0.4] if not provided, will default to "false"
```
