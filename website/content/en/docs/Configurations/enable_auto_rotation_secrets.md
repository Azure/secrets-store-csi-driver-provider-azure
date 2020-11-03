---
title: "Enable Auto Rotation of Secrets"
linkTitle: "Enable Auto Rotation of Secrets"
weight: 2
description: >
  Periodically update the pod mount and Kubernetes Secret with the latest content from external secrets store
---

You can setup the Secrets Store CSI Driver to periodically update the pod mount and Kubernetes Secret with the latest content from external secrets-store. Refer to [doc](https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/master/docs/README.rotation.md) for steps on enabling auto rotation.

**NOTE** The CSI driver **does not restart** the application pods. It only handles updating the pod mount and Kubernetes secret similar to how Kubernetes handles updates to Kubernetes secret mounted as volumes.