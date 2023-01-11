---
type: docs
title: "Workload Identity"
linkTitle: "Workload Identity"
weight: 1
description: >
  Use Workload Identity to access Keyvault.
---

<details>
<summary>Examples</summary>

- `SecretProviderClass`
```yaml
# This is a SecretProviderClass example using workload identity to access Key Vault
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: azure-kvname-wi
spec:
  provider: azure
  parameters:
    usePodIdentity: "false"         # set to true for pod identity access mode
    clientID: "<client id of the Azure AD Application or user-assigned managed identity to use for workload identity>"
    keyvaultName: "kvname"
    cloudName: ""                   # [OPTIONAL for Azure] if not provided, azure environment will default to AzurePublicCloud
    objects:  |
      array:
        - |
          objectName: secret1
          objectType: secret        # object types: secret, key or cert
          objectVersion: ""         # [OPTIONAL] object versions, default to latest if empty
        - |
          objectName: key1
          objectType: key
          objectVersion: ""
    tenantID: "tid"                    # the tenant ID of the KeyVault  
```

- `Pod` yaml
```yaml
# This is a sample pod definition for using SecretProviderClass and workload identity to access Key Vault
kind: Pod
apiVersion: v1
metadata:
  name: busybox-secrets-store-inline-wi
spec:
  containers:
    - name: busybox
      image: k8s.gcr.io/e2e-test-images/busybox:1.29
      command:
        - "/bin/sleep"
        - "10000"
      volumeMounts:
      - name: secrets-store01-inline
        mountPath: "/mnt/secrets-store"
        readOnly: true
  volumes:
    - name: secrets-store01-inline
      csi:
        driver: secrets-store.csi.k8s.io
        readOnly: true
        volumeAttributes:
          secretProviderClass: "azure-kvname-wi"
```
</details>

## Prerequisites

- [AKS Kubernetes 1.21+ cluster with OIDC Issuer](https://learn.microsoft.com/en-us/azure/aks/cluster-configuration#oidc-issuer)
  - The workload identity feature is implemented using the [Token Requests](https://kubernetes-csi.github.io/docs/token-requests.html) API in CSI driver. This is available by default in Kubernetes 1.21+.
- Secrets Store CSI Driver v1.1.0+
- Azure Key Vault Provider v1.1.0+
- Azure CLI version `2.40.0` or higher. Run `az --version` to verify.

## Configure Workload Identity to access Keyvault

### 1. Create an Azure AD Application or user-assigned managed identity

> [Workload Identity Federation](https://docs.microsoft.com/en-us/azure/active-directory/develop/workload-identity-federation) is supported for Azure AD Applications and user-assigned managed identity.

```bash
# environment variables for the AAD application
export APPLICATION_NAME="<your application name>"
az ad sp create-for-rbac --name "${APPLICATION_NAME}"
export APPLICATION_CLIENT_ID=$(az ad sp list --display-name ${APPLICATION_NAME} --query '[0].appId' -otsv)
```

> NOTE: `az ad sp create-for-rbac` will create a new application with secret. However, the secret is not required for workload identity federation.

If you want to use a user-assigned managed identity, you can create one using the following command:

```bash
export RESOURCE_GROUP=<resource group name>
export USER_ASSIGNED_IDENTITY_NAME="<your user-assigned managed identity name>"
az identity create -g ${RESOURCE_GROUP} -n ${USER_ASSIGNED_IDENTITY_NAME}
export USER_ASSIGNED_IDENTITY_CLIENT_ID=$(az identity show -g ${RESOURCE_GROUP} -n ${USER_ASSIGNED_IDENTITY_NAME} --query clientId -otsv)
```

### 2. Grant workload identity permission to access Keyvault

 Ensure that your Azure AD Application or user-assigned managed identity has the role assignments required to access content in keyvault instance. Run the following Azure CLI commands to assign the roles if required:

 ```bash
 # set policy to access keys in your Keyvault
 az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --spn ${APPLICATION_CLIENT_ID:-$USER_ASSIGNED_IDENTITY_CLIENT_ID}
 # set policy to access secrets in your Keyvault
 az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --spn ${APPLICATION_CLIENT_ID:-$USER_ASSIGNED_IDENTITY_CLIENT_ID}
 # set policy to access certs in your Keyvault
 az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --spn ${APPLICATION_CLIENT_ID:-$USER_ASSIGNED_IDENTITY_CLIENT_ID}
 ```

### 3. Establish federated identity credential between the workload identity and the service account issuer & subject

Get the AKS cluster OIDC issuer URL:

```bash
# Output the OIDC issuer URL
az aks show --resource-group <resource_group> --name <cluster_name> --query "oidcIssuerProfile.issuerUrl" -otsv
```

> If the URL is empty, ensure the oidc issuer is enabled in the cluster by following these [steps](https://learn.microsoft.com/en-us/azure/aks/cluster-configuration#oidc-issuer).
> If using other managed cluster, refer to [doc](https://azure.github.io/azure-workload-identity/docs/installation/managed-clusters.html) for retrieving the OIDC issuer URL.

#### Using Azure CLI

```bash
export SERVICE_ACCOUNT_NAME=<name of the service account used by the application pod (pod requesting the volume mount)>
export SERVICE_ACCOUNT_NAMESPACE=<namespace of the service account>
```

##### Using Azure AD Application

```bash
# Get the object ID of the AAD application
export APPLICATION_OBJECT_ID="$(az ad app show --id ${APPLICATION_CLIENT_ID} --query id -otsv)"
```

Add the federated identity credential:

```json
cat <<EOF > params.json
{
  "name": "kubernetes-federated-credential",
  "issuer": "${SERVICE_ACCOUNT_ISSUER}",
  "subject": "system:serviceaccount:${SERVICE_ACCOUNT_NAMESPACE}:${SERVICE_ACCOUNT_NAME}",
  "description": "Kubernetes service account federated credential",
  "audiences": [
    "api://AzureADTokenExchange"
  ]
}
EOF
```

```bash
az ad app federated-credential create --id "${APPLICATION_OBJECT_ID}" --parameters @params.json
```

##### Using user-assigned managed identity

Add the federated identity credential:

```bash
az identity federated-credential create \
  --name "kubernetes-federated-credential" \
  --identity-name "${USER_ASSIGNED_IDENTITY_NAME}" \
  --resource-group "${RESOURCE_GROUP}" \
  --issuer "${SERVICE_ACCOUNT_ISSUER}" \
  --subject "system:serviceaccount:${SERVICE_ACCOUNT_NAMESPACE}:${SERVICE_ACCOUNT_NAME}"
```

### 4. Deploy your secretproviderclass and application

Set the `clientID` in the `SecretProviderClass` to the client ID of the AAD application or user-assigned managed identity.

```yaml
clientID: "${APPLICATION_OR_MANAGED_IDENTITY_CLIENT_ID}"
```

## Pros

1. Supported on both Windows and Linux.
2. Supports Kubernetes clusters hosted in any cloud or on-premises.
