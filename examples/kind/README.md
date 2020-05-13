# KIND Demo

[kind](https://github.com/kubernetes-sigs/kind)(Kubernetes in Docker) is a tool for running local Kubernetes clusters using Docker container “nodes”. Azure Key Vault Provider for Secrets Store CSI Driver will work in kind using Service Principal.

## Prerequisite

- Follow [kind installation instruction](https://github.com/kubernetes-sigs/kind#installation-and-usage) to setup kind in your machine

> Windows 10 users could you wsl 2 to install kind and run this sample. Integrate docker for windows with wsl 2 by following the [instructions](https://docs.docker.com/docker-for-windows/wsl-tech-preview/).

## Setup

- We will use Azure Key Vault provider for Secrets Store CSI driver in [Service Principle Mode](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/docs/service-principal-mode.md) Create a Azure AD App to create Service Principal and give it "GET" permission for secrets in keyvault. Follow the steps in [keyvault docs](https://docs.microsoft.com/en-us/azure/key-vault/general/group-permissions-for-apps#applications). Keep `ClientID` and `ClientSecret` of the AD app handy.

- Copy [v1alpha1_secretproviderclass.yaml](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/examples/v1alpha1_secretproviderclass.yaml) and [nginx-pod-secrets-store-inline-volume-secretproviderclass.yaml](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/examples/nginx-pod-secrets-store-inline-volume-secretproviderclass.yaml) to this directory.

- Update `v1alpha1_secretproviderclass.yaml` to provide keyvault name and the keyvault resources to fetch.

```yaml
cloudName: 'AzurePublicCloud' # [OPTIONAL available for version > 0.0.4] if not provided, azure environment will default to AzurePublicCloud
keyvaultName: '' # the name of the KeyVault
objects: |
  array:
    - |
    objectName: secret1
    objectType: secret        # object types: secret, key or cert
    objectVersion: ""         # [OPTIONAL] object versions, default to latest if empty
    - |
    objectName: key1
    objectType: key
    objectVersion: ""
resourceGroup: '' # [REQUIRED for version < 0.0.4] the resource group of the KeyVault
subscriptionId: '' # [REQUIRED for version < 0.0.4] the subscription ID of the KeyVault
tenantId: '' # the tenant ID of the KeyVault
```

## Usage

Run the `kind-demo.sh` from this directory and pass AD App's `Client_ID` and `Client_Secret` as argument.

```sh
./kind-demo.sh <Client_ID> <Client_Secret>
```

The final output would contain the list of keys and secrets pulled from the keyvault as files in the directory `/mnt/secrets-store`

## Step by Step instruction

- Create a kind cluster

```sh
kind create cluster --name kind-csi-demo
```

- Install [csi-secrets-store-provider-azure](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/charts/csi-secrets-store-provider-azure/README.md)

- Add your AD app credentials as a Kubernetes secrets accessible by the Secrets Store CSI driver.

```sh
kubectl create secret generic secrets-store-creds --from-literal clientid=<CLIENTID> --from-literal clientsecret=<CLIENTSECRET>
```

- Deploy the app. This will deploy a nginx container and mount the secrets as volumne at path `/mnt/secrets-store`

```sh
kubectl apply -f nginx-pod-secrets-store-inline-volume.yaml.yaml
```

### Validate CSI Driver

Run the below command and it should list the secrets pulled from keyvault. Each of the file contains the value of the secret.

```sh
kubectl exec -it nginx-secrets-store-inline ls /mnt/secrets-store/
```
