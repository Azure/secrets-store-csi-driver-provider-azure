
---
type: docs
title: "Azure Key Vault Provider for Secrets Store CSI Driver"
linkTitle: "Documentation"
weight: 20
menu:
  main:
    weight: 20
---

Azure Key Vault provider for [Secrets Store CSI Driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver) allows you to get secret contents stored in an [Azure Key Vault](https://docs.microsoft.com/en-us/azure/key-vault/general/overview) instance and use the Secrets Store CSI driver interface to mount them into Kubernetes pods.

## Project Status

| Azure Key Vault Provider                                                                       | Compatible Kubernetes | `secrets-store.csi.x-k8s.io` Versions |
| ---------------------------------------------------------------------------------------------- | --------------------- | ------------------------------------- |
| [v1.5.0](https://github.com/Azure/secrets-store-csi-driver-provider-azure/releases/tag/v1.5.0) | 1.21+                 | `v1`, `v1alpha1 [DEPRECATED]`         |
| [v1.4.1](https://github.com/Azure/secrets-store-csi-driver-provider-azure/releases/tag/v1.4.1) | 1.21+                 | `v1`, `v1alpha1 [DEPRECATED]`         |

For Secrets Store CSI Driver project status and supported versions, check the doc [here](https://secrets-store-csi-driver.sigs.k8s.io/#project-status)

## Features

- Mounts secrets/keys/certs to pod using a CSI Inline volume
- Supports mounting multiple secrets store objects as a single volume
- Supports multiple secrets stores as providers. Multiple providers can run in the same cluster simultaneously.
- Supports pod portability with the SecretProviderClass CRD
- Supports Linux and Windows containers
- Supports sync with Kubernetes Secrets
- Supports auto rotation of secrets

## Managed Add-ons
Azure Key Vault provider for Secrets Store CSI Driver is available as a managed add-on in:
- Azure Kubernetes Service (AKS). For more information, see [Use the Azure Key Vault Provider for Secrets Store CSI Driver in an AKS cluster](https://learn.microsoft.com/en-us/azure/aks/csi-secrets-store-driver).
- Azure Arc enabled Kubernetes. For more information, see [Use the Azure Key Vault Secrets Provider extension to fetch secrets into Azure Arc-enabled Kubernetes clusters](https://learn.microsoft.com/en-us/azure/azure-arc/kubernetes/tutorial-akv-secrets-provider).
