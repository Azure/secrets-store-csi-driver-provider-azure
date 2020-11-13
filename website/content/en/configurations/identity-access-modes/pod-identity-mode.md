---
type: docs
title: "Pod Identity"
linkTitle: "Pod Identity"
weight: 2
description: >
  Use Pod Identity
---

> Supported only on Linux

**Prerequisites**

💡 Make sure you have installed pod identity to your Kubernetes cluster

   __This project makes use of the aad-pod-identity project located  [here](https://github.com/Azure/aad-pod-identity#getting-started) to handle the identity management of the pods. Reference the aad-pod-identity README if you need further instructions on any of these steps.__

Not all steps need to be followed on the instructions for the aad-pod-identity project as we will also complete some of the steps on our installation here.

1. Install the aad-pod-identity components to your cluster

   - Install the RBAC enabled aad-pod-identiy infrastructure components:
      ```
      kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
      ```

   - 💡 Follow the [Role assignment](https://github.com/Azure/aad-pod-identity/blob/master/docs/readmes/README.role-assignment.md) documentation to setup all the required roles for aad-pod-identity components.

1. Create an Azure User Identity

    Create an Azure User Identity with the following command.
    Get `clientId` and `id` from the output.
    ```
    az identity create -g <resourcegroup> -n <idname>
    ```

1. Assign permissions to new identity
    Ensure your Azure user identity has all the required permissions to read the keyvault instance and to access content within your key vault instance.
    If not, you can run the following using the Azure cli:

    ```bash
    # Assign Reader Role to new Identity for your keyvault
    az role assignment create --role Reader --assignee <principalid> --scope /subscriptions/<subscriptionid>/resourcegroups/<resourcegroup>/providers/Microsoft.KeyVault/vaults/<keyvaultname>

    # set policy to access keys in your keyvault
    az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --spn <YOUR AZURE USER IDENTITY CLIENT ID>
    # set policy to access secrets in your keyvault
    az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --spn <YOUR AZURE USER IDENTITY CLIENT ID>
    # set policy to access certs in your keyvault
    az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --spn <YOUR AZURE USER IDENTITY CLIENT ID>
    ```

1. Add a new `AzureIdentity` for the new identity to your cluster

    Edit and save this as `aadpodidentity.yaml`

    Set `type: 0` for Managed Service Identity; `type: 1` for Service Principal
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

1. Add a new `AzureIdentityBinding` for the new Azure identity to your cluster

    Edit and save this as `aadpodidentitybinding.yaml`
    ```yaml
    apiVersion: "aadpodidentity.k8s.io/v1"
    kind: AzureIdentityBinding
    metadata:
      name: <any-name>
    spec:
      azureIdentity: <name of AzureIdentity created from previous step>
      selector: <label value to match in your app>
    ```

    ```
    kubectl create -f aadpodidentitybinding.yaml
    ```

2. Add the following to [this](../examples/nginx-pod-inline-volume-pod-identity.yaml) deployment yaml:

    Include the `aadpodidbinding` label matching the `selector` value set in the previous step so that this pod will be assigned an identity
    ```yaml
    metadata:
    labels:
      aadpodidbinding: <AzureIdentityBinding Selector created from previous step>
    ```
    
3. Update [this sample deployment](../examples/v1alpha1_secretproviderclass_pod_identity.yaml) to create a `SecretProviderClass` resource with `usePodIdentity: "true"` to provide Azure-specific parameters for the Secrets Store CSI driver.

    Make sure to update `usepodidentity` to `true`
    ```yaml
    usepodidentity: "true"
    ```
    
4. Deploy your app

    ```bash
    kubectl apply -f ../examples/nginx-pod-secrets-store-inline-volume-secretproviderclass-podid.yaml
    ```

**NOTE** When using the `Pod Identity` option mode, there can be some amount of delay in obtaining the objects from keyvault. During the pod creation time, in this particular mode `aad-pod-identity` will need to create the `AzureAssignedIdentity` for the pod based on the `AzureIdentity` and `AzureIdentityBinding`, retrieve token for keyvault. This process can take time to complete and it's possible for the pod volume mount to fail during this time. When the volume mount fails, kubelet will keep retrying until it succeeds. So the volume mount will eventually succeed after the whole process for retrieving the token is complete.
