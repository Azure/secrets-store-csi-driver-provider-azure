---
type: docs
title: "Service Principal"
linkTitle: "Service Principal"
weight: 1
description: >
  Use a Service Principal to access Keyvault. 
---

<details>
<summary>Examples</summary>

- `SecretProviderClass`
```yaml
# This is a SecretProviderClass example using a service principal to access Key Vault
apiVersion: secrets-store.csi.x-k8s.io/v1alpha1
kind: SecretProviderClass
metadata:
  name: azure-kvname
spec:
  provider: azure
  parameters:
    usePodIdentity: "false"         # [OPTIONAL] if not provided, will default to "false"
    keyvaultName: "kvname"          # the name of the KeyVault
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
    tenantId: "tid"                 # the tenant ID of the KeyVault
``` 

- `Pod` yaml
```yaml
# This is a sample pod definition for using SecretProviderClass and service-principal to access Key Vault
kind: Pod
apiVersion: v1
metadata:
  name: busybox-secrets-store-inline
spec:
  containers:
  - name: busybox
    image: k8s.gcr.io/e2e-test-images/busybox:1.29
    command:
      - "/bin/sleep"
      - "10000"
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
          secretProviderClass: "azure-kvname"
        nodePublishSecretRef:                       # Only required when using service principal mode
          name: secrets-store-creds                 # Only required when using service principal mode
```
</details>

## Configure Service Principal to access Keyvault

1. Add your service principal credentials as a Kubernetes secrets accessible by the Secrets Store CSI driver. If using AKS you can learn about [service principals in AKS here.](https://docs.microsoft.com/azure/aks/kubernetes-service-principal) 

    A properly configured service principal will need to be passed in with the Service Principal's `appId` and `password`. Ensure this service principal has all the required permissions to access content in your Azure Key Vault instance.

    ```bash
    # Client ID (AZURE_CLIENT_ID) will be the App ID of your service principal
    # Client Secret (AZURE_CLIENT_SECRET) will be the Password of your service principal

    kubectl create secret generic secrets-store-creds --from-literal clientid=<AZURE_CLIENT_ID> --from-literal clientsecret=<AZURE_CLIENT_SECRET>

    # Label the secret
    # Refer to https://secrets-store-csi-driver.sigs.k8s.io/load-tests.html for more details on why this is necessary in future releases.
    kubectl label secret secrets-store-creds secrets-store.csi.k8s.io/used=true
    ```

    {{% alert title="NOTE" color="warning" %}}
    The Kubernetes Secret containing the credentials need to be created in the same namespace as the application pod. If pods in multiple namespaces need to use the same SP to access Keyvault, this Kubernetes Secret needs to be created in each namespace.
    {{% /alert %}}

    The requirement for `nodePublishSecretRef` to be in the same namespace as the pod referencing it in volume is imposed by the Kubernetes core object type. In case of CSI Volumes, the `nodePublishSecretRef` is a [LocalObjectReference](https://pkg.go.dev/k8s.io/api/core/v1?tab=doc#LocalObjectReference) which only accepts the name of the secret. The namespace is always [defaulted to the pod namespace for the secret](https://github.com/kubernetes/kubernetes/blob/release-1.18/pkg/volume/csi/csi_mounter.go#L169-L171). In case of `PersistentVolume` the `nodePublishSecretRef` is a [secretRef](https://pkg.go.dev/k8s.io/api/core/v1?tab=doc#SecretReference) which accepts both name and namespace.

    **If you do not have a service principal**, run the following Azure CLI command to create a new service principal.

    ```bash
    # OPTIONAL: Create a new service principal, be sure to notate the SP secret returned on creation.
    az ad sp create-for-rbac --skip-assignment --name $SPNAME

    # If you lose your AZURE_CLIENT_SECRET (SP Secret), you can reset and receive it with this command:
    # az ad sp credential reset --name $SPNAME --credential-description "APClientSecret" --query password -o tsv
    ```

    With an existing service principal, assign the following permissions:

    ```bash
    # Set environment variables
    SPNAME=<servicePrincipalName>
    AZURE_CLIENT_ID=$(az ad sp show --id http://${SPNAME} --query appId -o tsv)
    KEYVAULT_NAME=<key-vault-name>
    KEYVAULT_RESOURCE_GROUP=<resource-group-name-for-KV>
    SUBID=<subscription-id>

    az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --spn $AZURE_CLIENT_ID
    az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --spn $AZURE_CLIENT_ID
    az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --spn $AZURE_CLIENT_ID
    ```

2. Update your [deployment yaml](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/examples/service-principal/pod-inline-volume-service-principal.yaml) to reference the service principal kubernetes secret created in the previous step

    If you did not change the name of the secret reference previously, no changes are needed.

    ```yaml
    nodePublishSecretRef:
      name: secrets-store-creds
    ```


## Pros:
1. Supported on both Windows and Linux.

## Cons:
1. Service Principal credentials(client id & client secret) need to be created as a kubernetes *Secret* which is stored as plaintext in etcd.

1. The only supported way to connect to Azure Key Vault from a non Azure environment.