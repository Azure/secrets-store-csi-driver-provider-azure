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
- For linux - v1.16.0+
- For windows - v1.18.0+

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

#### Create secretproviderclasses

Create a `secretproviderclasses` resource to provide provider-specific parameters for the Secrets Store CSI driver. 

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
        cloudName: "cloudname"          # [OPTIONAL available for version > 0.0.4] if not provided, azure environment will default to AzurePublic Cloud
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

1. Update your [linux deployment yaml](examples/nginx-pod-secrets-store-inline-volume-secretproviderclass.yaml) or [windows deployment yaml](examples/windows-pod-secrets-store-inline-volume-secretproviderclass.yaml) to use the Secrets Store CSI driver and reference the `secretProviderClass` resource created in the previous step

    ```yaml
    volumes:
      - name: secrets-store-inline
        csi:
          driver: secrets-store.csi.k8s.io
          readOnly: true
          volumeAttributes:
            secretProviderClass: "azure-kvname"
    ```

#### Provide Identity to Access Key Vault
The Azure Key Vault Provider offers four modes for accessing a Key Vault instance: 
1. Service Principal 
1. Pod Identity
1. VMSS User Assigned Managed Identity
1. VMSS System Assigned Managed Identity

**OPTION 1 - Service Principal**

> Supported with linux and windows

1. Add your service principal credentials as a Kubernetes secrets accessible by the Secrets Store CSI driver.

    ```bash
    kubectl create secret generic secrets-store-creds --from-literal clientid=<CLIENTID> --from-literal clientsecret=<CLIENTSECRET>
    ```

    Ensure this service principal has all the required permissions to access content in your Azure key vault instance.
    If not, you can run the following using the Azure cli:

    ```bash
    # Assign Reader Role to the service principal for your keyvault
    az role assignment create --role Reader --assignee <principalid> --scope /subscriptions/<subscriptionid>/resourcegroups/<resourcegroup>/providers/Microsoft.KeyVault/vaults/<keyvaultname>

    az keyvault set-policy -n $KV_NAME --key-permissions get --spn <YOUR SPN CLIENT ID>
    az keyvault set-policy -n $KV_NAME --secret-permissions get --spn <YOUR SPN CLIENT ID>
    az keyvault set-policy -n $KV_NAME --certificate-permissions get --spn <YOUR SPN CLIENT ID>
    ```

1. Update your [linux deployment yaml](examples/nginx-pod-secrets-store-inline-volume-secretproviderclass.yaml) or [windows deployment yaml](examples/windows-pod-secrets-store-inline-volume-secretproviderclass.yaml) to reference the service principal kubernetes secret created in the previous step

    ```yaml
    nodePublishSecretRef:
      name: secrets-store-creds
    ```

**OPTION 2 - Pod Identity**

> Supported only on linux

**Prerequisites**

