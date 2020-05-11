# Testing

## Local End-To-End Testing for the Azure Key Vault Provider

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

Below are examples of how to quickly add a few objects to your Key Vault (make sure to add your own object name):

💡 **Please add 2 Objects (Keys and/or Secrets) and 2 Certificates (1 PEM, and 1 PKCS12) to your Key Vault.**

```bash
az keyvault secret set --vault-name $KEYVAULT_NAME --name <secretNameHere> --value $(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 40 | head -n 1)

az keyvault key create --vault-name $KEYVAULT_NAME --name <keyNameHere>

az keyvault certificate create --vault-name $KEYVAULT_NAME --name <certNameHere> -p "$(az keyvault certificate get-default-policy)"
```

You can retrieve the value of a Key Vault secret with the following script:

```bash
# to view a given secret's value
az keyvault secret show --vault-name $KEYVAULT_NAME --name <secretNameHere> --query value -o tsv
# to view a given key's value
az keyvault key show --vault-name $KEYVAULT_NAME --name <yourKeyNameHere> --query "key.n" -o tsv
# to view a given certificate's value
az keyvault certificate show --vault-name $KEYVAULT_NAME --name <yourCertificateNameHere> --query cer -o tsv
```
### Create a Service Principal

We now need to create a [Service Principal](https://docs.microsoft.com/en-us/azure/active-directory/develop/app-objects-and-service-principals#service-principal-object) with **Read Only** access to our Azure Key Vault. The Azure Provider will use this Service Principal to access our secrets from our Key Vault.

> You will need to keep track of `AZURE_CLIENT_ID` and `AZURE_CLIENT_SECRET`, as you will need to place these into the upcoming `secrets.env`.

```bash
KEYVAULT_RESOURCE_ID=$(az keyvault show -n $KEYVAULT_NAME --query id -o tsv)
AZURE_CLIENT_ID=$(az ad sp create-for-rbac --name $KEYVAULT_NAME --role Reader --scopes $KEYVAULT_RESOURCE_ID --query appId -o tsv)
AZURE_CLIENT_SECRET=$(az ad sp credential reset --name $AZURE_CLIENT_ID --credential-description "APClientSecret" --query password -o tsv)

# Assign Read Only Policy for our Key Vault to the Service Principal
az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --spn $AZURE_CLIENT_ID
az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --spn $AZURE_CLIENT_ID
az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --spn $AZURE_CLIENT_ID

```
The Service Principal(SP) created during this section uses just **Read Only** permissions. This SP is then applied to the Azure Key Vault since we want to limit the Service Principal's credentials to only allow for reading of the keys. This will prevent the chance of manipulating anything on the Key Vault when using the login of this Service Principal.

### Preparing your secrets

Add your secrets to a `secrets.env` file at the application `root` directory.

1. Add all secrets related to the Azure Key Vault, Service Principal, and your Azure Subscription.

> By this point you should have 4 Objects in your Key Vault. 2 Certs (Base64 Encoded), and 2 Objects (Secrets and/or Keys).

    ```bash
    # secrets.env

    OBJECT1_NAME=<yourKeyVaultSecretName>
    OBJECT1_TYPE=secret
    OBJECT1_ALIAS=""
    OBJECT1_VERSION=""

    OBJECT2_NAME=<yourKeyVaultKeyName>
    OBJECT2_VALUE=<yourKeyVaultKeyValue>
    OBJECT2_TYPE=key
    OBJECT2_ALIAS=<YOUR_KEY_VAULT_KEY_ALIAS>
    OBJECT2_VERSION=""
     
    # The Certs are Base64 Encoded.
    CERT1_NAME=<yourKeyVaultCertName>
    CERT1_VALUE=<yourKeyVaultCertBase64EncodedValue>
    CERT1_VERSION=""

    CERT2_NAME=<yourKeyVaultCertName>
    CERT2_VALUE=<yourKeyVaultCertBase64EncodedValue>
    CERT2_VERSION=""

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
    # the tag you'd like to give your docker image
    IMAGE_TAG=<image_tag>
    # assign an image version to know which changes you are testing.
    IMAGE_VERSION=<image_version>
    # name of docker image provided for the azure.bats tests. SHOULD be the same as DOCKER_IMAGE
    PROVIDER_TEST_IMAGE=e2e/secrets-store-csi-driver-provider-azure
    # local path to folder container cloned Secrets Store CSI Driver
    SECRETS_STORE_CSI_DRIVER_PATH=<your_local_path_to_the_secrets_store_csi_driver>
    # will disable tests not related to the Kind Cluster
    CI_KIND_CLUSTER=true
    ```
<details>
  <summary>The finished 'secrets.env' should look like this:</summary>
  <p>

    OBJECT1_NAME=<yourKeyVaultSecretName>
    OBJECT1_TYPE=secret
    OBJECT1_ALIAS=""
    OBJECT1_VERSION=""

    OBJECT2_NAME=<yourKeyVaultKeyName>
    OBJECT2_VALUE=<yourKeyVaultKeyValue>
    OBJECT2_TYPE=key
    OBJECT2_ALIAS=<YOUR_KEY_VAULT_KEY_ALIAS>
    OBJECT2_VERSION=""

    CERT1_NAME=<yourKeyVaultCertName>
    CERT1_VALUE=<yourKeyVaultCertBase64EncodedValue>
    CERT1_VERSION=""

    CERT2_NAME=<yourKeyVaultCertName>
    CERT2_VALUE=<yourKeyVaultCertBase64EncodedValue>
    CERT2_VERSION=""

    KEYVAULT_NAME=<yourAzureKeyVaultName>

    AZURE_CLIENT_ID=<yourAzureServicePrincipalId>
    AZURE_CLIENT_SECRET=<yourAzureServicePrincipalSecret>
    TENANT_ID=<yourAzureTenantId>

    DOCKER_IMAGE=e2e/secrets-store-csi-driver-provider-azure
    IMAGE_TAG=<image_tag>
    IMAGE_VERSION=<image_version>
    PROVIDER_TEST_IMAGE=e2e/secrets-store-csi-driver-provider-azure
    SECRETS_STORE_CSI_DRIVER_PATH=<your_local_path_to_the_secrets_store_csi_driver>
    CI_KIND_CLUSTER=true
  </p>
</details>

### Testing the Azure Key Vault Provider

Here are the steps that you can follow to test the Azure Key Vault Azure Provider.

1. Make sure you have covered all of the [prerequisites](#e2e-prerequisites) listed.
2. Now run the following make targets to test the Azure Provider e2e.
    ```bash
      # create and configure kind cluster
      make e2e-local-bootstrap
      # run the e2e-tests
      make e2e-test
    ```
