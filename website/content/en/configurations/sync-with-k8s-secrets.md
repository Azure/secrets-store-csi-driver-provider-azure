---
type: docs
title: "Sync Mounted Content with Kubernetes Secret"
linkTitle: "Sync Mounted Content with Kubernetes Secret"
weight: 1
description: >
  How to sync mounted content with Kubernetes secret
---

<details>
<summary>Examples</summary>

- `SecretProviderClass`

```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: azure-sync
spec:
  provider: azure
  secretObjects:                                 # [OPTIONAL] SecretObject defines the desired state of synced K8s secret objects
  - secretName: foosecret
    type: Opaque
    labels:                                   
      environment: "test"
    data: 
    - objectName: secretalias                    # name of the mounted content to sync. this could be the object name or object alias 
      key: username
  parameters:
    usePodIdentity: "true"                      
    keyvaultName: "$KEYVAULT_NAME"               # the name of the KeyVault
    objects: |
      array:
        - |
          objectName: $SECRET_NAME
          objectType: secret                     # object types: secret, key or cert
          objectAlias: secretalias
          objectVersion: $SECRET_VERSION         # [OPTIONAL] object versions, default to latest if empty
        - |
          objectName: $KEY_NAME
          objectType: key
          objectVersion: $KEY_VERSION
    tenantID: "tid"                             # the tenant ID of the KeyVault
```

- `Pod` yaml

```yaml
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
      - name: secrets-store01-inline
        mountPath: "/mnt/secrets-store"
        readOnly: true
  volumes:
    - name: secrets-store01-inline
      csi:
        driver: secrets-store.csi.k8s.io
        readOnly: true
        volumeAttributes:
          secretProviderClass: "azure-sync"
```

</details>

### How to sync mounted content with Kubernetes secret

In some cases, you may want to create a Kubernetes Secret to mirror the mounted content. Use the optional `secretObjects` field to define the desired state of the synced Kubernetes secret objects.

> NOTE: Make sure the `objectName` in `secretObjects` matches the file name of the mounted content. If object alias used, then it should be the object alias else this would be the object name.

> If the driver and provider have been installed using helm, ensure the `secrets-store-csi-driver.syncSecret.enabled=true` helm value is set as part of install/upgrade. This is required to install the RBAC clusterrole and clusterrolebinding required by the CSI driver to sync mounted content as Kubernetes secret. For a list of customizable values that can be injected when invoking helm install, please see the [Helm chart configurations](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/charts/csi-secrets-store-provider-azure/README.md#configuration).

{{% alert title="NOTE" color="warning" %}}

- The secrets will only sync once you *start a pod mounting the secrets*. Solely relying on the syncing with Kubernetes secrets feature thus does not work.
- The Kubernetes secrets will be synced to the same namespace as the application pod and `SecretProviderClass`.
- When all the pods consuming the secret are deleted, the Kubernetes secret is also deleted. This is done by adding the pods as owners to the created Kubernetes secret. When all the application pods consuming the Kubernetes secret are deleted, the Kubernetes secret will be garbage collected.
{{% /alert %}}

A `SecretProviderClass` custom resource should have the following components:

```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: my-provider
spec:
  provider: azure                             
  secretObjects:                              # [OPTIONAL] SecretObject defines the desired state of synced K8s secret objects
  - data:
    - key: username                           # data field to populate
      objectName: foo1                        # name of the mounted content to sync. this could be the object name or the object alias
    secretName: foosecret                     # name of the Kubernetes Secret object
    type: Opaque                              # type of the Kubernetes Secret object e.g. Opaque, kubernetes.io/tls
...
```

- Here is a sample [`SecretProviderClass` custom resource](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/examples/sync-as-kubernetes-secret/synck8s_v1alpha1_secretproviderclass.yaml) that syncs a secret from Azure Key Vault to a Kubernetes secret.
- To view an example of type `kubernetes.io/tls`, refer to the [example](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/examples/sync-as-kubernetes-secret/tls_synck8s_v1alpha1_secretproviderclass.yaml).
- To view an example of type `kubernetes.io/dockerconfigjson`, refer to the [example](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/examples/sync-as-kubernetes-secret/dockerconfigjson_synck8s_v1alpha1_secretproviderclass.yaml) that syncs `dockerconfigjson` from Azure Key Vault to a Kubernetes secret.
