
---
title: "Azure Key Vault Provider for Secrets Store CSI Driver"
linkTitle: "Documentation"
weight: 20
menu:
  main:
    weight: 20
---

Azure Key Vault provider for [Secrets Store CSI driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver) allows you to get secret contents stored in an [Azure Key Vault](https://docs.microsoft.com/en-us/azure/key-vault/general/overview) instance and use the Secrets Store CSI driver interface to mount them into Kubernetes pods.

## Features

- Mounts secrets/keys/certs on pod start using a CSI volume
- Supports mounting multiple secrets store objects as a single volume
- Supports pod identity to restrict access with specific identities
- Supports pod portability with the SecretProviderClass CRD
- Supports windows containers (Kubernetes version v1.18+)
- Supports sync with Kubernetes Secrets (Secrets Store CSI Driver v0.0.10+)
- Supports multiple secrets stores providers in the same cluster.

### Secret Content is Mounted on Pod Start
On pod start and restart, the driver will call the Azure provider binary to retrieve the secret content from the Azure Key Vault instance you have specified in the `SecretProviderClass` custom resource. Then the content will be mounted to the container's file system. 

To validate, once the pod is started, you should see the new mounted content at the volume path specified in your deployment yaml.

```bash
kubectl exec -it nginx-secrets-store-inline ls /mnt/secrets-store/
foo
```
