---
type: docs
title: "Feature Flags"
linkTitle: "Feature Flags"
weight: 1
description: >
  Optional configuration feature flags
---

## Construct PEM Chain Feature Flag

> Available in AKV Provider release `0.0.12+`

> This feature is enabled by default in AKV Provider release `v0.2.0`

The Azure Key Vault provider for Secrets Store CSI Driver by default fetches the chain of certificates from Keyvault and writes to the mount in the same order in which the certificate chain was uploaded. This is an experimental feature that supports reordering of the certificate chain in the following order:

```bash
SERVER
INTERMEDIATE
ROOT
KEY
```

To enable this feature, set `--construct-pem-chain=true` in the provider deployment YAMLs. If using helm to install the driver and provider, set `constructPEMChain: true`.

Refer to [#156](https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/156) for more details.
