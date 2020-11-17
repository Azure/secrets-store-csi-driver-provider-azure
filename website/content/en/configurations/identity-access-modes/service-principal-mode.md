---
type: docs
title: "Service Principal"
linkTitle: "Service Principal"
weight: 1
description: >
  Use a Service Principal
---

> Supported with Linux and Windows

1. Add your service principal credentials as a Kubernetes secrets accessible by the Secrets Store CSI driver. If using AKS you can learn about [service principals in AKS here.](https://docs.microsoft.com/azure/aks/kubernetes-service-principal)

    A properly configured service principal will need to be passed in with the SP's App ID and Password. Ensure this service principal has all the required permissions to access content in your Azure Key Vault instance.

    ```bash
    # Client ID (AZURE_CLIENT_ID) will be the App ID of your service principal
    # Client Secret (AZURE_CLIENT_SECRET) will be the Password of your service principal

    kubectl create secret generic secrets-store-creds --from-literal clientid=<AZURE_CLIENT_ID> --from-literal clientsecret=<AZURE_CLIENT_SECRET>
    ```

    **If you do not have a service principal**, run the following Azure CLI command to create a new service principal.

    ```bash
    # OPTIONAL: Create a new service principal, be sure to notate the SP secret returned on creation.
    az ad sp create-for-rbac --skip-assignment --name $SPNAME

    # If you lose your AZURE_CLIENT_SECRET (SP Secret), you can reset and receive it with this command:
    # az ad sp credential reset --name $SPNAME --credential-description "APClientSecret" --query password -o tsv
    ```

    With an existing service principal, assign the following permissions.

    ```bash
    # Set environment variables
    SPNAME=<servicePrincipalName>
    AZURE_CLIENT_ID=$(az ad sp show --id http://${SPNAME} --query appId -o tsv)
    KEYVAULT_NAME=<key-vault-name>
    KEYVAULT_RESOURCE_GROUP=<resource-group-name-for-KV>
    SUBID=<subscription-id>

    # Assign Reader Role to the service principal for your keyvault
    az role assignment create --role Reader --assignee $AZURE_CLIENT_ID --scope /subscriptions/$SUBID/resourcegroups/$KEYVAULT_RESOURCE_GROUP/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME

    az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --spn $AZURE_CLIENT_ID
    az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --spn $AZURE_CLIENT_ID
    az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --spn $AZURE_CLIENT_ID
    ```

2. Update your [linux deployment yaml](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/examples/nginx-pod-inline-volume-service-principal.yaml) or [windows deployment yaml](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/examples/windows-pod-secrets-store-inline-volume-secret-providerclass.yaml) to reference the service principal kubernetes secret created in the previous step

    If you did not change the name of the secret reference previously, no changes are needed.

    ```yaml
    nodePublishSecretRef:
      name: secrets-store-creds
    ```
