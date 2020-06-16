# Using Secrets Store CSI and Azure Key Vault Provider to Enable NGINX Ingress Controller with TLS

This guide demonstrates steps required to setup Secrets Store CSI driver and Azure Key Vault Provider to enable applications to work with NGINX Ingress Controller with TLS stored in Key Vault. 
For more information on securing an Ingress with TLS, refer to: https://kubernetes.io/docs/concepts/services-networking/ingress/#tls

## Generate a TLS Cert

```bash
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -out ingress-tls.crt \
    -keyout ingress-tls.key \
    -subj "/CN=demo.test.com/O=ingress-tls"
```

## Deploy Secrets-store CSI and the Azure Key Vault Provider
https://github.com/Azure/secrets-store-csi-driver-provider-azure#install-the-secrets-store-csi-driver-and-the-azure-keyvault-provider

## Deploy Ingress Controller

**Create a namespace**

```bash
kubectl create ns ingress-test
```

**Helm install ingress-controller**

```bash
helm install stable/nginx-ingress --generate-name \
    --namespace ingress-test \
    --set controller.replicaCount=2 \
    --set controller.nodeSelector."beta\.kubernetes\.io/os"=linux \
    --set defaultBackend.nodeSelector."beta\.kubernetes\.io/os"=linux
```

## Deploy a SecretsProviderClass Resource

- To provide identity to access key vault, refer to the following [section](https://github.com/Azure/secrets-store-csi-driver-provider-azure#provide-identity-to-access-key-vault).
- Set the `tenantId` and `keyvaultName`
- Use `objectType: secret` for `ingresscert` as this is the only way to retrieve the certificate and private key from azure key vault as documented [here](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/docs/getting-certs-and-keys.md#getting-certificates-and-keys-using-azure-key-vault-provider)
- Set secret type to `kubernetes.io/tls`

```bash
$ cat <<EOF | kubectl apply -f -
apiVersion: secrets-store.csi.x-k8s.io/v1alpha1
kind: SecretProviderClass
metadata:
  name: azure-tls
spec:
  provider: azure
  secretObjects:                                # secretObjects defines the desired state of synced K8s secret objects
  - secretName: ingress-tls-csi
    type: kubernetes.io/tls
    data: 
    - objectName: ingresscert
      key: tls.key
    - objectName: ingresscert
      key: tls.crt
  parameters:
    usePodIdentity: "false"
    keyvaultName: "azkv"                        # the name of the KeyVault
    objects: |
      array:
        - |
          objectName: ingresscert
          objectType: secret
    tenantId: "xx-xxxxxxxx-xx"                  # the tenant ID of the KeyVault
EOF
```

## Deploy Test Apps with Reference to Secrets Store CSI

> NOTE: These apps reference a Secrets Store CSI volume and a `secretProviderClass` object created earlier. A Kubernetes secret `ingress-tls-csi` will be created by the CSI driver as a result of the app creation.

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

If not using [service principal mode](../../docs/service-principal-mode.md), remove the following snippet from [deployment-app-one.yaml](deployment-app-one.yaml) and [deployment-app-two.yaml](deployment-app-two.yaml)

```yaml
            nodePublishSecretRef:
              name: secrets-store-creds
```

**Deploy the test apps**

```bash
kubectl apply -f sample/ingress-controller-tls/deployment-app-one.yaml -n ingress-test
kubectl apply -f sample/ingress-controller-tls/deployment-app-two.yaml -n ingress-test
```

## Check for the Kubernetes Secret created by the CSI driver
```bash
kubectl get secret -n ingress-test

NAME                                             TYPE                                  DATA   AGE
ingress-tls-csi                                  kubernetes.io/tls                     2      1m34s
```

## Deploy an Ingress Resource referencing the Secret created by the CSI driver

> NOTE: The ingress resource references the Kubernetes secret `ingress-tls-csi` created by the CSI driver as a result of the app creation.

```yaml
tls:
  - hosts:
    - demo.test.com
    secretName: ingress-tls-csi
```

```bash
kubectl apply -f sample/ingress-controller-tls/ingress.yaml -n ingress-test
```

## Get the External IP of the Ingress Controller

```bash
 kubectl get service -l app=nginx-ingress --namespace ingress-test 
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