💡 Make sure you have installed pod identity to your Kubernetes cluster

   __This project makes use of the aad-pod-identity project located  [here](https://github.com/Azure/aad-pod-identity#deploy-the-azure-aad-identity-infra) to handle the identity management of the pods. Reference the aad-pod-identity README if you need further instructions on any of these steps.__

Not all steps need to be followed on the instructions for the aad-pod-identity project as we will also complete some of the steps on our installation here.

1. Install the aad-pod-identity components to your cluster

   - Install the RBAC enabled aad-pod-identiy infrastructure components:
      ```
      kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
      ```

   - (Optional) Providing required permissions for MIC

     - If the SPN you are using for the AKS cluster was created separately (before the cluster creation - i.e. not part of the MC_ resource group) you will need to assign it the "Managed Identity Operator" role.
       ```
       az role assignment create --role "Managed Identity Operator" --assignee <sp id> --scope <full id of the managed identity>
       ```

1. Create an Azure User Identity

    Create an Azure User Identity with the following command.
    Get `clientId` and `id` from the output.
    ```
    az identity create -g <resourcegroup> -n <idname>
    ```

1. Assign permissions to new identity
    Ensure your Azure user identity has all the required permissions to read the keyvault instance and to access content within your key vault instance.
    If not, you can run the following using the Azure cli:

    ```bash
    # Assign Reader Role to new Identity for your keyvault
    az role assignment create --role Reader --assignee <principalid> --scope /subscriptions/<subscriptionid>/resourcegroups/<resourcegroup>/providers/Microsoft.KeyVault/vaults/<keyvaultname>

    # set policy to access keys in your keyvault
    az keyvault set-policy -n $KV_NAME --key-permissions get --spn <YOUR AZURE USER IDENTITY CLIENT ID>
    # set policy to access secrets in your keyvault
    az keyvault set-policy -n $KV_NAME --secret-permissions get --spn <YOUR AZURE USER IDENTITY CLIENT ID>
    # set policy to access certs in your keyvault
    az keyvault set-policy -n $KV_NAME --certificate-permissions get --spn <YOUR AZURE USER IDENTITY CLIENT ID>
    ```

1. Add a new `AzureIdentity` for the new identity to your cluster

    Edit and save this as `aadpodidentity.yaml`

    Set `type: 0` for Managed Service Identity; `type: 1` for Service Principal
    In this case, we are using managed service identity, `type: 0`.
    Create a new name for the AzureIdentity.
    Set `resourceID` to `id` of the Azure User Identity created from the previous step.

    ```yaml
    apiVersion: "aadpodidentity.k8s.io/v1"
    kind: AzureIdentity
    metadata:
      name: <any-name>
    spec:
      type: 0
      resourceID: /subscriptions/<subid>/resourcegroups/<resourcegroup>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<idname>
      clientID: <clientid>
    ```

    ```bash
    kubectl create -f aadpodidentity.yaml
    ```

1. Add a new `AzureIdentityBinding` for the new Azure identity to your cluster

    Edit and save this as `aadpodidentitybinding.yaml`
    ```yaml
    apiVersion: "aadpodidentity.k8s.io/v1"
    kind: AzureIdentityBinding
    metadata:
      name: <any-name>
    spec:
      azureIdentity: <name of AzureIdentity created from previous step>
      selector: <label value to match in your app>
    ```

    ```
    kubectl create -f aadpodidentitybinding.yaml
    ```

1. Add the following to [this](examples/nginx-pod-secrets-store-inline-volume-secretproviderclass-podid.yaml) deployment yaml:

    a. Include the `aadpodidbinding` label matching the `selector` value set in the previous step so that this pod will be assigned an identity
    ```yaml
    metadata:
    labels:
      aadpodidbinding: <AzureIdentityBinding Selector created from previous step>
    ```

    b. make sure to update `usepodidentity` to `true`
    ```yaml
    usepodidentity: "true"
    ```

1. Update [this sample deployment](examples/v1alpha1_secretproviderclass_podid.yaml) to create a `secretproviderclasses` resource with `usePodIdentity: "true"` to provide Azure-specific parameters for the Secrets Store CSI driver.

1. Deploy your app

    ```bash
    kubectl apply -f examples/nginx-pod-secrets-store-inline-volume-secretproviderclass-podid.yaml
    ```

1. Validate the pod has access to the secret from key vault:

    ```bash
    kubectl exec -it nginx-secrets-store-inline-podid ls /mnt/secrets-store/
    secret1
    ```

**OPTION 3 - VMSS User Assigned Managed Identity**

> Supported with linux and windows

This option allows azure KeyVault to use the user assigned managed identity on the k8s cluster VMSS directly.

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
   az keyvault set-policy -n $KV_NAME --key-permissions get --spn <YOUR AZURE MANAGED IDENTITY CLIENT ID>
   # set policy to access secrets in your Key Vault
   az keyvault set-policy -n $KV_NAME --secret-permissions get --spn <YOUR AZURE MANAGED IDENTITY CLIENT ID>
   # set policy to access certs in your Key Vault
   az keyvault set-policy -n $KV_NAME --certificate-permissions get --spn <YOUR AZURE MANAGED IDENTITY CLIENT ID>
   ```

4. Deploy your application. Specify `useVMManagedIdentity` to `true` and provide `userAssignedIdentityID`.

```yaml
useVMManagedIdentity: "true"               # [OPTIONAL available for version > 0.0.4] if not provided, will default to "false"
userAssignedIdentityID: "clientid"      # [OPTIONAL available for version > 0.0.4] use the client id to specify which user assigned managed identity to use. If using a user assigned identity as the VM's managed identity, then specify the identity's client id. If empty, then defaults to use the system assigned identity on the VM
```

**OPTION 4 - VMSS System Assigned Managed Identity**

> Supported with linux and windows

This option allows azure KeyVault to use the system assigned managed identity on the k8s cluster VMSS directly.

1. Verify that the nodes have its own system assigned managed identity

```bash
az vmss identity show -g <resource group>  -n <vmss scalset name> -o yaml
```

The output should contain `type: SystemAssigned`.  

2. Grant Azure Managed Identity KeyVault permissions

   Ensure that your Azure Identity has the role assignments required to see your Key Vault instance and to access its content. Run the following Azure CLI commands to assign these roles if needed:

   ```bash
   # set policy to access keys in your Key Vault
   az keyvault set-policy -n $KV_NAME --key-permissions get --spn <YOUR AZURE MANAGED IDENTITY CLIENT ID>
   # set policy to access secrets in your Key Vault
   az keyvault set-policy -n $KV_NAME --secret-permissions get --spn <YOUR AZURE MANAGED IDENTITY CLIENT ID>
   # set policy to access certs in your Key Vault
   az keyvault set-policy -n $KV_NAME --certificate-permissions get --spn <YOUR AZURE MANAGED IDENTITY CLIENT ID>
   ```

3. Deploy your application. Specify `useVMManagedIdentity` to `true`.

```yaml
useVMManagedIdentity: "true"            # [OPTIONAL available for version > 0.0.4] if not provided, will default to "false"
```

**NOTE** When using the `Pod Identity` option mode, there can be some amount of delay in obtaining the objects from keyvault. During the pod creation time, in this particular mode `aad-pod-identity` will need to create the `AzureAssignedIdentity` for the pod based on the `AzureIdentity` and `AzureIdentityBinding`, retrieve token for keyvault. This process can take time to complete and it's possible for the pod volume mount to fail during this time. When the volume mount fails, kubelet will keep retrying until it succeeds. So the volume mount will eventually succeed after the whole process for retrieving the token is complete.

## Local End-To-End Testing for the Key Vault Azure Provider

This section will show you how to locally test the Azure Key Vault Provider end-to-end (e2e). The e2e tests utilize Bats for testing the scripts. Take a look inside the `test/bats` folder to see the tests and the deployments needed for creating the e2e tests.

### E2E Prerequisites

- [Helm 3.x](https://helm.sh/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [Azure Service Principal](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli?view=azure-cli-latest)
- [Azure Key Vault with objects stored inside](https://docs.microsoft.com/en-us/azure/key-vault/key-vault-manage-with-cli2)
- [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [Docker](https://docs.docker.com/get-started/)
- [Bats](https://github.com/bats-core/bats-core)

As as prerequisite, you will need to have an [Azure Key Vault](https://docs.microsoft.com/en-us/azure/key-vault/key-vault-manage-with-cli2) with objects stored and an [Azure Service Principal](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli?view=azure-cli-latest). You can follow the next few steps to set up both.

### Set up Azure Key Vault

The [Azure Key Vault](https://docs.microsoft.com/azure/key-vault/) will be used to store your secrets. Run the following to create a Resource Group, and a uniquely named Azure Key Vault:

```bash
  KEYVAULT_RESOURCE_GROUP=<keyvault-resource-group>
  KEYVAULT_LOCATION=<keyvault-location>
  KEYVAULT_NAME=secret-store-$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 10 | head -n 1)

  az group create -n $KEYVAULT_RESOURCE_GROUP --location $KEYVAULT_LOCATION
  az keyvault create -n $KEYVAULT_NAME -g $KEYVAULT_RESOURCE_GROUP --location $KEYVAULT_LOCATION
