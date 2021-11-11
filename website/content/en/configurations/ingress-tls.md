---
type: docs
title: "Enable NGINX Ingress Controller with TLS"
linkTitle: "Enable NGINX Ingress Controller with TLS"
weight: 3
description: >
  This guide demonstrates steps required to setup Secrets Store CSI driver and Azure Key Vault Provider to enable applications to work with NGINX Ingress Controller with TLS certificates stored in Key Vault
---

For more information on securing an Ingress with TLS, refer to [this guide](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls)

Importing the ingress TLS certificate to the cluster can be included in the following deployments:

* Application - Application deployment manifest declares and mounts the csi provider volume. Only when the application is deployed the certificate is made available in the cluster and when it is removed the secret is gone. This scenario fits development teams who are responsible for the application's security infrastructure and their integration with the cluster.
* Ingress Controller - Ingress deployment is modified to declare and mount the csi provider volume. The secret is imported when Ingress pods are created and the application's pods have no access to the TLS certificate. This scenario fits scenarios where one team (i.e. IT) manages and provisions infrastructure and networking components (including HTTPS TLS certificates) and other teams manage application lifecycle. note: in this case, ingress is specific to a single namespace/workload and is deployed in the same namespace as the application.

## Generate a TLS Cert

```bash
export CERT_NAME=ingresscert
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -out ingress-tls.crt \
    -keyout ingress-tls.key \
    -subj "/CN=demo.test.com/O=ingress-tls"
```

### Import the TLS certificate to Azure Key Vault

Convert the .crt certificate to pfx format and import it to Azure Key Vault. For example:

```bash
export AKV_NAME="[YOUR AKV NAME]"
openssl pkcs12 -export -in ingress-tls.crt -inkey ingress-tls.key  -out $CERT_NAME.pfx
# skip Password prompt

az keyvault certificate import --vault-name $AKV_NAME -n $CERT_NAME -f $CERT_NAME.pfx
```

## Setup Cluster Prerequisites

### Deploy Secrets Store CSI Driver and the Azure Key Vault Provider

Deploy the Azure Key Vault Provider and Secrets Store CSI Driver components:

```bash
helm repo add csi-secrets-store-provider-azure https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/charts
helm install csi csi-secrets-store-provider-azure/csi-secrets-store-provider-azure --set secrets-store-csi-driver.syncSecret.enabled=true
```

Refer to [installation](../getting-started/installation/_index.md) for more details and validation.

### Optional: Deploy AAD Pod Identity

