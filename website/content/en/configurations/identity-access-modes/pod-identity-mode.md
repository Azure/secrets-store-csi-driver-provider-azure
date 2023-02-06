---
type: docs
title: "Pod Identity"
linkTitle: "Pod Identity"
weight: 4
description: >
  Use Pod Identity to access Keyvault.
---

<details>
<summary>Examples</summary>

- `SecretProviderClass`
```yaml
# This is a SecretProviderClass example using aad-pod-identity to access Key Vault
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: azure-kvname-podid
spec:
  provider: azure
  parameters:
    usePodIdentity: "true"          # set to true for pod identity access mode
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
# This is a sample pod definition for using SecretProviderClass and aad-pod-identity to access Key Vault
kind: Pod
apiVersion: v1
metadata:
  name: busybox-secrets-store-inline-podid
  labels:
    aadpodidbinding: "demo"         # Set the label value to match selector defined in AzureIdentityBinding
spec:
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
          secretProviderClass: "azure-kvname-podid"
```
</details>

## Configure AAD Pod Identity to access Keyvault

> NOTE: [AAD Pod Identity](https://github.com/Azure/aad-pod-identity) has been [DEPRECATED](https://github.com/Azure/aad-pod-identity#-announcement). We recommend using [Workload Identity](../workload-identity-mode) instead.

**Prerequisites**

💡 Make sure you have installed pod identity to your Kubernetes cluster

   __This project makes use of the [aad-pod-identity](https://github.com/Azure/aad-pod-identity#getting-started) project to handle the identity management of the pods. Reference the aad-pod-identity README if you need further instructions on any of these steps.__

Not all steps need to be followed on the instructions for the aad-pod-identity project as we will also complete some of the steps on our installation here.

1. Install the aad-pod-identity components to your cluster

   - 💡 Follow the [Role assignment](https://azure.github.io/aad-pod-identity/docs/getting-started/role-assignment/) documentation to setup all the required roles for aad-pod-identity components.

   - Install the RBAC enabled aad-pod-identiy infrastructure components:
      ```
      kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
      ```


2. Create an Azure User-assigned Managed Identity

    Create an Azure User-assigned Managed Identity with the following command.
    Get `clientId` and `id` from the output.
    ```
    az identity create -g <resourcegroup> -n <idname>
    ```

3. Assign permissions to new identity
    Ensure your Azure user identity has all the required permissions to read the keyvault instance and to access content within your key vault instance.
    If not, you can run the following using the Azure CLI:

    ```bash
    # set policy to access keys in your keyvault
    az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --spn <YOUR AZURE USER IDENTITY CLIENT ID>
    # set policy to access secrets in your keyvault
    az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --spn <YOUR AZURE USER IDENTITY CLIENT ID>
    # set policy to access certs in your keyvault
    az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --spn <YOUR AZURE USER IDENTITY CLIENT ID>
    ```

4. Add an `AzureIdentity` for the new identity to your cluster

    Edit and save this as `aadpodidentity.yaml`

    Set `type: 0` for User-Assigned Managed Identity; `type: 1` for Service Principal
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

5. Add `AzureIdentityBinding` for the `AzureIdentity` to your cluster

    Edit and save this as `aadpodidentitybinding.yaml`
    ```yaml
    apiVersion: "aadpodidentity.k8s.io/v1"
    kind: AzureIdentityBinding
    metadata:
      name: <any-name>
    spec:
      azureIdentity: <name of the AzureIdentity created in previous step>
      selector: <label value to match in your pod>
    ```

    ```
    kubectl create -f aadpodidentitybinding.yaml
    ```

6. Add the following to [this](https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/examples/pod-identity/pod-inline-volume-pod-identity.yaml) deployment yaml:

    Include the `aadpodidbinding` label matching the `selector` value set in the previous step so that this pod will be assigned an identity
    ```yaml
    metadata:
    labels:
      aadpodidbinding: <AzureIdentityBinding Selector created from previous step>
    ```

7. Update [this sample deployment](https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/examples/pod-identity/v1alpha1_secretproviderclass_pod_identity.yaml) to create a `SecretProviderClass` resource with `usePodIdentity: "true"` to provide Azure-specific parameters for the Secrets Store CSI driver.

    Make sure to set `usePodIdentity` to `true`
    ```yaml
    usePodIdentity: "true"
    ```

8. Deploy your app

    ```bash
    kubectl apply -f pod.yaml
    ```

**NOTE** When using the `Pod Identity` option mode, there can be some amount of delay in obtaining the objects from keyvault. During the pod creation time, in this particular mode `aad-pod-identity` will need to create the `AzureAssignedIdentity` for the pod based on the `AzureIdentity` and `AzureIdentityBinding`, retrieve token for keyvault. This process can take time to complete and it's possible for the pod volume mount to fail during this time. When the volume mount fails, kubelet will keep retrying until it succeeds. So the volume mount will eventually succeed after the whole process for retrieving the token is complete.

<br>

## Pros:
1. Provides secure way to access cloud resources that depends on Azure Active Directory as identity provider.

## Cons:
1. Supported only on Linux
