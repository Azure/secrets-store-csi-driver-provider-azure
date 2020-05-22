# Demo using KIND cluster

[kind](https://github.com/kubernetes-sigs/kind)(Kubernetes in Docker) is a tool for running local Kubernetes clusters using Docker container “nodes”. Azure Key Vault Provider for Secrets Store CSI Driver will work in kind using Service Principal.

## Prerequisite

- Follow [instructions](https://github.com/kubernetes-sigs/kind#installation-and-usage) to setup kind in your machine

> Windows 10 users can use WSL 2 to install kind. Integrate docker for windows with WSL 2 by following the [instructions](https://kind.sigs.k8s.io/docs/user/using-wsl2/)..

## Setup

- Follow the [instructions](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/docs/service-principal-mode.md) to setup Service Principal and give it access to Azure Key Vault. Keep `ClientID` and `ClientSecret` of the Service Principal handy.

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

## Demo

- Create a kind cluster

```sh
kind create cluster --name kind-csi-demo
```

- Install [csi-secrets-store-provider-azure](https://github.com/Azure/secrets-store-csi-driver-provider-azure#install-the-secrets-store-csi-driver-and-the-azure-keyvault-provider)

- Add your Service Principal credentials as a Kubernetes secrets accessible by the Secrets Store CSI driver.

```sh
kubectl create secret generic secrets-store-creds --from-literal clientid=<CLIENTID> --from-literal clientsecret=<CLIENTSECRET>
```

- Deploy the app. This will deploy a nginx container and mount the secrets as volume at path `/mnt/secrets-store`

```sh
kubectl apply -f nginx-pod-secrets-store-inline-volume.yaml
```

### Validate the secret

Run the below command and it should list the secrets pulled from keyvault. Each of the file contains the value of the secret.

```sh
kubectl exec -it nginx-secrets-store-inline ls /mnt/secrets-store/
```
