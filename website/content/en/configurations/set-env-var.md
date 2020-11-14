---
type: docs
title: "Set your Environment Variable to Reference a Kubernetes Secret"
linkTitle: "Set your Environment Variable to Reference a Kubernetes Secret"
weight: 2
description: >
  Configure your environment variable to reference a Kubernetes Secret
---

Once the secret is created, you may wish to set an ENV VAR in your deployment to reference the new Kubernetes secret.

```yaml
spec:
  containers:
  - image: nginx
    name: nginx
    env:
    - name: SECRET_USERNAME
      valueFrom:
        secretKeyRef:
          name: foosecret
          key: username
```
Here is a sample [deployment yaml](https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/master/test/bats/tests/azure/nginx-deployment-synck8s-azure.yaml) that creates an ENV VAR from the synced Kubernetes secret.