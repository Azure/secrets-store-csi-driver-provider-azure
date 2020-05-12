# Service Principal

> Supported with Linux and Windows

1. Add your service principal credentials as a Kubernetes secrets accessible by the Secrets Store CSI driver. If using AKS you can learn about [service principals in AKS here.](https://docs.microsoft.com/azure/aks/kubernetes-service-principal) 

    A properly configured service principal will need to be passed in with the SP's App ID and Password. Ensure this service principal has all the required permissions to access content in your Azure Key Vault instance.

    ```bash
    # Client ID will be the App ID of your service principal
    # Client Secret will be the Password of your service principal

    kubectl create secret generic secrets-store-creds --from-literal clientid=<CLIENTID> --from-literal clientsecret=<CLIENTSECRET>
    ```
    
    **If you do not have a service principal**, run the following Azure CLI command to create a new service principal.

    ```bash
    # OPTIONAL: Create a new service principal, be sure to notate the SP secret returned on creation.
    az ad sp create-for-rbac --skip-assignment --name $SPNAME
    ```

    With an existing service principal, assign the following permissions.

    ```bash
    # Set environment variables
    SPNAME=<servicePrincipalName>
    APPID=$(az ad sp show --id http://${SPNAME} --query appId -o tsv)
    KV_NAME=<key-vault-name>
    RG=<resource-group-name-for-KV>
    SUBID=<subscription-id>

    # Assign Reader Role to the service principal for your keyvault
    az role assignment create --role Reader --assignee $APPID --scope /subscriptions/$SUBID/resourcegroups/$RG/providers/Microsoft.KeyVault/vaults/$KV_NAME
    
    az keyvault set-policy -n $KV_NAME --key-permissions get --spn $APPID
    az keyvault set-policy -n $KV_NAME --secret-permissions get --spn $APPID
    az keyvault set-policy -n $KV_NAME --certificate-permissions get --spn $APPID
    ```

1. Update your [linux deployment yaml](../examples/nginx-pod-secrets-store-inline-volume-secretproviderclass.yaml) or [windows deployment yaml](../examples/windows-pod-secrets-store-inline-volume-secret-providerclass.yaml) to reference the service principal kubernetes secret created in the previous step

    If you did not change the name of the secret reference previously, no changes are needed.

    ```yaml
    nodePublishSecretRef:
      name: secrets-store-creds
    ```