If using AAD pod identity to access Azure Keyvault, make sure it is [configured properly](https://azure.github.io/aad-pod-identity/docs/demo/standard_walkthrough/) in the cluster. Refer to [doc](../configurations/identity-access-modes/pod-identity-mode.md) on how to use AAD Pod identity to access keyvault.

```bash
export AAD_POD_IDENTITY_NAME=azure-kv
```

## Deploy a SecretsProviderClass Resource

### Create a namespace

```bash
export NAMESPACE=ingress-test
kubectl create ns $NAMESPACE
```

### Create the SecretProviderClass

* To provide identity to access key vault, refer to the following [section](../configurations/identity-access-modes/_index.md).
* Set the `tenantId` and `keyvaultName`
* If using **AAD pod identity** to access Azure Key Vault - set `usePodIdentity: "true"`
* Use `objectType: secret` for the certificate, as this is the only way to retrieve the certificate and private key from azure key vault as documented [here](../configurations/getting-certs-and-keys.md)
* Set secret type to `kubernetes.io/tls`

```bash
export TENANT_ID=[YOUR TENANT ID]
```

```yaml
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: azure-tls
spec:
  provider: azure
  secretObjects:                            # secretObjects defines the desired state of synced K8s secret objects
  - secretName: ingress-tls-csi
    type: kubernetes.io/tls
    data: 
    - objectName: $CERT_NAME
      key: tls.key
    - objectName: $CERT_NAME
      key: tls.crt
  parameters:
    usePodIdentity: "false"
    keyvaultName: $AKV_NAME                 # the name of the KeyVault
    objects: |
      array:
        - |
          objectName: $CERT_NAME
          objectType: secret
    tenantId: $TENANT_ID                    # the tenant ID of the KeyVault
EOF
```

## Deploy Ingress Controller

### Add the official ingress chart repository

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
```

### Deploy Ingress

Depending on the TLS certificate lifecycle, follow one of the following steps:

* #### Bind certificate to Application

> Application's deployment will reference the csi store provider.

```bash
helm install ingress-nginx/ingress-nginx --generate-name \
    --namespace $NAMESPACE \
    --set controller.replicaCount=2 \
    --set controller.nodeSelector."beta\.kubernetes\.io/os"=linux \
    --set defaultBackend.nodeSelector."beta\.kubernetes\.io/os"=linux
```

Next, [Deploy the application](#deploy-application-with-reference-to-secrets-store-csi).

* #### Bind certificate to Ingress

> NOTE: Ingress controller references a Secrets Store CSI volume and a `secretProviderClass` object created earlier. A Kubernetes secret `ingress-tls-csi` will be created by the CSI driver as a result of ingress controller creation.

```bash
helm install ingress-nginx/ingress-nginx --generate-name \
    --namespace $NAMESPACE \
    --set controller.replicaCount=2 \
    --set controller.nodeSelector."beta\.kubernetes\.io/os"=linux \
    --set defaultBackend.nodeSelector."beta\.kubernetes\.io/os"=linux \
    --set controller.podLabels.aadpodidbinding=$AAD_POD_IDENTITY_NAME \
    -f - <<EOF
controller:
  extraVolumes:
      - name: secrets-store-inline
        csi:
          driver: secrets-store.csi.k8s.io
          readOnly: true
          volumeAttributes:
            secretProviderClass: "azure-tls"
          nodePublishSecretRef:
            name: secrets-store-creds
  extraVolumeMounts:
      - name: secrets-store-inline
        mountPath: "/mnt/secrets-store"
        readOnly: true
EOF
```

If not using [service principal mode](../configurations/identity-access-modes/service-principal-mode.md), remove the following snippet from the script:

```bash
            nodePublishSecretRef:
              name: secrets-store-creds
```

#### Check for the Kubernetes Secret created by the CSI driver (ingress-bound certificate)

```bash
kubectl get secret -n $NAMESPACE

NAME                                             TYPE                                  DATA   AGE
ingress-tls-csi                                  kubernetes.io/tls                     2      1m34s
```

Next, [Deploy the application](#deploy-application-with-ingress-reference-to-secrets-store-csi).

## Deploy Test Apps

Depending on the TLS certificate lifecycle, follow one of the following steps:

* ### Deploy Application with Reference to Secrets Store CSI

> NOTE: These apps reference a Secrets Store CSI volume and a `secretProviderClass` object created earlier. A Kubernetes secret `ingress-tls-csi` will be created by the CSI driver as a result of the app creation in the same namespace.

```yaml
      volumes:
        - name: secrets-store-inline
          csi:
            driver: secrets-store.csi.k8s.io
            readOnly: true
            volumeAttributes:
              secretProviderClass: "azure-tls"
            nodePublishSecretRef:
              name: secrets-store-creds
```

If not using [service principal mode](../configurations/identity-access-modes/service-principal-mode.md), remove the following snippet from [deployment-app-one.yaml](https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/docs/sample/ingress-controller-tls/deployment-app-one.yaml) and [deployment-app-two.yaml](https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/docs/sample/ingress-controller-tls/deployment-app-two.yaml)

```yaml
            nodePublishSecretRef:
              name: secrets-store-creds
```

#### Deploy the test apps (application-bound certificate)

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/docs/sample/ingress-controller-tls/deployment-app-one.yaml -n $NAMESPACE
kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/docs/sample/ingress-controller-tls/deployment-app-two.yaml -n $NAMESPACE
```

#### Check for the Kubernetes Secret created by the CSI driver (application-bound certificate)

```bash
kubectl get secret -n $NAMESPACE

NAME                                             TYPE                                  DATA   AGE
ingress-tls-csi                                  kubernetes.io/tls                     2      1m34s
```

Next, [Deploy the ingress resource](#deploy-an-ingress-resource-referencing-the-secret-created-by-the-csi-driver)

* ### Deploy Application with Ingress reference to Secrets Store CSI

remove the following snippet from [deployment-app-one.yaml](https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/docs/sample/ingress-controller-tls/deployment-app-one.yaml) and [deployment-app-two.yaml](https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/docs/sample/ingress-controller-tls/deployment-app-two.yaml)

```yaml
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
              secretProviderClass: "azure-tls"
            nodePublishSecretRef:
              name: secrets-store-creds
```

#### Deploy the test apps (ingress-bound certificate)

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/docs/sample/ingress-controller-tls/deployment-app-one.yaml -n $NAMESPACE
kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/docs/sample/ingress-controller-tls/deployment-app-two.yaml -n $NAMESPACE
```

Next, [Deploy the ingress resource](#deploy-an-ingress-resource-referencing-the-secret-created-by-the-csi-driver)

## Deploy an Ingress Resource referencing the Secret created by the CSI driver

> NOTE: The ingress resource references the Kubernetes secret `ingress-tls-csi` created by the CSI driver as a result of the app creation.  
> The following snippet shows the code which makes this happen:

```yaml
tls:
  - hosts:
    - demo.test.com
    secretName: ingress-tls-csi
```

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/docs/sample/ingress-controller-tls/ingress.yaml -n $NAMESPACE
```

## Get the External IP of the Ingress Controller

```bash
 kubectl get service -l app=nginx-ingress --namespace $NAMESPACE
NAME                                       TYPE           CLUSTER-IP     EXTERNAL-IP      PORT(S)                      AGE
nginx-ingress-1588032400-controller        LoadBalancer   10.0.255.157   52.xx.xx.xx      80:31293/TCP,443:31265/TCP   19m
nginx-ingress-1588032400-default-backend   ClusterIP      10.0.223.214   <none>           80/TCP                       19m
```

## Test Ingress with TLS

Using `curl` to verify ingress configuration using TLS.
Replace the public IP with the external IP of the ingress controller service from the previous step.  

```bash
curl -v -k --resolve demo.test.com:443:52.xx.xx.xx https://demo.test.com

# You should see the following in your output
*  subject: CN=demo.test.com; O=ingress-tls
*  start date: Apr 15 04:23:46 2020 GMT
*  expire date: Apr 15 04:23:46 2021 GMT
*  issuer: CN=demo.test.com; O=ingress-tls
*  SSL certificate verify result: self signed certificate (18), continuing anyway.
```
