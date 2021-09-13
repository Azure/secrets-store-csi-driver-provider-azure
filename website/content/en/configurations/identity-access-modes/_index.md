---
type: docs
title: "Identity Access Modes"
linkTitle: "Identity Access Modes"
weight: 1
description: >
  The Azure Key Vault Provider offers four modes for accessing a Key Vault instance
---

## Best Practices:
Following order of access modes is recommended for Secret Store CSI driver AKV provider:

| Access Option 	| Comment 	|
|---	|---	|
| Pod Identity 	| This is the most secure way to get access to Azure resources (AKV in this case) as it's restricted for a given Pod. 	|
| Managed Identities (System-assigned and User-assigned) 	| AKS using managed identities manages credentials required to connect to Azure Resources. 	|
| Service Principal 	| This is the last option to consider while connecting to AKV as access credentials need to be created as k8s Secret and stored in plain text in etcd.<br>Also, this is the only option to connect to Azure resources from non Azure environment/cluster. 	|