```
 If you were to check in your Azure Subscription, you will have a new Resource Group and Key Vault created. You can learn more [here](https://docs.microsoft.com/azure/key-vault/about-keys-secrets-and-certificates#objects-identifiers-and-versioning) about Azure Key Vault naming, objects, types, and versioning.

You can add objects to your Key Vault with the commands below (make sure to add your own object name):

```bash
az keyvault secret set --vault-name $KEYVAULT_NAME --name <secretNameHere> --value $(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 40 | head -n 1)
az keyvault key create --vault-name $KEYVAULT_NAME --name <keyNameHere>
```

You can retrieve the value of a Key Vault secret with the following script:

```bash
# to view a given secret's value
az keyvault secret show --vault-name $KEYVAULT_NAME --name <secretNameHere> --query value -o tsv
```
### Create a Service Principal

We now need to create a [Service Principal](https://docs.microsoft.com/en-us/azure/active-directory/develop/app-objects-and-service-principals#service-principal-object) with **Read Only** access to our Azure Key Vault. The Azure Provider will use this Service Principal to access our secrets from our Key Vault.

> You will need to keep track of `AZURE_CLIENT_ID` and `AZURE_CLIENT_SECRET`, as you will need to place these into the upcoming `secrets.env`.

```bash
KEYVAULT_RESOURCE_ID=$(az keyvault show -n $KEYVAULT_NAME --query id -o tsv)
AZURE_CLIENT_ID=$(az ad sp create-for-rbac --name $KEYVAULT_NAME --role Reader --scopes $KEYVAULT_RESOURCE_ID --query appId -o tsv)
AZURE_CLIENT_SECRET=$(az ad sp credential reset --name $AZURE_CLIENT_ID --credential-description "APClientSecret" --query password -o tsv)

