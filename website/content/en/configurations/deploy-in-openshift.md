---
type: docs
title: "Setup Secrets Store CSI Driver on Azure RedHat OpenShift (ARO)"
linkTitle: "Setup Secrets Store CSI Driver on Azure RedHat OpenShift (ARO)"
weight: 6
description: >
  How to setup Azure Keyvault Provider for Secrets Store CSI Driver on Azure RedHat OpenShift (ARO) 
---

### Installation

1. Install the Azure Keyvault provider for Secrets Store CSI Driver on Azure RedHat OpenShift run:

    ```bash
    helm repo add csi-secrets-store-provider-azure https://azure.github.io/secrets-store-csi-driver-provider-azure/charts
    helm install csi csi-secrets-store-provider-azure/csi-secrets-store-provider-azure --set linux.privileged=true
    ```

    Setting `linux.privileged=true` with `helm install` will enable privileged mode for the Linux *daemonset* pods.

    ```yml
        securityContext:
          privileged: true
    ```

    This is required for the AKV provider pods to successfully startup in ARO.

1. Bind SecurityContextConstraints (SCC) to the Secrets Store CSI Driver and Azure Keyvault Provider service accounts

    ```bash
    # Replace $target_namespace with the namespace used for helm install
    # Secrets Store CSI Driver service account
    oc adm policy add-scc-to-user privileged system:serviceaccount:$target_namespace:secrets-store-csi-driver
    # Azure Keyvault Provider service account
    oc adm policy add-scc-to-user privileged system:serviceaccount:$target_namespace:csi-secrets-store-provider-azure
    ```

### Uninstall

1. Run the following command to uninstall

    ```bash
    helm delete <release name>
    ```

1. Remove the SCC bindings

    ```bash
    # Replace $target_namespace with the namespace used for helm install
    oc adm policy remove-scc-to-user privileged system:serviceaccount:$target_namespace:secrets-store-csi-driver
    oc adm policy remove-scc-to-user privileged system:serviceaccount:$target_namespace:csi-secrets-store-provider-azure
    ```
