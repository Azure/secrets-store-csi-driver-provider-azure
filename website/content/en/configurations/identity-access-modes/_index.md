---
type: docs
title: "Identity Access Modes"
linkTitle: "Identity Access Modes"
weight: 1
description: >
  The Azure Key Vault Provider offers five modes for accessing a Key Vault instance
---

## Best Practices

Following order of access modes is recommended for Secret Store CSI driver AKV provider:

| Access Option                                          | Comment                                                                                                                                                                                                                                                                    |
| ------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Identity Binding [**RECOMMENDED for AKS**]             | Uses [AKS Identity Binding](https://learn.microsoft.com/azure/aks/identity-bindings-concepts) to access Key Vault. Only requires a single FIC on the managed identity regardless of how many clusters or workloads use it, eliminating workload identity's 20 FIC-per-identity limit. AKS only. |
| Workload Identity [**RECOMMENDED**]                    | Access Key Vault using [Workload Identity Federation](https://docs.microsoft.com/en-us/azure/active-directory/develop/workload-identity-federation). Works on any Kubernetes cluster with an OIDC issuer.                                                                   |
| Pod Identity [**DEPRECATED**]                          | [AAD Pod Identity](https://github.com/Azure/aad-pod-identity) has been [DEPRECATED](https://github.com/Azure/aad-pod-identity#-announcement).<br>This provides a way to get access to Azure resources (AKV in this case) using the managed identity bound to the Pod.</br> |
| Managed Identities (System-assigned and User-assigned) | Managed identities eliminate the need for developers to manage credentials. Managed identities provide an identity for applications to use when connecting to Azure Keyvault.                                                                                              |
| Service Principal                                      | This is the last option to consider while connecting to AKV as access credentials need to be created as Kubernetes Secret and stored in plain text in etcd.                                                                                                                |
