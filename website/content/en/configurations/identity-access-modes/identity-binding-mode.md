---
type: docs
title: "Identity Binding"
linkTitle: "Identity Binding"
weight: 0
description: >
  Use Identity Binding to access Key Vault on AKS.
---

{{% alert title="AKS Only" color="info" %}}
Identity binding is only available on Azure Kubernetes Service (AKS) clusters.
{{% /alert %}}

Identity binding is an AKS-specific auth mode that eliminates a key limitation of [workload identity](../workload-identity-mode): the **20 federated identity credential (FIC) cap per managed identity**. With workload identity, each combination of namespace, service account, and cluster requires its own FIC since each cluster has a unique issuer, which means a single managed identity can only be used by 20 unique workloads across all clusters. Identity binding requires only a single FIC on the managed identity regardless of how many clusters or workloads use it, so any number of workloads can share a managed identity without per-workload credential management.

For more details on identity binding concepts, see the [AKS identity binding documentation](https://learn.microsoft.com/azure/aks/identity-bindings-concepts).

<details>
<summary>Examples</summary>

- `SecretProviderClass`
```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: azure-kvname-idb
spec:
  provider: azure
  parameters:
    useAzureTokenProxy: "true"
    clientID: "<client id of the user-assigned managed identity>"
    keyvaultName: "kvname"
    objects:  |
      array:
        - |
          objectName: secret1
          objectType: secret
          objectVersion: ""
        - |
          objectName: key1
          objectType: key
          objectVersion: ""
    tenantID: "tid"
```

- `Pod` yaml
```yaml
kind: Pod
apiVersion: v1
metadata:
  name: busybox-secrets-store-inline-idb
spec:
  serviceAccountName: ${SERVICE_ACCOUNT_NAME}
  containers:
    - name: busybox
      image: registry.k8s.io/e2e-test-images/busybox:1.29-4
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
          secretProviderClass: "azure-kvname-idb"
```
</details>

## Prerequisites

- AKS cluster
- Azure CLI version `2.40.0` or higher
- A user-assigned managed identity with access to the Key Vault
- An [AKS identity binding](https://learn.microsoft.com/azure/aks/identity-bindings-concepts) resource created for the managed identity

## Configure Identity Binding to access Key Vault

### 1. Create a managed identity and grant Key Vault access

Create a user-assigned managed identity and grant it access to your Key Vault:

```bash
export RESOURCE_GROUP=<resource group name>
export USER_ASSIGNED_IDENTITY_NAME="<your managed identity name>"
export KEYVAULT_NAME="<your key vault name>"

az identity create -g ${RESOURCE_GROUP} -n ${USER_ASSIGNED_IDENTITY_NAME}
export USER_ASSIGNED_IDENTITY_CLIENT_ID=$(az identity show -g ${RESOURCE_GROUP} -n ${USER_ASSIGNED_IDENTITY_NAME} --query clientId -otsv)

# Grant Key Vault access
az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --spn ${USER_ASSIGNED_IDENTITY_CLIENT_ID}
az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --spn ${USER_ASSIGNED_IDENTITY_CLIENT_ID}
az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --spn ${USER_ASSIGNED_IDENTITY_CLIENT_ID}
```

### 2. Create the identity binding and configure RBAC

Follow the [AKS identity binding documentation](https://learn.microsoft.com/azure/aks/identity-bindings-concepts) to:

1. Create an identity binding resource for your AKS cluster
2. Create the required Kubernetes RBAC (ClusterRole and ClusterRoleBinding) to authorize your service account to use the managed identity

### 3. Deploy your SecretProviderClass and application

Set `useAzureTokenProxy: "true"` and the `clientID` to the managed identity's client ID in the `SecretProviderClass`:

```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: azure-kvname-idb
spec:
  provider: azure
  parameters:
    useAzureTokenProxy: "true"
    clientID: "${USER_ASSIGNED_IDENTITY_CLIENT_ID}"
    keyvaultName: "${KEYVAULT_NAME}"
    tenantID: "${TENANT_ID}"
    objects: |
      array:
        - |
          objectName: secret1
          objectType: secret
```

Deploy a pod with a service account that has the identity binding RBAC configured in step 2.

{{% alert title="Note" color="info" %}}
If you are installing the Secrets Store CSI Driver using raw manifests instead of the Helm chart, ensure the `CSIDriver` resource includes `api://AKSIdentityBinding` in the `tokenRequests` field. The Helm chart includes this by default.
{{% /alert %}}

## Pros

1. **No per-workload FIC**: Only a single FIC on the managed identity is needed regardless of how many clusters or workloads use it, unlike workload identity which requires one per namespace, service account, and cluster combination (capped at 20 per identity).
2. **No per-workload credential management**: New workloads can use a managed identity without creating individual FICs.
3. Supported on both Windows and Linux.
