---
type: docs
title: "Sync Multiple Versions of a Secret"
linkTitle: "Sync Multiple Versions of a Secret"
weight: 1
description: >
  How to sync multiple versions of a key vault secret, key, or certificate
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
          objectVersionHistory: 5                # The number of versions to sync (including the specified version)
        - |
          objectName: $KEY_NAME
          objectType: key
          objectVersion: $KEY_VERSION
          objectVersionHistory: 5                # The number of versions to sync (including the specified version)
    tenantID: "tid"                              # the tenant ID of the KeyVault
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
      image: registry.k8s.io/e2e-test-images/busybox:1.29-4
      command:
        - "/bin/sleep"
        - "10000"
  volumes:
    - name: secrets-store01-inline
      csi:
        driver: secrets-store.csi.k8s.io
        readOnly: true
        volumeAttributes:
          secretProviderClass: "azure-sync"
```

</details>

### How to sync multiple versions of a secret

In some cases, you may need to sync the latest N versions of a secret. Use the optional `objectVersionHistory` field to define the number of previous versions to sync. You will also need to add the secrets/list permission to whichever principal is being used to interact with Key Vault.

When you do this, the provider will treat the object name/alias as a folder and place the top N (where N is `objectVersionHistory`) versions of the secret (sorted by secret creation date) into that folder. The file name for each version will be an integer, starting with `0` for the specified version, `1` for the next most recent, and so on.

> NOTE: If you specify a version, the provider will sync the top N starting with that specified version. If you do not specify a version, or specify `latest`, then it will sync the most recent N as determined by the secret creation date.

> NOTE: If you are syncing this secret with a Kubernetes secret, make sure the `objectName` in `secretObjects` also indicates which version of the secret to sync (i.e., `foosecret/0` instead of just `foosecret`)

{{% alert title="NOTE" color="warning" %}}

- There may be fewer than `objectVersionHistory` versions synced. For instance if you specify 5 and the secret only has 3 versions, then only 3 versions will be synced.
- Disabled versions of the secret are ignored.

{{% /alert %}}
