---
type: docs
title: "Identity Access Modes"
linkTitle: "Identity Access Modes"
weight: 1
description: >
  The Azure Key Vault Provider offers four modes for accessing a Key Vault instance
---

## Best Practices

Following order of access modes is recommended for Secret Store CSI driver AKV provider:

| Access Option                                          | Comment                                                                                                                                                                                                        |
| ------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Workload Identity (Preview) [**RECOMMENDED**]          | This is currently in preview. It's a secure way to access Key Vault based on the [Workload Identity Federation](https://docs.microsoft.com/en-us/azure/active-directory/develop/workload-identity-federation). |
| Pod Identity                                           | This is the most secure way to get access to Azure resources (AKV in this case) as it uses the managed identity bound to the Pod.                                                                              |
| Managed Identities (System-assigned and User-assigned) | Managed identities eliminate the need for developers to manage credentials. Managed identities provide an identity for applications to use when connecting to Azure Keyvault.                                  |
| Service Principal                                      | This is the last option to consider while connecting to AKV as access credentials need to be created as Kubernetes Secret and stored in plain text in etcd.                                                    |
