---
type: docs
title: "Troubleshooting"
linkTitle: "Troubleshooting"
weight: 4
description: >
  An overview of a list of components to assist in troubleshooting.
---

## Logging

Below is a list of commands you can use to view relevant logs of Azure Key Vault provider and Secrets Store CSI Driver.

### Isolate errors from logs

You can use `grep ^E` and `--since` flag from `kubectl` to isolate any errors occurred after a given duration.

#### For Azure Key Vault provider versions less than 0.0.9

To troubleshoot issues with the csi driver and the provider, you can look at logs from the `secrets-store` container of the csi driver pod running on the same node as your application pod:

```bash
# find the secrets store csi driver pod running on the same node as your application pod
kubectl get pods -l app=secrets-store-csi-driver -o wide
kubectl logs <driver pod name> secrets-store --since=1h | grep ^E
```

#### For Azure Key Vault provider versions equal to and greater than 0.0.9

For `0.0.9+` the provider logs are available in the provider pods. To troubleshoot issues with the provider, you can look at logs from the provider pod running on the same node as your application pod

```bash
# find the csi-secrets-store-provider-azure pod running on the same node as your application pod
kubectl get pods -l app=csi-secrets-store-provider-azure -o wide
kubectl logs <provider pod name> --since=1h | grep ^E
```

#### For CSI driver logs

```bash
# find the secrets store csi driver pod running on the same node as your application pod
kubectl get pods -l app=secrets-store-csi-driver -o wide
kubectl logs <driver pod name> secrets-store --since=1h | grep ^E
```

> It is always a good idea to include relevant logs from Azure Key Vault provider and Secrets Store CSI Driver when opening a new issue.

## Common Issues

Common issues or questions that users have run into when using Azure Key Vault provider for Secrets Store CSI Driver are detailed below.

### driver name `secrets-store.csi.k8s.io` not found in the list of registered CSI drivers

If you received the following error message in the pod events:

```bash
Warning FailedMount 42s (x12 over 8m56s) kubelet, akswin000000 MountVolume.SetUp failed for volume "secrets-store01-inline" : kubernetes.io/csi: mounter.SetUpAt failed to get CSI client: driver name secrets-store.csi.k8s.io not found in the list of registered CSI drivers
```

It means the Secrets Store CSI Driver pods aren't running on the node where application is running.

- If you've installed the AKV provider using deployment manifests, then make sure to follow the [instructions](../getting-started/installation) to install the Secrets Store CSI Driver. 
- If you've already deployed the Secrets Store CSI Driver, then check if the node is tainted. If node is tainted, then redeploy the Secrets Store CSI Driver and Azure Key Vault provider by adding toleration for the taints.
- If your application is running on windows node, then make sure to install the Secrets Store CSI Driver and Azure Key Vault provider on windows nodes by using the helm configuration values.

Past issues:

- https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/213
- https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/346

### failed to get key vault token: nmi response failed with status code: 404

If you received the following error message in the logs/events:

```bash
  Warning  FailedMount  74s    kubelet            MountVolume.SetUp failed for volume "secrets-store-inline" : kubernetes.io/csi: mounter.SetupAt failed: rpc error: code = Unknown desc = failed to mount secrets store objects for pod default/test, err: rpc error: code = Unknown desc = failed to mount objects, error: failed to get keyvault client: failed to get key vault token: nmi response failed with status code: 404, err: <nil>
```

It means the NMI component in aad-pod-identity returned an error for token request. To get more details on the error, check the MIC pod logs and refer to the AAD Pod Identity [troubleshooting guide](https://azure.github.io/aad-pod-identity/docs/troubleshooting/) to resolve the issue.

Past issues:

- https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/119
- https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/200
- https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/352

### failed to find provider binary azure, err: stat /etc/kubernetes/secrets-store-csi-providers/azure/provider-azure: no such file or directory

If you received the following error message in the logs/events:

```bash
Warning FailedMount 85s (x10 over 5m35s) kubelet, aks-default-28951543-vmss000000 MountVolume.SetUp failed for volume "secrets-store01-inline" : kubernetes.io/csi: mounter.SetupAt failed: rpc error: code = Unknown desc = failed to mount secrets store objects for pod default/nginx-secrets-store-inline-user-msi, err: failed to find provider binary azure, err: stat /etc/kubernetes/secrets-store-csi-providers/azure/provider-azure: no such file or directory
```

It means the driver is unable to communicate with the provider.

- If you're installing provider version < 0.0.9, check if the provider pods are running on all nodes.
- If you're installing provider version >= 0.0.9, follow the [Installation steps](../getting-started/installation/#using-deployment-yamls) to configure the driver to use grpc for communication with the provider.

Past issues:

- https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/254
- https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/259
- https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/269
- https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/303

### keyvault.BaseClient#GetSecret: Failure sending request: StatusCode=0 -- Original Error: context canceled"

If you received the following error message in the provider logs:

```bash
E1029 17:37:42.461313       1 server.go:54] failed to process mount request, error: keyvault.BaseClient#GetSecret: Failure sending request: StatusCode=0 -- Original Error: context deadline exceeded
```

It means the provider pod is unable to access the keyvault instance because

1. There is a firewall rule blocking egress traffic from the provider.
2. Network policies configured in the cluster that's blocking egress traffic.

The provider pods run on `hostNetwork`. So if there is a policy blocking this traffic or there are network jitters on the node it could result in the above failure. Check for policies configured to block traffic and whitelist the provider pods. Also, ensure there is connectivity to AAD and Keyvault from the node.

Past issues:

- https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/292
- https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/471
