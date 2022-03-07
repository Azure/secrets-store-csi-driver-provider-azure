---
type: docs
title: "Enable Auto Rotation of Secrets"
linkTitle: "Enable Auto Rotation of Secrets"
weight: 2
description: >
  Periodically update the pod mount and Kubernetes Secret with the latest content from external secrets store
---

You can setup the Secrets Store CSI Driver to periodically update the pod volume mount and Kubernetes Secret with the latest content from external secrets-store. Refer to [doc](https://secrets-store-csi-driver.sigs.k8s.io/topics/secret-auto-rotation.html) for steps on enabling auto rotation.

To enable this feature while installing azure provider helm chart, you can use the following helm `--set` flags:
```bash
helm install csi csi-secrets-store-provider-azure/csi-secrets-store-provider-azure --namespace kube-system --set secrets-store-csi-driver.enableSecretRotation=true --set secrets-store-csi-driver.rotationPollInterval=2m
``` 

{{% alert title="NOTE" color="warning" %}}
The CSI driver **does not restart** the application pods. It only handles updating the pod mount and Kubernetes secret similar to how Kubernetes handles updates to Kubernetes secret mounted as volumes.
{{% /alert %}}
