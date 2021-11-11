---
type: docs
title: "Upgrading"
linkTitle: "Upgrading"
weight: 4
description: >
  This document highlights the required actions for upgrading to the latest release
---

## Upgrading to `v1.0.0` release for the driver and provider

> Helm chart version `1.0.0`

The `v1.0.0` version of the Secrets Store CSI Driver and later uses the `v1` API version for `SecretProviderClass` and `SecretProviderClassPodStatus` `CustomResourceDefinition`s. `secrets-store.csi.x-k8s.io/v1alpha1` version of the CRDs will continue to work, but consider updating your YAMLs to use `secrets-store.csi.x-k8s.io/v1`.

## Upgrading to Helm chart version 0.0.20+

{{% alert title="Warning" color="warning" %}}
**tl;dr** - `syncSecret.enabled` is set to false by default for the CSI driver.
{{% /alert %}}

- `syncSecret.enabled` has been set to false by default. This means the RBAC clusterrole and clusterrolebinding required for [sync mounted content as Kubernetes secret](../configurations/sync-with-k8s-secrets) will no longer be created by default as part of `helm install/upgrade`. If you're using the driver to sync mounted content as Kubernetes secret, you'll need to set `secrets-store-csi-driver.syncSecret.enabled=true` as part of `helm install/upgrade`.

If the `secrets-store-csi-driver.syncSecret.enabled=true` isn't explicitly set in `helm install/upgrade` command, it'll result in failure to create Kubernetes secret and the error would be similar to:

```bash
E0610 22:27:02.283100       1 secretproviderclasspodstatus_controller.go:325] "failed to create Kubernetes secret" err="secrets is forbidden: User \"system:serviceaccount:default:secrets-store-csi-driver\" cannot create resource \"secrets\" in API group \"\" in the namespace \"default\"" spc="default/azure-linux" pod="default/busybox-linux-5f479855f7-jvfw4" secret="default/dockerconfig" spcps="default/busybox-linux-5f479855f7-jvfw4-default-azure-linux"
```

## Upgrading to Key Vault provider 0.0.9+

{{% alert title="Warning" color="warning" %}}
**tl;dr** - `0.0.9+` release of the Azure Key Vault provider is incompatible with the Secrets Store CSI Driver versions < `v0.0.14`.
{{% /alert %}}

Prior to `v0.0.14` release of the Secrets Store CSI Driver, the driver communicated with the provider by invoking the provider binary installed on the host. However with `v0.0.14` the driver now introduces a new option to communicate with the provider using gRPC. This feature is enabled by a feature flag in the driver `--grpc-supported-providers=azure`. The `0.0.9` release of the Azure Key Vault provider implements the gRPC server changes and is no longer backward compatible with the Secrets Store CSI Driver versions < `v0.0.14`.

Please carefully read this doc as you upgrade to the latest release of the Azure Key Vault Provider

### If the Secrets Store CSI Driver and Azure Key Vault Provider were installed using helm charts from this [repo](https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/master/charts/csi-secrets-store-provider-azure/README.md)

`helm upgrade` to the latest chart release in the repo will update the Azure Key Vault Provider and Secrets Store CSI Driver to the compatible versions

- This updates the driver version to `v0.0.14+`
- This updates the provider version to `0.0.9+`
- This updates the driver manifest to include the flag `--grpc-supported-providers=azure` to enable communication between driver and provider using gRPC

Run the following commands to confirm the images have been updated -

1. secrets-store container in secrets-store-csi-driver pod is running v0.0.14+

```bash
➜ kubectl get ds -l app=secrets-store-csi-driver -o jsonpath='{range .items[*]}{.spec.template.spec.containers[1].image}{"\n"}'
mcr.microsoft.com/k8s/csi/secrets-store/driver:v0.0.14
```

2. secrets-store container in the secrets-store-csi-driver pod contains the arg `--grpc-supported-providers=azure`

```bash
➜ kubectl get ds -l app=secrets-store-csi-driver -o jsonpath='{range .items[*]}{.spec.template.spec.containers[1].args}{"\n"}'
["--debug=true","--endpoint=$(CSI_ENDPOINT)","--nodeid=$(KUBE_NODE_NAME)","--provider-volume=/etc/kubernetes/secrets-store-csi-providers","--grpc-supported-providers=azure","--metrics-addr=:8080"]
```

3. csi-secrets-store-provider-azure pod is running `0.0.9+`

```bash
➜ kubectl get ds -l app=csi-secrets-store-provider-azure -o jsonpath='{range .items[*]}{.spec.template.spec.containers[0].image}{"\n"}'
mcr.microsoft.com/oss/azure/secrets-store/provider-azure:0.0.9
```

### If the Secrets Store CSI Driver and Azure Key Vault Provider were installed using [deployment yamls](../getting-started/installation/#using-deployment-yamls)

The driver and provider need to be updated one after the other to ensure compatible versions are being run.

1. Update the driver by installing the yamls from [Install the Secrets Store CSI Driver](https://secrets-store-csi-driver.sigs.k8s.io/getting-started/installation.html)
     - **ACTION REQUIRED** If using the yamls from the Secrets Store CSI Driver, add the following flag `--grpc-supported-providers=azure` to the [Linux](https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/master/deploy/secrets-store-csi-driver.yaml) and [Windows](https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/master/deploy/secrets-store-csi-driver-windows.yaml) *daemonset* manifests.
       - The flag needs to be added to the secrets-store container args
     - **ACTION REQUIRED** If using the helm charts from secrets-store-csi-driver, then run `helm upgrade` with `--set grpcSupportedProviders=azure`
2. After the driver is upgraded to the latest version install the latest Azure Key Vault provider by following the [doc](../getting-started/installation/#using-deployment-yamls)
