---
type: docs
title: "For Azure Key Vault Provider version `0.0.9+`"
linkTitle: "For Azure Key Vault Provider version `0.0.9+`"
weight: 2
description: >
  For versions equal to and greater than 0.0.9
---

For `0.0.9+` the provider logs are available in the provider pods. To troubleshoot issues with the provider, you can look at logs from the provider pod running on the same node as your application pod

  ```bash
  kubectl get pod -o wide
  # find the csi-secrets-store-provider-azure pod running on the same node as your application pod

  kubectl logs csi-csi-secrets-store-provider-azure-lmx6p
  ```