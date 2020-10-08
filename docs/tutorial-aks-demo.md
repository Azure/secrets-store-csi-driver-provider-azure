# Secure Secrets in AKS with Key Vault

## Introduction

A Secret in Kubernetes is meant to save sensitive and secure data like passwords, certificates, and keys. These Secret objects aren't secret or secure! They're encoded using Base64 and saved into *etcd*. Also, by default, any pod have access to them. 

To secure these secrets, one solution could be to encrypt data at rest in Kubernetes as explained in [Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data). But this feature (KMS) is not yet supported in AKS. However some work is in progress. Check the [AKS Roadmap](https://github.com/Azure/AKS/projects/1#card-36434441) and the [KMS Plugin for Key Vault](https://github.com/Azure/kubernetes-kms).

Another solution would be using Azure Key Vault. It can securely encrypt sensitive data like keys, secrets, files, and certificates. Even more than just encryption, they offer powerful features like key rotation, expiration date and access policies.

Now, to access Key Vault, a password is needed. Thus it will be non-securely saved in a Secret object in *etcd*! We returned back to the first problem. Fortunately in Azure, AKS can connect to Key Vault or SQL Database using an Identity. This connection could be done using the open-source [AAD Pod Identity](https://github.com/Azure/aad-pod-identity) project. It uses Azure AD to create an Identity and assign the roles and resources.

Now that we have access to Key Vault, we can use its SDK or REST API in the application to retrieve the secrets. The SDK has support for .NET, Java, Python, JS, Ruby, PHP, etc. Or we can retrieve the secrets from a mounted volume. Historically, in Azure, this solution was implemented through [Kubernetes Key Vault Flex Volume](https://github.com/Azure/kubernetes-keyvault-flexvol). Now it's being deprecated. The new solution is [Azure Key Vault provider for Secret Store CSI driver](https://github.com/Azure/secrets-store-csi-driver-provider-azure). Which is the Azure implementation of [Secrets Store CSI driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver).

This tutorial will help you to securely retrieve secrets in Key Vault right from the Pod using Secrets Store CSI and AAD Pod Identity.

## Setting up the environment

To complete this workshop, we need az, kubectl and helm CLI. Also, we need to create the following resources in Azure:
•	AKS cluster that uses Service Principal or Managed Identity through the variable $isAKSWithManagedIdentity.
•	Key Vault with the following secrets: “DatabaseLogin: DbUserName” and “DatabasePassword: MyP@ssword123456”.
•	Container Registry (ACR).
We provision these resources from the Azure Portal or using the following Powershell script:

```azurepowershell-interactive
$suffix = "demo01"
$subscriptionId = (az account show | ConvertFrom-Json).id
$tenantId = (az account show | ConvertFrom-Json).tenantId
$location = "westeurope"
$resourceGroupName = "rg-" + $suffix
$aksName = "aks-" + $suffix
$aksVersion = "1.16.13"
$keyVaultName = "keyvaultaks" + $suffix
$secret1Name = "DatabaseLogin"
$secret2Name = "DatabasePassword"
$secret1Alias = "DATABASE_LOGIN"
$secret2Alias = "DATABASE_PASSWORD" 
$identityName = "identity-aks-kv"
$identitySelector = "azure-kv"
$secretProviderClassName = "secret-provider-kv"
$acrName = "acrforaks" + $suffix
$isAKSWithManagedIdentity = "true"

# echo "Creating Resource Group..."
$resourceGroup = az group create -n $resourceGroupName -l $location | ConvertFrom-Json

# echo "Creating ACR..."
$acr = az acr create --resource-group $resourceGroupName --name $acrName --sku Basic | ConvertFrom-Json
az acr login -n $acrName --expose-token

If ($isAKSWithManagedIdentity -eq "true") {
echo "Creating AKS cluster with Managed Identity..."
$aks = az aks create -n $aksName -g $resourceGroupName --kubernetes-version $aksVersion --node-count 1 --attach-acr $acrName  --enable-managed-identity | ConvertFrom-Json
} Else {
echo "Creating AKS cluster with Service Principal..."
$aks = az aks create -n $aksName -g $resourceGroupName --kubernetes-version $aksVersion --node-count 1 --attach-acr $acrName | ConvertFrom-Json
}
# retrieve the existing or created AKS
$aks = (az aks show -n $aksName -g $resourceGroupName | ConvertFrom-Json)
# echo "Connecting/authenticating to AKS..."
az aks get-credentials -n $aksName -g $resourceGroupName
echo "Creating Key Vault..."
$keyVault = az keyvault create -n $keyVaultName -g $resourceGroupName -l $location --enable-soft-delete true --retention-days 7 | ConvertFrom-Json
# $keyVault = az keyvault show -n $keyVaultName | ConvertFrom-Json # retrieve existing KV
echo "Creating Secrets in Key Vault..."
az keyvault secret set --name $secret1Name --value "DbUserName" --vault-name $keyVaultName
az keyvault secret set --name $secret2Name --value "P@ssword123456" --vault-name $keyVaultName
```

> [!IMPORTANT]
> 
> Key Vault, AKS and Identity are in the same resource group here for simplicity. But they can be deployed on different ones.
> 

## Installing Secrets Store CSI driver and Key Vault Provider

We’ll start by installing Secrets Store CSI driver using Helm charts into a separate namespace.

```azurecli-interactive
helm repo add csi-secrets-store-provider-azure https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/charts
"secrets-store-csi-driver-provider-azure" has been added to your repositories
kubectl create ns csi-driver
namespace/csi-driver created
helm install csi-azure csi-secrets-store-provider-azure/csi-secrets-store-provider-azure --namespace csi-driver
NAME: csi-azure
LAST DEPLOYED: Thu Jun 11 11:57:28 2020
NAMESPACE: csi-driver
STATUS: deployed
REVISION: 1
TEST SUITE: None
```
Let's check the new created pods:

```azurecli-interactive
kubectl get pods --namespace=csi-driver
NAME                                               READY   STATUS              RESTARTS   AGE
csi-azure-csi-secrets-store-provider-azure-9mf84   0/1     ContainerCreating   0          3s
csi-azure-secrets-store-csi-driver-rpn7f           0/3     ContainerCreating   0          3s
```

## Using the Azure Key Vault Provider

Now the driver is installed. Let's use the SecretProviderClass to configure the Key Vault instance to connect to the specific keys, secrets, or certificates to retrieve. Note how we are providing the Key Vault name, resource group, subscription Id, tenant Id, and then the name of the secrets.

```azurecli-interactive
$secretProviderKV = @"
apiVersion: secrets-store.csi.x-k8s.io/v1alpha1
kind: SecretProviderClass
metadata:
  name: $($secretProviderClassName)
spec:
  provider: azure
  parameters:
    usePodIdentity: "true"
    useVMManagedIdentity: "false"
    userAssignedIdentityID: ""
    keyvaultName: $keyVaultName
    cloudName: AzurePublicCloud
    objects:  |
      array:
        - |
          objectName: $secret1Name
          objectAlias: $secret1Alias
          objectType: secret
          objectVersion: ""
        - |
          objectName: $secret2Name
          objectAlias: $secret2Alias
          objectType: secret
          objectVersion: ""
    resourceGroup: $resourceGroupName
    subscriptionId: $subscriptionId
    tenantId: $tenantId
"@
$secretProviderKV | kubectl create -f -
secretproviderclass.secrets-store.csi.x-k8s.io/secret-provider-kv created
```

> [!NOTE]
> Note: This sample used only Secrets in Key Vault, but we can also retrieve Keys and Certificates.

## Creating role assignments for AKS cluster (only for Managed Identity)

If we are using AKS with Managed Identity, then we should create the following two role assignments:

```azurepowershell-interactive
# Run the following 2 commands only if using AKS with Managed Identity
If ($isAKSWithManagedIdentity -eq "true") {
az role assignment create --role "Managed Identity Operator" --assignee $aks.identityProfile.kubeletidentity.clientId --scope /subscriptions/$subscriptionId/resourcegroups/$($aks.nodeResourceGroup)
az role assignment create --role "Virtual Machine Contributor" --assignee $aks.identityProfile.kubeletidentity.clientId --scope /subscriptions/$subscriptionId/resourcegroups/$($aks.nodeResourceGroup)
# If user-assigned identities that are not within the cluster resource group
# az role assignment create --role "Managed Identity Operator" --assignee $aks.identityProfile.kubeletidentity.clientId --scope /subscriptions/$subscriptionId/resourcegroups/$resourceGroupName
}
```

## Installing Pod Identity and providing access to Key Vault

The Azure Key Vault Provider offers four modes for accessing a Key Vault instance: Service Principal, Pod Identity, VMSS User Assigned Managed Identity and VMSS System Assigned Managed Identity.
Here we'll be using Pod Identity. Azure AD Pod Identity will be used to create an Identity in AAD and assign the right roles and resources. Let's first install it into the cluster.

## Installing AAD Pod Identity into AKS

We deploy Pod Identity using a Helm chart. The chart will install Node Managed Identity (NMI) and Managed Identity Controllers (MIC) on each node.

```azurecli-interactive
helm repo add aad-pod-identity https://raw.githubusercontent.com/Azure/aad-pod-identity/master/charts
helm install pod-identity aad-pod-identity/aad-pod-identity
kubectl get pods

NAME                                   READY   STATUS              RESTARTS   AGE
aad-pod-identity-mic-c558d8649-4vmj2   1/1     Running             0          29s
aad-pod-identity-mic-c558d8649-rrhvd   1/1     Running             0          29s
aad-pod-identity-nmi-t62zh             1/1     Running             0          29s
```

## Creating or retrieving Azure Identity

If we are using an AKS cluster with Managed Identity, then Azure has already created the Identity resource with AKS. So, we’ll go to retrieve it. But if we created an AKS cluster with Service Principal, then we need to create a new Identity. 

```azurepowershell-interactive
# If using AKS with Managed Identity, retrieve the existing Identity
If ($isAKSWithManagedIdentity -eq "true") {
echo "Retrieving the existing Azure Identity..."
$existingIdentity = az resource list -g $aks.nodeResourceGroup --query "[?contains(type, 'Microsoft.ManagedIdentity/userAssignedIdentities')]"  | ConvertFrom-Json
$identity = az identity show -n $existingIdentity.name -g $existingIdentity.resourceGroup | ConvertFrom-Json
} Else {
# If using AKS with Service Principal, create new Identity
echo "Creating an Azure Identity..."
$identity = az identity create -g $resourceGroupName -n $identityName | ConvertFrom-Json
}
 
$identity
{
  "clientId": "a0c038fd-3df3-4eaf-bb34-abdd4f78a0db",
  "clientSecretUrl": "https://control-westeurope.identity.azure.net/subscriptions/<AZURE_SUBSCRIPTION_ID>/resourcegroups/rg-demo/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identity-aks-kv/crede
ntials?tid=<AZURE_TENANT_ID>&oid=f8bb59bd-b704-4274-8391-3b0791d7a02c&aid=a0c038fd-3df3-4eaf
  "id": "/subscriptions/<AZURE_SUBSCRIPTION_ID>/resourcegroups/rg-demo/providers/Microsoft.Managed
Identity/userAssignedIdentities/identity-aks-kv",
  "location": "westeurope",
  "name": "identity-aks-kv",
  "principalId": "000000-b704-4274-8391-3b0791d7a02c",
  "resourceGroup": "rg-demo",
  "tags": {},
  "tenantId": "<AZURE_TENANT_ID>",
  "type": "Microsoft.ManagedIdentity/userAssignedIdentities"
}
```

## Assigning Reader Role to new Identity for Key Vault

The Identity we created earlier will be used by AKS Pods to read secrets from Key Vault. Thus, it should have permissions to do so. We will assign it the Reader role to the KV scope.

```azurecli-interactive
az role assignment create --role "Reader" --assignee $identity.principalId --scope $keyVault.id

{
  "canDelegate": null,
  "id": "/subscriptions/<AZURE_SUBSCRIPTION_ID>/resourcegroups/rg-demo/providers/Microsoft.KeyVault/vaults/
az-key-vault-demo/providers/Microsoft.Authorization/roleAssignments/d6bd00b8-9734-4c53-9de3-5a5b203c3286",
  "name": "d6bd00b8-9734-4c53-9de3-5a5b203c3286",
  "principalId": "f8bb59bd-b704-4274-8391-3b0791d7a02c",
  "principalName": "a0c038fd-3df3-4eaf-bb34-abdd4f78a0db",
  "principalType": "ServicePrincipal",
  "resourceGroup": "rg-demo",
  "roleDefinitionId": "/subscriptions/<AZURE_SUBSCRIPTION_ID>/providers/Microsoft.Authorization/roleDefinit
ions/acdd72a7-3385-48ef-bd42-f606fba81ae7",
  "roleDefinitionName": "Reader",
  "scope": "/subscriptions/<AZURE_SUBSCRIPTION_ID>/resourcegroups/rg-demo/providers/Microsoft.KeyVault/vaul
ts/az-key-vault-demo",
  "type": "Microsoft.Authorization/roleAssignments"
}
```

## Providing required permissions for MIC

In case you chose AKS with Service Principal, you need also to grant permissions to the AKS cluster with role “Managed Identity Operator”. For that we will need the Service Principal (SPN) used with AKS.
 
```azurecli-interactive
# Run the following command only if using AKS with Service Principal
If ($isAKSWithManagedIdentity -eq "false") {
echo "Providing required permissions for MIC..."
az role assignment create --role "Managed Identity Operator" --assignee $aks.servicePrincipalProfile.clientId --scope $identity.id
}

{
  "canDelegate": null,
  "id": "/subscriptions/<AZURE_SUBSCRIPTION_ID>/resourcegroups/rg-demo/providers/Microsoft.ManagedIdentity/
userAssignedIdentities/identity-aks-kv/providers/Microsoft.Authorization/roleAssignments/c018c932-c06b-446c-863e-bc85c68
7cf69",
  "name": "c018c932-c06b-446c-863e-bc85c687cf69",
  "principalId": "2736b5eb-e79e-48fa-9348-19f9c64ce7b3",
  "principalName": "http://aks-demoSP-20200430052736",
  "principalType": "ServicePrincipal",
  "resourceGroup": "rg-demo",
  "roleDefinitionId": "/subscriptions/<AZURE_SUBSCRIPTION_ID>/providers/Microsoft.Authorization/roleDefinit
ions/f1a07417-d97a-45cb-824c-7a7467783830",
  "roleDefinitionName": "Managed Identity Operator",
  "scope": "/subscriptions/<AZURE_SUBSCRIPTION_ID>/resourcegroups/rg-demo/providers/Microsoft.ManagedIdenti
ty/userAssignedIdentities/identity-aks-kv",
  "type": "Microsoft.Authorization/roleAssignments"
}
```

## Setting policy to access secrets in Key Vault

We should tell Key Vault to allow the Identity to do only specific actions on the secrets like get, list, delete, update. In our case, we need only permissions for GET. This permission is granted by using a Policy.

```azurecli-interactive
az keyvault set-policy -n $keyVaultName --secret-permissions get --spn $identity.clientId

{
  "id": "/subscriptions/<AZURE_SUBSCRIPTION_ID>/resourceGroups/demo-rg/providers/Microsoft.KeyVault/vaults/kv-aks-demo",
  "location": "westeurope",
  "name": "kv-aks-demo",
  "properties": {
    "accessPolicies": [
      {
… code removed for brievety …
        "permissions": {
          "certificates": null,
          "keys": null,
          "secrets": [
            "get"

    "softDeleteRetentionInDays": null,
    "tenantId": "<AZURE_TENANT_ID>",
    "vaultUri": "https://kv-aks-demo.vault.azure.net/"
  },
  "resourceGroup": "demo-rg",
  "tags": {},
  "type": "Microsoft.KeyVault/vaults"
}
```

> [!NOTE]
> Note: To set policy to access keys or certs in keyvault, replace: *--secret-permissions* by: *--key-permissions* or *--certificate-permissions*.

## Adding AzureIdentity and AzureIdentityBinding

The Pod needs to use the Identity to access to Key Vault. We’ll point to that Identity in AKS using AzureIdentity object and then we’ll assign it to the Pod through AzureIdentityBinding.

```azurecli-interactive
$aadPodIdentityAndBinding = @"
apiVersion: aadpodidentity.k8s.io/v1
kind: AzureIdentity
metadata:
  name: $($identityName)
spec:
  type: 0
  resourceID: $($identity.id)
  clientID: $($identity.clientId)
---
apiVersion: aadpodidentity.k8s.io/v1
kind: AzureIdentityBinding
metadata:
  name: $($identityName)-binding
spec:
  azureIdentity: $($identityName)
  selector: $($identitySelector)
"@

$aadPodIdentityAndBinding | kubectl apply -f -

azureidentity.aadpodidentity.k8s.io/identity-aks-kv created
azureidentitybinding.aadpodidentity.k8s.io/identity-aks-kv-binding created
```


> [!TIP]
> Note: Set type: 0 for Managed Service Identity; type: 1 for Service Principal. In this case, we are using managed service identity, type: 0.

> [!NOTE]
> Note the selector here as we’ll reuse it later with the pod that needs access to the Identity.

## Accessing Key Vault secrets from a Pod in AKS

At this stage, we can create a Pod and mount CSI driver on which we’ll find the login and password retrieved from Key Vault. Let's deploying a Nginx Pod for testing

```azurecli-interactive
$nginxPod = @"
kind: Pod
apiVersion: v1
metadata:
  name: nginx-secrets-store
  labels:
    aadpodidbinding: $($identitySelector)
spec:
  containers:
    - name: nginx
      image: nginx
      volumeMounts:
      - name: secrets-store-inline
        mountPath: "/mnt/secrets-store"
        readOnly: true
  volumes:
    - name: secrets-store-inline
      csi:
        driver: secrets-store.csi.k8s.io
        readOnly: true
        volumeAttributes:
          secretProviderClass: $($secretProviderClassName)
"@

$nginxPod | kubectl apply -f -
pod/nginx-secrets-store created
```

## Validating the pod has access to the secrets from Key Vault

Now, we’ll validate all what we have done before. Let’s list and read the content of the mounted CSI volume. If all is fine, we should see the secret values from our Key Vault.

```azurecli-interactive
kubectl exec -it nginx-secrets-store ls /mnt/secrets-store/
DATABASE_LOGIN  DATABASE_PASSWORD
kubectl exec -it nginx-secrets-store cat /mnt/secrets-store/DATABASE_LOGIN
DbUserName
kubectl exec -it nginx-secrets-store cat /mnt/secrets-store/DATABASE_PASSWORD
MyP@ssword123456
```

## Clean up resources 
To clean up the resources, you will purge Key Vault and delete the created resource groups.

```azurecli-interactive
az keyvault purge -n $keyVaultName
az group delete --no-wait --yes -n $resourceGroupName
az group delete --no-wait --yes -n $aks.nodeResourceGroup
```
