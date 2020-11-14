---
type: docs
title: "Syncronize Mounted Content with Kubernetes Secret"
linkTitle: "Syncronize Mounted Content with Kubernetes Secret"
weight: 1
description: >
  How to syncronize mounted content with Kubernetes secret 
---

In some cases, you may want to create a Kubernetes Secret to mirror the mounted content. Use the optional `secretObjects` field to define the desired state of the synced Kubernetes secret objects.

> NOTE: Make sure the `objectName` in `secretObjects` matches the name of the mounted content. This could be the object name or the object alias.

A `SecretProviderClass` custom resource should have the following components:
```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1alpha1
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
```
> NOTE: Here is the list of supported Kubernetes Secret types: `Opaque`, `kubernetes.io/basic-auth`, `bootstrap.kubernetes.io/token`, `kubernetes.io/dockerconfigjson`, `kubernetes.io/dockercfg`, `kubernetes.io/ssh-auth`, `kubernetes.io/service-account-token`, `kubernetes.io/tls`.  

- Here is a sample [`SecretProviderClass` custom resource](https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/master/test/bats/tests/azure/azure_synck8s_v1alpha1_secretproviderclass.yaml) that syncs a secret from Azure Key Vault to a Kubernetes secret.
- To view an example of type `kubernetes.io/tls`, refer to the [ingress-controller-tls sample](sample/ingress-controller-tls/README.md#deploy-a-secretsproviderclass-resource)