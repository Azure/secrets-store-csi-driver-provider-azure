# Demo using KIND cluster

[kind](https://github.com/kubernetes-sigs/kind)(Kubernetes in Docker) is a tool for running local Kubernetes clusters using Docker container “nodes”. Azure Key Vault Provider for Secrets Store CSI Driver will work in kind using Service Principal.

## Prerequisite

- Follow [instructions](https://github.com/kubernetes-sigs/kind#installation-and-usage) to set up kind in your machine

> Windows 10 users can use WSL 2 to install kind. Integrate docker for windows with WSL 2 by following the [instructions](https://kind.sigs.k8s.io/docs/user/using-wsl2/).

## Setup

- Follow the [instructions](https://azure.github.io/secrets-store-csi-driver-provider-azure/configurations/identity-access-modes/service-principal-mode/) to set up Service Principal and give it access to Azure Key Vault. Keep `ClientID` and `ClientSecret` of the Service Principal handy.

- Copy [v1alpha1_secretproviderclass.yaml](https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/examples/service-principal/v1alpha1_secretproviderclass_service_principal.yaml) and [pod-secrets-store-inline-volume-secretproviderclass.yaml](https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/examples/service-principal/pod-secrets-store-inline-volume-secretproviderclass.yaml) to this directory.

- Update `v1alpha1_secretproviderclass.yaml` to provide keyvault name and keyvault resources to fetch.

```yaml
cloudName: 'AzurePublicCloud' # [OPTIONAL available for version > 0.0.4] if not provided, azure environment will default to AzurePublicCloud
keyvaultName: ''              # the name of the KeyVault
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
tenantID: '<tenant id>'       # the tenant ID of the KeyVault
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

- Install [csi-secrets-store-provider-azure](https://azure.github.io/secrets-store-csi-driver-provider-azure/getting-started/installation/)

- Add your Service Principal credentials as a Kubernetes secrets accessible by the Secrets Store CSI driver.

```sh
kubectl create secret generic secrets-store-creds --from-literal clientid=<CLIENTID> --from-literal clientsecret=<CLIENTSECRET>
kubectl label secret secrets-store-creds secrets-store.csi.k8s.io/used=true
```

- Deploy the app. This will deploy a busybox container and mount the secrets as volume at path `/mnt/secrets-store`

```sh
kubectl apply -f pod-secrets-store-inline-volume.yaml
```

### Validate the secret

Run the below command to list the secrets pulled from keyvault. Each of the file contains the value of the secret.

```sh
kubectl exec busybox-secrets-store-inline ls /mnt/secrets-store/
```
