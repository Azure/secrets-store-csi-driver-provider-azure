# Contribution Guide
 We have laid out 2 ways to get set up for contributing to the **Secrets Store CSI Driver Provider for Azure**. The first way is through your own local dev environment and your editor/IDE of choice. The second way is through the `VS Code - Remote Container Extension`.




## Prerequisites
1. Azure Subscription


### Table of Contents
- [Fork and Clone Repository](#fork-and-clone-repository)
- [OPTION 1 - Local Dev Environment](#option-1---local-dev-environment)
- [OPTION 2 - VS Code with Remote Container Extension](#option-2---vs-code-with-remote-container-Extension)


## Fork and Clone Repository

Before we dive into setting up a local or remote environment, fork and clone the repository first. Once cloned, enter into the `root` folder of the project:

  ```bash
  $ cd secrets-store-csi-driver-provider-azure
  ```

## OPTION 1 - Local Dev Environment

### Prerequisites
In addition to the [Prerequisites](#prerequisites) listed at the beginning of the doc, you will need:
1. Go 1.14
2. `jq` and `yq` dev tools installed on your system.
    - Find [yq here](https://github.com/mikefarah/yq)
    - Find [jq here](https://stedolan.github.io/jq/download/)

### Set Up

Once you are inside the `root` directory of the project, you can build the `secrets-store-csi-driver-provider-azure` binary with:

```bash
make build
```

## OPTION 2 - VS Code with Remote Container Extension
The [VS Code Remote Container Extension](LINKHERE)- link utilizes the `.devcontainer` folder to build a remote container that will have all necessary dependencies installed to contribute to the **Secrets Store CSI Driver Provider for Azure**.

### Dependencies Included Inside The Remote Container
- `yq and jq` command line utilities for manipulating YAML and JSON files
- `Azure CLI` for access to your Azure Subscription
  - Your `.azure` folder on your host machine is mounted into the container, so you will be logged in to the same Azure Subscription.
- Go 1.14
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

    ![remote dev cointainer](/docs/images/container_open.png)

Your Environment is now set up. You can now contribute, run, and debug the **Secrets Store CSI Driver Azure Provider**. If you want to learn more about the VS Code Remote Container Extension, read the [Debug with VS Code Guidance](/docs/debug-with-vscode.md)



