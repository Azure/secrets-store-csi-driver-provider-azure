# Contributing with VS Code Remote Container Extension

 We have laid out steps for contributing to the **Azure Key Vault Provider for Secrets Store CSI Driver** using the `VS Code - Remote Container Extension`.

## Prerequisites
1. Azure Subscription

## Fork and Clone Repository

Before we dive into setting up a remote container environment, fork and clone the repository first. Once cloned, enter into the `root` folder of the project:

  ```bash
  $ cd secrets-store-csi-driver-provider-azure
  ```

## VS Code with Remote Container Extension

The [VS Code Remote Container Extension](https://code.visualstudio.com/docs/remote/containers) utilizes the `.devcontainer` folder to build a remote container that will have all necessary dependencies installed to contribute to the **Azure Key Vault Provider for Secrets Store CSI Driver**.

### Dependencies Included Inside The Remote Container

- `yq and jq` command line utilities for manipulating YAML and JSON files
- `Azure CLI` for access to your Azure Subscription
  - Your `.azure` folder on your host machine is mounted into the container, so you will be logged in to the same Azure Subscription.
- `kind` for to allow configuring and using KinD clusters
- `helm` is installed to allow deployment of the Secrets Store CSI Driver and Provider helm charts
- Go 1.15+

### Set Up

1. Open up the project in VS Code.
2. In the bottom-left corner of VS Code click on the remote window icon as shown below:

    ![open a remote window icon](/docs/images/bottom-left.png)

3. Select `Remote-Containers: Reopen in Container` from the drop-down list

    ![Reopen in Container](/docs/images/reopen-container.png)

4. The Azure Key Vault Provider should now be opened inside a Remote Container!
    - In the bottom-left you should see the tag updated to show: `Dev Container: Secrets Store CSI Driver Provider Azure`
    - Open the [integrated terminal](https://code.visualstudio.com/docs/editor/integrated-terminal) with `ctrl + `\`.
    - You can open up a bash shell in the container such as shown below:

    ![remote dev container](/docs/images/container_open.png)

Your Environment is now set up using the VS Code Remote Devcontainer Extension.
