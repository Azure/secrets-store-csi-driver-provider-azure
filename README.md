# Azure Key Vault Provider for Secret Store CSI Driver

[![Build Status](https://dev.azure.com/azure/secrets-store-csi-driver-provider-azure/_apis/build/status/secrets-store-csi-driver-provider-azure-ci?branchName=master)](https://dev.azure.com/azure/secrets-store-csi-driver-provider-azure/_build/latest?definitionId=67&branchName=master)

Azure Key Vault provider for [Secret Store CSI driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver) allows you to get secret contents stored in an [Azure Key Vault](https://docs.microsoft.com/en-us/azure/key-vault/general/overview) instance and use the Secrets Store CSI driver interface to mount them into Kubernetes pods.

## Demo

_WIP_

## Usage

This guide will walk you through the steps to configure and run the Azure Key Vault provider for Secret Store CSI driver on Kubernetes.

### Install the Secrets Store CSI Driver and the Azure Keyvault Provider
**Prerequisites**

Recommended Kubernetes version:
- For Linux - v1.16.0+
- For Windows - v1.18.0+

> For Kubernetes version 1.15, please use [Azure Keyvault Flexvolume](https://github.com/Azure/kubernetes-keyvault-flexvol)

**Deployment using Helm**

Follow [this guide to install using Helm](charts/csi-secrets-store-provider-azure/README.md)

<details>
<summary><strong>[ALTERNATIVE DEPLOYMENT OPTION] Using Deployment Yamls</strong></summary>

### Install the Secrets Store CSI Driver

💡 Follow the [Installation guide for the Secrets Store CSI Driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver#usage) to install the driver.


### Install the Azure Key Vault Provider

For linux nodes
```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/deployment/provider-azure-installer.yaml
```

For windows nodes
```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/deployment/provider-azure-installer-windows.yaml
```

To validate the provider's installer is running as expected, run the following commands:

```bash
kubectl get pods -l app=csi-secrets-store-provider-azure
```

You should see the provider pods running on each agent node:

```bash
NAME                                     READY   STATUS    RESTARTS   AGE
csi-secrets-store-provider-azure-4ngf4   1/1     Running   0          8s
csi-secrets-store-provider-azure-bxr5k   1/1     Running   0          8s
```
</details>

### Using the Azure Key Vault Provider

#### Create a new Azure Key Vault resource or use an existing one

In addition to a Kubernetes cluster, you will need an Azure Key Vault resource with secret content to access. Follow [this quickstart tutorial](https://docs.microsoft.com/azure/key-vault/secrets/quick-create-portal) to deploy an Azure Key Vault and add an example secret to it.

Review the settings you desire for your Key Vault, such as what resources (Azure VMs, Azure Resource Manager, etc.) and what kind of network endpoints can access secrets in it.

Take note of the following properties for use in the next section:

1. Name of secret object in Key Vault
1. Secret content type (secret, key, cert)
1. Name of Key Vault resource
1. Resource group the Key Vault resides within
1. Azure Subscription ID the Key Vault was provisioned with
1. Azure Tenant ID the Subscription ID belongs to

#### Create secretproviderclasses

Create a `secretproviderclasses` resource to provide provider-specific parameters for the Secrets Store CSI driver. In this example, use an existing Azure Key Vault or the Azure Key Vault resource created previously.

1. Update [this sample deployment](examples/v1alpha1_secretproviderclass.yaml) to create a `secretproviderclasses` resource to provide Azure-specific parameters for the Secrets Store CSI driver.

    ```yaml
    apiVersion: secrets-store.csi.x-k8s.io/v1alpha1
    kind: SecretProviderClass
    metadata:
      name: azure-kvname
    spec:
      provider: azure                   # accepted provider options: azure or vault
      parameters:
        usePodIdentity: "false"         # [OPTIONAL for Azure] if not provided, will default to "false"
        useVMManagedIdentity: "false"   # [OPTIONAL available for version > 0.0.4] if not provided, will default to "false"
        userAssignedIdentityID: "client_id"  # [OPTIONAL available for version > 0.0.4] use the client id to specify which user assigned managed identity to use. If using a user assigned identity as the VM's managed identity, then specify the identity's client id. If empty, then defaults to use the system assigned identity on the VM
        keyvaultName: "kvname"          # the name of the KeyVault
        cloudName: ""          # [OPTIONAL available for version > 0.0.4] if not provided, azure environment will default to AzurePublicCloud
        objects:  |
          array:
            - |
              objectName: secret1
              objectAlias: SECRET_1     # [OPTIONAL available for version > 0.0.4] object alias
              objectType: secret        # object types: secret, key or cert
              objectVersion: ""         # [OPTIONAL] object versions, default to latest if empty
            - |
              objectName: key1
              objectAlias: ""
              objectType: key
              objectVersion: ""
        resourceGroup: "rg1"            # [REQUIRED for version < 0.0.4] the resource group of the KeyVault
        subscriptionId: "subid"         # [REQUIRED for version < 0.0.4] the subscription ID of the KeyVault
        tenantId: "tid"                 # the tenant ID of the KeyVault

    ```

    | Name                   | Required | Description                                                     | Default Value |
    | -----------------------| -------- | --------------------------------------------------------------- | ------------- |
    | provider               | yes      | specify name of the provider                                    | ""            |
    | usePodIdentity         | no       | specify access mode: service principal or pod identity          | "false"       |
    | useVMManagedIdentity   | no       | [__*available for version > 0.0.4*__] specify access mode to enable use of VM's managed identity    |  "false"|
    | userAssignedIdentityID | no       | [__*available for version > 0.0.4*__] the user assigned identity ID is required for VMSS User Assigned Managed Identity mode  | ""       |
    | keyvaultName           | yes      | name of a Key Vault instance                                    | ""            |
    | cloudName              | no       | [__*available for version > 0.0.4*__] name of the azure cloud based on azure go sdk (AzurePublicCloud,AzureUSGovernmentCloud, AzureChinaCloud, AzureGermanCloud)| "" |
    | objects                | yes      | a string of arrays of strings                                   | ""            |
    | objectName             | yes      | name of a Key Vault object                                      | ""            |
    | objectAlias            | no       | [__*available for version > 0.0.4*__] specify the filename of the object when written to disk - defaults to objectName if not provided | "" |
    | objectType             | yes      | type of a Key Vault object: secret, key or cert                 | ""            |
    | objectVersion          | no       | version of a Key Vault object, if not provided, will use latest | ""            |
    | resourceGroup          | no      | [__*required for version < 0.0.4*__] name of resource group containing key vault instance            | ""            |
    | subscriptionId         | no      | [__*required for version < 0.0.4*__] subscription ID containing key vault instance                   | ""            |
    | tenantId               | yes      | tenant ID containing key vault instance                         | ""            |

1. Update your [linux deployment yaml](examples/nginx-pod-secrets-store-inline-volume-secretproviderclass.yaml) or [windows deployment yaml](examples/windows-pod-secrets-store-inline-volume-secretproviderclass.yaml) to use the Secrets Store CSI driver and reference the `secretProviderClass` resource created in the previous step. 

      If you did not change the name of the secretProviderClass previously, no changes are needed.
    
      ```yaml
        volumes:
          - name: secrets-store-inline
            csi:
              driver: secrets-store.csi.k8s.io
              readOnly: true
              volumeAttributes:
                secretProviderClass: "azure-kvname"
      ```

  1. Select and complete an option from [below to enable access to the Key Vault](#provide-identity-to-access-key-vault)

  1. Deploy the secretProviderClass

     `kubectl apply -f ./examples/v1alpha1_secretproviderclass.yaml`

  1. Deploy the Linux deployment yaml

     `kubectl apply -f ./examples/nginx-pod-secrets-store-inline-volume-secretproviderclass.yaml`

#### Validate the secret

1. To validate, once the pod is started, you should see the new mounted content at the volume path specified in your deployment yaml.

    ```bash
    ## show secrets held in secrets-store
    kubectl exec -it nginx-secrets-store-inline ls /mnt/secrets-store/

    ## print a test secret held in secrets-store
    kubectl exec -it nginx-secrets-store-inline cat /mnt/secrets-store/secret1
    ```

#### Provide Identity to Access Key Vault

The Azure Key Vault Provider offers four modes for accessing a Key Vault instance:

1. [Service Principal](docs/service-principal-mode.md)
1. [Pod Identity](docs/pod-identity-mode.md)
1. [VMSS User Assigned Managed Identity](docs/user-assigned-msi-mode.md)
1. [VMSS System Assigned Managed Identity](docs/system-assigned-msi-mode.md)
