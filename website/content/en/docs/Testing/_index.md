---
title: "Testing"
linkTitle: "Testing"
weight: 5
description: >
  This section will show you how to locally test the Azure Key Vault Provider end-to-end (e2e)
---

## Local End-To-End Testing for the Azure Key Vault Provider

This section will show you how to locally test the Azure Key Vault Provider end-to-end (e2e). The e2e tests utilize Bats for testing the scripts. Take a look inside the [test/bats](/test/bats) folder to see the tests and the deployments needed for creating the e2e tests.

### E2E Prerequisites

- [Helm 3.x](https://helm.sh/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [Azure Service Principal](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli?view=azure-cli-latest)
- [Azure Key Vault with objects stored inside](https://docs.microsoft.com/en-us/azure/key-vault/key-vault-manage-with-cli2)
- [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [Docker](https://docs.docker.com/get-started/)
- [Bats](https://github.com/bats-core/bats-core)

As as prerequisite, you will need to have an [Azure Key Vault](https://docs.microsoft.com/en-us/azure/key-vault/key-vault-manage-with-cli2) with objects stored and an [Azure Service Principal](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli?view=azure-cli-latest). You can follow the next few steps to set up both.

### Set up an Azure Key Vault

For assistance on setting up an Azure Key Vault specific to testing this project, please refer to [this guide](/docs/setup-keyvault.md)

### Assign a Service Principal to Your Azure Key Vault

For assistance on assigning an existing or new Service Principal to your Key Vault, please follow [this guide](/docs/service-principal-mode.md).

### Preparing your secrets

This step will make sure that you have all the necessary environment variables prepared to test this project. Now, add your environment variables to a `secrets.env` file at the application `root` directory.

1. Add all secrets related to the Azure Key Vault, Service Principal, and your Azure Subscription.

> By this point you should have 4 Objects in your Key Vault. 2 Certs, and 2 Objects(Secrets and/or Keys).

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

  CERT1_NAME=<yourKeyVaultCertName>
  CERT1_VALUE=<yourKeyVaultCertValue>
  CERT1_VERSION=""

  CERT2_NAME=<yourKeyVaultCertName>
  CERT2_VALUE=<yourKeyVaultCertValue>
  CERT2_VERSION=""

  KEYVAULT_NAME=<yourAzureKeyVaultName>

  AZURE_CLIENT_ID=<yourAzureServicePrincipalId>
  AZURE_CLIENT_SECRET=<yourAzureServicePrincipalSecret>
  TENANT_ID=<yourAzureTenantId>
```

2. We'll add the necssary environment variables needed inside the `Makefile` and `azure.bats`.

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
    CERT1_VALUE=<yourKeyVaultCertValue>
    CERT1_VERSION=""

    CERT2_NAME=<yourKeyVaultCertName>
    CERT2_VALUE=<yourKeyVaultCertValue>
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
  # set your cluster context to the kind cluster
  export KUBECONFIG=$(kind get kubeconfig-path)
  # add the dev namespace where the Secrets Store CSI Driver and the Azure Provider will be deployed
  kubectl create namespace dev
  # run the e2e-tests
  make e2e-test
```
