# Azure Key Vault Provider for Secrets Store CSI Driver

[![Build Status](https://dev.azure.com/AzureContainerUpstream/Secrets%20Store%20CSI%20Driver%20Provider%20Azure/_apis/build/status/csi-secrets-store-provider-azure-nightly?branchName=master)](https://dev.azure.com/AzureContainerUpstream/Secrets%20Store%20CSI%20Driver%20Provider%20Azure/_build/latest?definitionId=370&branchName=master)
[![codecov](https://codecov.io/gh/Azure/secrets-store-csi-driver-provider-azure/branch/master/graph/badge.svg)](https://codecov.io/gh/Azure/secrets-store-csi-driver-provider-azure)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/Azure/secrets-store-csi-driver-provider-azure)
[![Go Report Card](https://goreportcard.com/badge/Azure/secrets-store-csi-driver-provider-azure)](https://goreportcard.com/report/Azure/secrets-store-csi-driver-provider-azure)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/Azure/secrets-store-csi-driver-provider-azure)

Azure Key Vault provider for [Secrets Store CSI Driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver) allows you to get secret contents stored in an [Azure Key Vault](https://docs.microsoft.com/en-us/azure/key-vault/general/overview) instance and use the Secrets Store CSI driver interface to mount them into Kubernetes pods.

## Features

- Mounts secrets/keys/certs to pod using a CSI Inline volume
- Supports mounting multiple secrets store objects as a single volume
- Supports multiple secrets stores as providers. Multiple providers can run in the same cluster simultaneously.
- Supports pod portability with the SecretProviderClass CRD
- Supports Linux and Windows containers
- Supports sync with Kubernetes Secrets
- Supports auto rotation of secrets

## Demo

![Azure Key Vault Provider for Secrets Store CSI Driver Demo](images/demo.gif "Secrets Store CSI Driver Azure Key Vault Provider Demo")

## Getting started

Setup the correct [role assignments and access policies](https://azure.github.io/secrets-store-csi-driver-provider-azure/docs/configurations/identity-access-modes/) and install Azure Keyvault Provider for Secrets Store CSI Driver through [Helm](https://azure.github.io/secrets-store-csi-driver-provider-azure/docs/getting-started/installation/#deployment-using-helm) or [YAML deployment files](https://azure.github.io/secrets-store-csi-driver-provider-azure/docs/getting-started/installation/#using-deployment-yamls). Get familiar with [how to use the Azure Keyvault Provider](https://azure.github.io/secrets-store-csi-driver-provider-azure/docs/getting-started/usage/) and supported [configurations](https://azure.github.io/secrets-store-csi-driver-provider-azure/docs/configurations/).

Try our [walkthrough](https://azure.github.io/secrets-store-csi-driver-provider-azure/docs/demos/standard-walkthrough/) to get a better understanding of the application workflow.

## Contributing

Please refer to [CONTRIBUTING.md](./CONTRIBUTING.md) for more information.

## Code of Conduct

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). For more information, see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Support

Azure Key Vault Provider for Secrets Store CSI Driver is an open source project that is [**not** covered by the Microsoft Azure support policy](https://support.microsoft.com/en-us/help/2941892/support-for-linux-and-open-source-technology-in-azure). [Please search open issues here](https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues), and if your issue isn't already represented please [open a new one](https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/new/choose). The project maintainers will respond to the best of their abilities.