# Assign Read Only Policy for our Key Vault to the Service Principal
az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --spn $KEYVAULT_CLIENT_ID
```
The Service Principal(SP) created during this section uses just **Read Only** permissions. This SP is then applied to the Azure Key Vault since we want to limit the Service Principal's credentials to only allow for reading of the keys. This will prevent the chance of manipulating anything on the Key Vault when using the login of this Service Principal.

### Preparing your secrets

Add your secrets to a `secrets.env` file at the application `root` directory.

1. Add all secrets related to the Azure Key Vault, Service Principal, and your Azure Subscription

    💡 The third Key Vault object information is the same as the first  object. Only, the 3rd object will also include an `objectAlias`.

    ```bash
    # secrets.env

    KEYVAULT_SECRET_NAME=<yourKeyVaultSecretName>
    KEYVAULT_SECRET_TYPE=secret
    KEYVAULT_SECRET_ALIAS=""
    KEYVAULT_SECRET_VERSION=""

    KEYVAULT_KEY_NAME=<yourKeyVaultKeyName>
    KEYVAULT_KEY_VALUE=<yourKeyVaultKeyValue>
    KEYVAULT_KEY_TYPE=key
    KEYVAULT_KEY_ALIAS=<YOUR_KEY_VAULT_KEY_ALIAS>
    KEYVAULT_KEY_VERSION=""

    KEYVAULT_NAME=<yourAzureKeyVaultName>

    AZURE_CLIENT_ID=<yourAzureServicePrincipalId>
    AZURE_CLIENT_SECRET=<yourAzureServicePrincipalSecret>
    TENANT_ID=<yourAzureTenantId>
    ```

2. We'll add the necssary environment variables needed inside the `Makefile` and `azure.bats` .

    ```bash
    # secrets.env

    # ...

    # the name you want to give for the docker image
    DOCKER_IMAGE=e2e/secrets-store-csi-driver-provider-azure
    # name of docker image provided for the azure.bats tests. SHOULD be the same as DOCKER_IMAGE
    PROVIDER_TEST_IMAGE=e2e/secrets-store-csi-driver-provider-azure
    # local path to folder container cloned Secrets Store CSI Driver
    SECRETS_STORE_CSI_DRIVER_PATH=<your_local_path_to_the_secrets_store_csi_driver>

    ```
<details>
  <summary>The finished 'secrets.env' should look like this:</summary>
  <p>

    KEYVAULT_SECRET_NAME=<yourKeyVaultSecretName>
    KEYVAULT_SECRET_TYPE=secret
    KEYVAULT_SECRET_ALIAS=""
    KEYVAULT_SECRET_VERSION=""

    KEYVAULT_KEY_NAME=<yourKeyVaultKeyName>
    KEYVAULT_KEY_VALUE=<yourKeyVaultKeyValue>
    KEYVAULT_KEY_TYPE=key
    KEYVAULT_KEY_ALIAS=<YOUR_KEY_VAULT_KEY_ALIAS>
    KEYVAULT_KEY_VERSION=""

    KEYVAULT_NAME=<yourAzureKeyVaultName>

    AZURE_CLIENT_ID=<yourAzureServicePrincipalId>
    AZURE_CLIENT_SECRET=<yourAzureServicePrincipalSecret>
    TENANT_ID=<yourAzureTenantId>

    DOCKER_IMAGE=e2e/secrets-store-csi-driver-provider-azure
    PROVIDER_TEST_IMAGE=e2e/secrets-store-csi-driver-provider-azure
    SECRETS_STORE_CSI_DRIVER_PATH=<your_local_path_to_the_secrets_store_csi_driver>

  </p>
</details>

### Testing the Azure Key Vault Provider

Here are the steps that you can follow to test the Azure Key Vault Azure Provider.

1. Make sure you have covered all of the [prerequisites](#e2e-prerequisites) listed.
2. Now run the following make targets to test the Azure Provider e2e.
    ```bash
      # create and configure kind cluster
      make e2e-bootstrap
      # run the e2e-tests
      make e2e-test
    ```
