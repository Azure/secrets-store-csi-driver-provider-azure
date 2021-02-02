---
type: docs
title: "Using the Azure Key Vault Provider"
linkTitle: "Using the Azure Key Vault Provider"
weight: 1
description: >
  This guide will walk you through the steps to configure and run the Azure Key Vault provider for Secrets Store CSI driver on Kubernetes.
---

#### Create a new Azure Key Vault resource or use an existing one

In addition to a Kubernetes cluster, you will need an Azure Key Vault resource with secret content to access. Follow [this quickstart tutorial](https://docs.microsoft.com/azure/key-vault/secrets/quick-create-portal) to deploy an Azure Key Vault and add an example secret to it.

Review the settings you desire for your Key Vault, such as what resources (Azure VMs, Azure Resource Manager, etc.) and what kind of network endpoints can access secrets in it.

Take note of the following properties for use in the next section:

1. Name of secret object in Key Vault
1. Secret content type (secret, key, cert)
1. Name of Key Vault resource
1. Azure Tenant ID the Subscription belongs to

#### Create your own SecretProviderClass Object

Create a `SecretProviderClass` custom resource to provide provider-specific parameters for the Secrets Store CSI driver. In this example, use an existing Azure Key Vault or the Azure Key Vault resource created previously.

> NOTE: The `SecretProviderClass` has to be in the same namespace as the pod referencing it.

Update [this sample deployment](https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/examples/v1alpha1_secretproviderclass_service_principal.yaml) to create a `SecretProviderClass` resource to provide Azure-specific parameters for the Secrets Store CSI driver.

To provide identity to access key vault, refer to the following [section](#provide-identity-to-access-key-vault).

  ```yaml
  apiVersion: secrets-store.csi.x-k8s.io/v1alpha1
  kind: SecretProviderClass
  metadata:
    name: azure-kvname
  spec:
    provider: azure
    parameters:
      usePodIdentity: "false"               # [OPTIONAL] if not provided, will default to "false"
      useVMManagedIdentity: "false"         # [OPTIONAL available for version > 0.0.4] if not provided, will default to "false"
      userAssignedIdentityID: "client_id"   # [OPTIONAL available for version > 0.0.4] use the client id to specify which user assigned managed identity to use. If using a user assigned identity as the VM's managed identity, then specify the identity's client id. If empty, then defaults to use the system assigned identity on the VM
      keyvaultName: "kvname"                # the name of the KeyVault
      cloudName: ""                         # [OPTIONAL available for version > 0.0.4] if not provided, azure environment will default to AzurePublicCloud
      cloudEnvFileName: ""                  # [OPTIONAL available for version > 0.0.7] use to define path to file for populating azure environment
      objects:  |
        array:
          - |
            objectName: secret1
            objectAlias: SECRET_1           # [OPTIONAL available for version > 0.0.4] object alias
            objectType: secret              # object types: secret, key or cert. For Key Vault certificates, refer to https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/docs/getting-certs-and-keys.md for the object type to use
            objectVersion: ""               # [OPTIONAL] object versions, default to latest if empty
          - |
            objectName: key1
            objectAlias: ""                 # If provided then it has to be referenced in [secretObjects].[objectName] to sync with Kubernetes secrets 
            objectType: key
            objectVersion: ""
      tenantId: "tid"                       # the tenant ID of the KeyVault

  ```

  | Name                   | Required | Description                                                                                                                                                                                                     | Default Value |
  | ---------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------- |
  | provider               | yes      | specify name of the provider                                                                                                                                                                                    | ""            |
  | usePodIdentity         | no       | set to true for using aad-pod-identity to access keyvault                                                                                                                                                       | "false"       |
  | useVMManagedIdentity   | no       | [__*available for version > 0.0.4*__] specify access mode to enable use of User-assigned managed identity                                                                                                       | "false"       |
  | userAssignedIdentityID | no       | [__*available for version > 0.0.4*__] the user assigned identity ID is required for User-assigned Managed Identity mode                                                                                         | ""            |
  | keyvaultName           | yes      | name of a Key Vault instance                                                                                                                                                                                    | ""            |
  | cloudName              | no       | [__*available for version > 0.0.4*__] name of the azure cloud based on azure go sdk (AzurePublicCloud, AzureUSGovernmentCloud, AzureChinaCloud, AzureGermanCloud, AzureStackCloud)                              | ""            |
  | cloudEnvFileName       | no       | [__*available for version > 0.0.7*__] path to the file to be used while populating the Azure Environment (required if target cloud is AzureStackCloud). More details [here](#other-azure-clouds).               | ""            |
  | objects                | yes      | a string of arrays of strings                                                                                                                                                                                   | ""            |
  | objectName             | yes      | name of a Key Vault object                                                                                                                                                                                      | ""            |
  | objectAlias            | no       | [__*available for version > 0.0.4*__] specify the filename of the object when written to disk - defaults to objectName if not provided                                                                          | ""            |
  | objectType             | yes      | type of a Key Vault object: secret, key or cert.<br>For Key Vault certificates, refer to [doc](../../configurations/getting-certs-and-keys.md) for the object type to use.</br>                                 | ""            |
  | objectVersion          | no       | version of a Key Vault object, if not provided, will use latest                                                                                                                                                 | ""            |
  | objectFormat           | no       | [__*available for version > 0.0.7*__] the format of the Azure Key Vault object, supported types are pem and pfx. `objectFormat: pfx` is only supported with `objectType: secret` and PKCS12 or ECC certificates | "pem"         |
  | objectEncoding         | no       | [__*available for version > 0.0.8*__] the encoding of the Azure Key Vault secret object, supported types are `utf-8`, `hex` and `base64`. This option is supported only with `objectType: secret`               | "utf-8"       |
  | tenantId               | yes      | tenant ID containing key vault instance                                                                                                                                                                         | ""            |

#### Provide Identity to Access Key Vault

The Azure Key Vault Provider offers four modes for accessing a Key Vault instance:

1. [Service Principal](../../configurations/identity-access-modes/service-principal-mode)
2. [Pod Identity](../../configurations/identity-access-modes/pod-identity-mode)
3. [User-assigned Managed Identity](../../configurations/identity-access-modes/user-assigned-msi-mode)
4. [System-assigned Managed Identity](../../configurations/identity-access-modes/system-assigned-msi-mode)

#### Update your Deployment Yaml

To ensure your application is using the Secrets Store CSI driver, update your deployment yaml to use the `secrets-store.csi.k8s.io` driver and reference the `SecretProviderClass` resource created in the previous step.

Update your [linux deployment yaml](https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/examples/nginx-pod-inline-volume-service-principal.yaml) or [windows deployment yaml](https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/examples/windows-pod-secrets-store-inline-volume-secret-providerclass.yaml) to use the Secrets Store CSI driver and reference the `SecretProviderClass` resource created in the previous step.

  ```yaml
    volumes:
      - name: secrets-store-inline
        csi:
          driver: secrets-store.csi.k8s.io
          readOnly: true
          volumeAttributes:
            secretProviderClass: "azure-kvname"
  ```

#### Deploy your Kubernetes Resources

  1. Deploy the SecretProviderClass yaml created previously. For example:

     `kubectl apply -f ./examples/v1alpha1_secretproviderclass.yaml`

  1. Deploy the application yaml created previously. For example:

     `kubectl apply -f ./examples/nginx-pod-inline-volume-service-principal.yaml`

#### Validate the secret

To validate, once the pod is started, you should see the new mounted content at the volume path specified in your deployment yaml.

  ```bash
  ## show secrets held in secrets-store
  kubectl exec nginx-secrets-store-inline -- ls /mnt/secrets-store/

  ## print a test secret held in secrets-store
  kubectl exec nginx-secrets-store-inline -- cat /mnt/secrets-store/secret1
  ```
