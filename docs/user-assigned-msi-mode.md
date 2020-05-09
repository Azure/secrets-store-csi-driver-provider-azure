# VMSS User Assigned Managed Identity

> Supported with Linux and Windows

This option allows azure KeyVault to use the user assigned managed identity on the k8s cluster VMSS directly.

AKS does not support user assigned managed identities yet, only system assigned managed identities. Until this gap is covered, use a service principal or system assigned identities with AKS.

> You can create AKS with [managed identities](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity) now and then you can skip steps 1 and 2. To be able to get the CLIENT ID, the user can run the following command
>
>```bash
>az aks show -g <resource group> -n <aks cluster name> --query identityProfile.kubeletidentity.clientId -o tsv
>```

1. Create Azure Managed Identity

```bash
az identity create -g <RESOURCE GROUP> -n <IDENTITY NAME>
```

2. Assign Azure Managed Identity to VMSS

```bash
az vmss identity assign -g <RESOURCE GROUP> -n <K8S-AGENT-POOL-VMSS> --identities <USER ASSIGNED IDENTITY RESOURCE ID>
```

3. Grant Azure Managed Identity KeyVault permissions

   Ensure that your Azure Identity has the role assignments required to see your Key Vault instance and to access its content. Run the following Azure CLI commands to assign these roles if needed:

   ```bash
   # set policy to access keys in your Key Vault
   az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --spn <YOUR AZURE MANAGED IDENTITY CLIENT ID>
   # set policy to access secrets in your Key Vault
   az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --spn <YOUR AZURE MANAGED IDENTITY CLIENT ID>
   # set policy to access certs in your Key Vault
   az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --spn <YOUR AZURE MANAGED IDENTITY CLIENT ID>
   ```

4. Deploy your application. Specify `useVMManagedIdentity` to `true` and provide `userAssignedIdentityID`.

```yaml
useVMManagedIdentity: "true"               # [OPTIONAL available for version > 0.0.4] if not provided, will default to "false"
userAssignedIdentityID: "clientid"      # [OPTIONAL available for version > 0.0.4] use the client id to specify which user assigned managed identity to use. If using a user assigned identity as the VM's managed identity, then specify the identity's client id. If empty, then defaults to use the system assigned identity on the VM
```
