---
type: docs
title: "Standard Walkthrough"
linkTitle: "Standard Walkthrough"
weight: 1
description: >
  You will need Azure CLI installed and a Kubernetes cluster.
---

Run the following commands to set Azure-related environment variables and login to Azure via az login:

```bash
export SUBSCRIPTION_ID="<SubscriptionID>"
export TENANT_ID="<tenant id>"

# login as a user and set the appropriate subscription ID
az login
az account set -s "${SUBSCRIPTION_ID}"

export KEYVAULT_RESOURCE_GROUP=<keyvault-resource-group>
export KEYVAULT_LOCATION=<keyvault-location>
export KEYVAULT_NAME=secret-store-$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 10 | head -n 1)
```

### 1. Deploy Azure Key Vault Provider for Secrets Store CSI Driver

Deploy the Azure Key Vault Provider and Secrets Store CSI Driver components:

```bash
helm repo add csi-secrets-store-provider-azure https://azure.github.io/secrets-store-csi-driver-provider-azure/charts
helm install csi csi-secrets-store-provider-azure/csi-secrets-store-provider-azure
```

Refer to [installation](../../getting-started/installation) for more details and validation.

### 2. Create Keyvault and set secrets

Create an Azure Keyvault instance:

```bash
  az group create -n ${KEYVAULT_RESOURCE_GROUP} --location ${KEYVAULT_LOCATION}
  az keyvault create -n ${KEYVAULT_NAME} -g ${KEYVAULT_RESOURCE_GROUP} --location ${KEYVAULT_LOCATION}
```

Add a secret to your Keyvault:

```bash
az keyvault secret set --vault-name ${KEYVAULT_NAME} --name secret1 --value "Hello\!"
```

### 3. Create an identity on Azure and set access policies

Refer to [Identity Access Modes](../../configurations/identity-access-modes) to see the list of supported modes for accessing the Key Vault instance.

In this walkthrough, we will be using the [Service Principal](../../configurations/identity-access-modes/service-principal-mode) auth mode for accessing the Key Vault instance we just created.

```bash
# Create a service principal to access keyvault
export SERVICE_PRINCIPAL_CLIENT_SECRET="$(az ad sp create-for-rbac --skip-assignment --name http://secrets-store-test --query 'password' -otsv)"
export SERVICE_PRINCIPAL_CLIENT_ID="$(az ad sp show --id http://secrets-store-test --query 'appId' -otsv)"
```

Set the access policy for keyvault objects:

```bash
az keyvault set-policy -n ${KEYVAULT_NAME} --secret-permissions get --spn ${SERVICE_PRINCIPAL_CLIENT_ID}
```

### 4. Create the Kubernetes Secret with credentials

Create the Kubernetes secret with the service principal credentials:

```bash
kubectl create secret generic secrets-store-creds --from-literal clientid=${SERVICE_PRINCIPAL_CLIENT_ID} --from-literal clientsecret=${SERVICE_PRINCIPAL_CLIENT_SECRET}
kubectl label secret secrets-store-creds secrets-store.csi.k8s.io/used=true
```

> NOTE: This step is required only if you're using service principal to provide access to Keyvault.

### 5. Deploy `SecretProviderClass`

Refer to [section](../../getting-started/usage/#create-your-own-secretproviderclass-object) on the required and configurable parameters in `SecretProviderClass` object.

Create `SecretProviderClass` in your cluster that contains all the required parameters:

```yaml
cat <<EOF | kubectl apply -f -
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: azure-kvname
  namespace: default
spec:
  provider: azure
  parameters:
    usePodIdentity: "false"
    useVMManagedIdentity: "false"
    userAssignedIdentityID: ""
    keyvaultName: "${KEYVAULT_NAME}"
    objects: |
      array:
        - |
          objectName: secret1              
          objectType: secret
          objectVersion: ""
    tenantID: "${TENANT_ID}"
EOF
```

### 6. Deployment and Validation

Create the pod with volume referencing the `secrets-store.csi.k8s.io` driver:

```yaml
cat <<EOF | kubectl apply -f -
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
EOF
```

To validate, once the pod is started, you should see the new mounted content at the volume path specified in your deployment yaml.

  ```bash
  ## show secrets held in secrets-store
  kubectl exec busybox-secrets-store-inline -- ls /mnt/secrets-store/

  ## print a test secret held in secrets-store
  kubectl exec busybox-secrets-store-inline -- cat /mnt/secrets-store/secret1
  ```

If successful, the output will be similar to:

  ```bash
  kubectl exec busybox-secrets-store-inline -- ls /mnt/secrets-store/
  secret1
  
  kubectl exec busybox-secrets-store-inline -- cat /mnt/secrets-store/secret1
  Hello!
  ```
