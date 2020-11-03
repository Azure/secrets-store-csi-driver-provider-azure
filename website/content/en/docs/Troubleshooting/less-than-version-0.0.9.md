---
title: "For Azure Key Vault Provider version < `0.0.9`"
linkTitle: "For Azure Key Vault Provider version < `0.0.9`"
weight: 1
description: >
  For versions less than 0.0.9
---

To troubleshoot issues with the csi driver and the provider, you can look at logs from the `secrets-store` container of the csi driver pod running on the same node as your application pod:

  ```bash
  kubectl get pod -o wide
  # find the secrets store csi driver pod running on the same node as your application pod

  kubectl logs csi-secrets-store-secrets-store-csi-driver-7x44t secrets-store
  ```